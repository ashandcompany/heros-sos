package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// Worker consomme la file de mails de RabbitMQ et les remet à un Mailer (SMTP). C'est la seconde moitié de
// Publisher : l'API dépose, le worker envoie, dans un autre processus (commande mailworker).
//
// Garanties :
//   - un mail n'est retiré de la file (acquitté) qu'une fois envoyé : si le worker s'arrête brutalement, le
//     courtier le redonne à un autre envoi. Au pire, un mail est envoyé deux fois (jamais zéro) ;
//   - un échec est retenté MaxAttempts fois avec un délai croissant, comme dans Queue ; au-delà, ou pour un
//     message illisible, le mail part en file morte et n'est plus rejoué tout seul ;
//   - à l'arrêt, les envois en cours ont SendTimeout pour finir ; ceux qui n'ont pas fini reviennent dans la
//     file.
//
// Au plus cfg.Workers mails sont envoyés en même temps : c'est aussi le nombre de messages que le courtier
// confie au worker d'avance (prefetch), les autres restent dans la file, visibles et durables.
type Worker struct {
	transport Mailer
	cfg       config.Mail
	topology  topology
	logger    *slog.Logger
}

// NewWorker crée un worker qui envoie les mails de la file avec transport.
func NewWorker(transport Mailer, cfg config.Mail, logger *slog.Logger) *Worker {
	return newWorker(transport, cfg, defaultTopology(), logger)
}

func newWorker(transport Mailer, cfg config.Mail, topo topology, logger *slog.Logger) *Worker {
	return &Worker{transport: transport, cfg: cfg, topology: topo, logger: logger}
}

// Run consomme la file jusqu'à l'annulation de ctx, puis retourne nil après avoir laissé les envois en cours
// se terminer. Une coupure de RabbitMQ n'arrête pas le worker : il se reconnecte tous les ReconnectDelay.
func (w *Worker) Run(ctx context.Context) error {
	down := false
	for ctx.Err() == nil {
		down = w.cycle(ctx, down)
	}
	return nil
}

