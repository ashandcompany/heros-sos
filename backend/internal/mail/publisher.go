package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// publisherAppID identifie l'API comme émettrice dans les propriétés des messages.
const publisherAppID = "heros-sos-api"

// Publisher dépose les mails dans RabbitMQ au lieu de les envoyer : c'est un Mailer, comme Queue, et ses
// appelants ne voient aucune différence. Un worker (Worker, commande mailworker) les envoie ensuite.
//
// Send ne retourne nil qu'une fois que RabbitMQ a confirmé avoir écrit le message sur disque (confirmation du
// dépôt) : un mail accepté survit à un arrêt brutal de l'API comme du courtier. Quand le courtier est
// injoignable ou bloqué (alarme disque ou mémoire), Send échoue vite avec ErrQueueFull plutôt que de faire
// attendre la requête : l'API reste disponible sans RabbitMQ, et un envoi groupé (newsletter) patiente et
// réessaie. Une attente de confirmation expirée peut avoir réussi côté courtier : le réessai risque alors un
// doublon, jamais une perte (livraison « au moins une fois »).
//
// La connexion vit dans une goroutine qui appartient au Publisher : elle se reconnecte seule après une
// coupure et s'arrête dans Close.
type Publisher struct {
	cfg      config.Mail
	topology topology
	logger   *slog.Logger

	mu       sync.RWMutex // protège closed et session
	closed   bool
	session  *publishSession // nil tant que le courtier n'est pas joint
	inflight sync.WaitGroup  // dépôts en cours, que Close attend

	// blocked est vrai quand le courtier a suspendu les dépôts de cette connexion (alarme de ressources).
	blocked atomic.Bool

	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{} // fermé quand la goroutine de connexion s'est arrêtée
}

// publishSession est une connexion ouverte, son canal en mode confirmation et ce que le courtier signale.
type publishSession struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	connClosed <-chan *amqp.Error
	chanClosed <-chan *amqp.Error
	blocks     <-chan amqp.Blocking
	returns    <-chan amqp.Return
}

// NewPublisher démarre la connexion à RabbitMQ en arrière-plan et retourne tout de suite : l'API démarre
// même si le courtier n'est pas encore là.
func NewPublisher(cfg config.Mail, logger *slog.Logger) *Publisher {
	return newPublisher(cfg, defaultTopology(), logger)
}