// cycle tient une connexion jusqu'à sa fin, puis attend ReconnectDelay avant de rendre la main. down dit si la panne en
// cours a déjà été signalée ; le résultat est le même indicateur, à jour.
func (w *Worker) cycle(ctx context.Context, down bool) bool {
	connected := false
	err := w.consume(ctx, func() { connected = true })
	if ctx.Err() != nil {
		return down
	}
	if connected {
		down = false
	}
	// Comme pour Publisher : seule la première tentative d'une panne est visible.
	level := slog.LevelDebug
	if !down {
		level, down = slog.LevelWarn, true
	}
	w.logger.Log(ctx, level, "consommation de la file de mails interrompue", "error", err, "retry_in", w.cfg.RabbitMQ.ReconnectDelay)

	timer := time.NewTimer(w.cfg.RabbitMQ.ReconnectDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
	return down
}

// consume tient une connexion : il se connecte, consomme, et retourne quand la connexion se perd, que le
// consommateur est annulé par le courtier, ou que ctx est annulé (erreur alors ignorée par Run).
func (w *Worker) consume(ctx context.Context, onConnected func()) error {
	conn, err := dialBroker(w.cfg.RabbitMQ, "mailworker")
	if err != nil {
		return err
	}
	defer func() { _ = conn.CloseDeadline(time.Now().Add(w.cfg.RabbitMQ.ConfirmTimeout)) }()

	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := w.topology.declare(channel); err != nil {
		return err
	}
	if err := channel.Qos(w.cfg.Workers, 0, false); err != nil {
		return fmt.Errorf("mail : prefetch : %w", err)
	}
	connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	tag := "mailworker-" + ids.New().String()
	deliveries, err := channel.Consume(w.topology.queue, tag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("mail : consommation de %s : %w", w.topology.queue, err)
	}
	w.logger.Info("worker de mails connecté", "queue", w.topology.queue, "concurrency", w.cfg.Workers)
	onConnected()

	// abort interrompt les envois qui n'ont pas fini dans le délai de grâce de l'arrêt. Il ne dérive pas de ctx :
	// l'arrêt demandé ne doit pas couper les envois en cours, seulement empêcher d'en commencer d'autres.
	abort, stopSending := context.WithCancel(context.WithoutCancel(ctx))
	defer stopSending()

	var handlers sync.WaitGroup
	drained := make(chan struct{})
	for range w.cfg.Workers {
		handlers.Add(1)
		go func() {
			defer handlers.Done()
			for delivery := range deliveries {
				if ctx.Err() != nil {
					// Arrêt demandé entre la livraison et le traitement : rendre le mail sans l'envoyer.
					w.settle("remise en file", delivery.Nack(false, true))
					continue
				}
				w.handle(abort, delivery)
			}
		}()
	}
	go func() {
		handlers.Wait()
		close(drained)
	}()

	select {
	case <-ctx.Done():
		// Plus de nouveaux mails ; les envois en cours ont SendTimeout pour finir.
		if err := channel.Cancel(tag, false); err != nil {
			w.logger.Warn("annulation du consommateur impossible", "error", err)
		}
		grace := time.NewTimer(w.cfg.SendTimeout)
		defer grace.Stop()
		select {
		case <-drained:
		case <-grace.C:
			w.logger.Warn("des envois ne sont pas terminés à l'arrêt : ils sont interrompus et reviennent dans la file")
			stopSending()
			<-drained
		}
		return ctx.Err()
	case err := <-connClosed:
		stopSending()
		<-drained
		return sessionLost("connexion", err)
	case <-drained:
		return errors.New("mail : le courtier a mis fin à la consommation")
	}
}

// handle envoie un mail puis le règle auprès du courtier : acquitté s'il est parti, mis en file morte s'il ne
// partira jamais, remis en file si l'arrêt a interrompu l'envoi.
func (w *Worker) handle(abort context.Context, delivery amqp.Delivery) {
	message, err := decodeMessage(delivery.Body)
	if err != nil {
		w.logger.Error("mail illisible : envoyé en file morte", "message_id", delivery.MessageId, "error", err)
		w.settle("file morte", delivery.Reject(false))
		return
	}

	err = sendWithRetry(abort, w.transport, w.cfg, w.logger, message)
	switch {
	case err == nil:
		w.settle("acquittement", delivery.Ack(false))
	case abort.Err() != nil:
		w.settle("remise en file", delivery.Nack(false, true))
	default:
		w.logger.Error("mail envoyé en file morte", "message_id", delivery.MessageId, "queue", w.topology.deadQueue)
		w.settle("file morte", delivery.Reject(false))
	}
}

// settle journalise l'échec d'un acquittement : la connexion est perdue, et le courtier redonnera le mail
// (envoi en double possible, perte impossible).
func (w *Worker) settle(what string, err error) {
	if err != nil {
		w.logger.Warn("le courtier n'a pas reçu le règlement d'un mail : il sera redonné", "action", what, "error", err)
	}
}

// CheckWorker interroge RabbitMQ comme le ferait un contrôle de santé : la file existe et au moins un
// consommateur y est attaché. C'est ce que lance `mailworker -healthcheck`, dans une image sans shell.
func CheckWorker(cfg config.RabbitMQ) error {
	return checkWorker(cfg, defaultTopology())
}

func checkWorker(cfg config.RabbitMQ, topo topology) error {
	conn, err := dialBroker(cfg, "mailworker-healthcheck")
	if err != nil {
		return err
	}
	defer func() { _ = conn.CloseDeadline(time.Now().Add(cfg.ConfirmTimeout)) }()
	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	queue, err := channel.QueueDeclarePassive(topo.queue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("mail : file %s : %w", topo.queue, err)
	}
	if queue.Consumers == 0 {
		return fmt.Errorf("mail : aucun worker n'est attaché à la file %s", topo.queue)
	}
	return nil
}