func newPublisher(cfg config.Mail, topo topology, logger *slog.Logger) *Publisher {
	p := &Publisher{
		cfg: cfg, topology: topo, logger: logger,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go p.run()
	return p
}

// Send dépose le mail dans la file. Il retourne ErrQueueFull (enveloppée) si le courtier ne l'a pas
// accepté, ErrQueueClosed après Close, ErrInvalidMessage pour un mail mal formé.
func (p *Publisher) Send(ctx context.Context, message Message) error {
	if _, err := message.validate(); err != nil {
		return err
	}
	body, err := encodeMessage(message)
	if err != nil {
		return fmt.Errorf("mail : encodage du message : %w", err)
	}

	session, err := p.acquire()
	if err != nil {
		return err
	}
	defer p.inflight.Done()

	publishCtx, cancel := context.WithTimeout(ctx, p.cfg.RabbitMQ.ConfirmTimeout)
	defer cancel()
	confirmation, err := session.channel.PublishWithDeferredConfirmWithContext(
		publishCtx, p.topology.exchange, routingSend,
		// mandatory : un message qui n'atteint aucune file revient (voir watch) au lieu de disparaître.
		true, false,
		amqp.Publishing{
			ContentType:  contentTypeMail,
			DeliveryMode: deliveryPersistent,
			MessageId:    ids.New().String(),
			AppId:        publisherAppID,
			Body:         body,
		},
	)
	if err != nil {
		return publishFailure(ctx, "dépôt impossible", err)
	}
	acked, err := confirmation.WaitContext(publishCtx)
	if err != nil {
		return publishFailure(ctx, "RabbitMQ n'a pas confirmé le dépôt", err)
	}
	if !acked {
		return queueUnavailable("RabbitMQ a refusé le dépôt")
	}
	return nil
}

// acquire vérifie que le dépôt est possible et le déclare en cours : Close l'attendra.
func (p *Publisher) acquire() (*publishSession, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	switch {
	case p.closed:
		return nil, ErrQueueClosed
	case p.session == nil:
		return nil, queueUnavailable("RabbitMQ injoignable")
	case p.blocked.Load():
		return nil, queueUnavailable("RabbitMQ a suspendu les dépôts (disque ou mémoire saturé)")
	}
	p.inflight.Add(1)
	return p.session, nil
}

// publishFailure distingue l'annulation voulue par l'appelant (qui ne concerne pas le courtier) d'un dépôt
// qui n'a pas abouti.
func publishFailure(ctx context.Context, what string, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return queueUnavailable(what + " : " + err.Error())
}

func queueUnavailable(reason string) error {
	return fmt.Errorf("%w : %s", ErrQueueFull, reason)
}

// Close refuse les nouveaux mails, attend la confirmation de ceux qui sont en cours de dépôt, puis ferme la
// connexion. Si ctx expire avant, la connexion est fermée de force et ctx.Err() est retourné.
func (p *Publisher) Close(ctx context.Context) error {
	p.mu.Lock()
	first := !p.closed
	p.closed = true
	p.mu.Unlock()

	if first {
		go func() {
			p.inflight.Wait()
			p.halt()
		}()
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.halt()
		<-p.done
		return ctx.Err()
	}
}

func (p *Publisher) halt() { p.stopOnce.Do(func() { close(p.stop) }) }

// run garde une connexion ouverte jusqu'à halt : il se connecte, surveille la session, et recommence
// après ReconnectDelay quand elle se perd.
func (p *Publisher) run() {
	defer close(p.done)
	down := false
	for {
		session, err := p.open()
		if err == nil {
			p.setSession(session)
			p.logger.Info("RabbitMQ connecté : les mails sont déposés dans la file", "queue", p.topology.queue)
			down = false
			err = p.watch(session)
			p.setSession(nil)
			p.blocked.Store(false)
			session.close(p.cfg.RabbitMQ.ConfirmTimeout)
		}

		select {
		case <-p.stop:
			return
		default:
		}
		// Une panne prolongée ne doit pas écrire une ligne toutes les ReconnectDelay : seule la première compte.
		level := slog.LevelDebug
		if !down {
			level, down = slog.LevelWarn, true
		}
		p.logger.Log(context.Background(), level, "RabbitMQ indisponible : les mails sont refusés jusqu'au retour du courtier",
			"error", err, "retry_in", p.cfg.RabbitMQ.ReconnectDelay)

		timer := time.NewTimer(p.cfg.RabbitMQ.ReconnectDelay)
		select {
		case <-timer.C:
		case <-p.stop:
			timer.Stop()
			return
		}
	}
}

func (p *Publisher) setSession(session *publishSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.session = session
}

// open se connecte, déclare la topologie et passe le canal en mode confirmation.
func (p *Publisher) open() (*publishSession, error) {
	conn, err := dialBroker(p.cfg.RabbitMQ, "api")
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err == nil {
		err = p.topology.declare(channel)
	}
	if err == nil {
		err = channel.Confirm(false)
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	// Les écoutes sont posées avant tout dépôt : la bibliothèque bloque si une notification n'est pas lue.
	return &publishSession{
		conn: conn, channel: channel,
		connClosed: conn.NotifyClose(make(chan *amqp.Error, 1)),
		chanClosed: channel.NotifyClose(make(chan *amqp.Error, 1)),
		blocks:     conn.NotifyBlocked(make(chan amqp.Blocking, 4)),
		returns:    channel.NotifyReturn(make(chan amqp.Return, 4)),
	}, nil
}

func (s *publishSession) close(timeout time.Duration) {
	_ = s.channel.Close()
	_ = s.conn.CloseDeadline(time.Now().Add(timeout))
}

// errUnroutable signale un mail que le courtier n'a rangé dans aucune file.
var errUnroutable = errors.New("mail : un message n'a atteint aucune file")

// watch suit la session jusqu'à sa fin et retourne pourquoi elle s'est terminée (nil si l'arrêt est voulu).
func (p *Publisher) watch(s *publishSession) error {
	blocks, returns := s.blocks, s.returns
	for {
		select {
		case <-p.stop:
			return nil
		case err := <-s.connClosed:
			return sessionLost("connexion", err)
		case err := <-s.chanClosed:
			return sessionLost("canal", err)
		case block, ok := <-blocks:
			if !ok {
				blocks = nil
				continue
			}
			p.blocked.Store(block.Active)
			if block.Active {
				p.logger.Warn("RabbitMQ suspend les dépôts : les mails sont refusés jusqu'à la levée de l'alarme", "reason", block.Reason)
			} else {
				p.logger.Info("RabbitMQ accepte de nouveau les dépôts")
			}
		case returned, ok := <-returns:
			if !ok {
				returns = nil
				continue
			}
			// La file a disparu (supprimée à la main ?) : le mail est perdu, et les suivants le seraient aussi.
			// Se reconnecter redéclare la topologie.
			p.logger.Error("un mail n'a atteint aucune file : la topologie RabbitMQ va être redéclarée",
				"exchange", returned.Exchange, "routing_key", returned.RoutingKey, "message_id", returned.MessageId)
			return errUnroutable
		}
	}
}

// sessionLost décrit la fin d'une connexion ou d'un canal ; err est nil quand la fermeture est voulue.
func sessionLost(what string, err *amqp.Error) error {
	if err == nil {
		return fmt.Errorf("%s fermée", what)
	}
	return fmt.Errorf("%s perdue : %w", what, err)
}
