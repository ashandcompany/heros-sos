package mail

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

var (
	// ErrQueueFull indique que la file d'envoi est saturée : le mail n'a pas été accepté.
	ErrQueueFull = errors.New("mail : file d'envoi saturée")
	// ErrQueueClosed indique que la file ne reçoit plus de mails (arrêt en cours).
	ErrQueueClosed = errors.New("mail : file d'envoi fermée")
)

// Queue envoie les mails en arrière-plan : une requête HTTP n'attend jamais le serveur SMTP.
// Des goroutines travailleuses consomment une file bornée, réessaient les échecs avec un délai
// croissant, et se vident proprement à l'arrêt. Queue est un Mailer : Send ne fait que mettre
// le mail en file.
//
// Les mails en file sont perdus si le processus s'arrête brutalement ; RabbitMQ (ou une table
// d'envoi) les rendrait durables, sans changer les appelants qui ne connaissent que Mailer.
type Queue struct {
	mailer Mailer
	cfg    config.Mail
	logger *slog.Logger

	mu     sync.RWMutex // protège closed et l'envoi sur jobs
	closed bool
	jobs   chan Message

	workers sync.WaitGroup
	abort   context.CancelFunc
	baseCtx context.Context
}

// NewQueue démarre cfg.Workers goroutines d'envoi.
func NewQueue(mailer Mailer, cfg config.Mail, logger *slog.Logger) *Queue {
	baseCtx, abort := context.WithCancel(context.Background())
	queue := &Queue{
		mailer:  mailer,
		cfg:     cfg,
		logger:  logger,
		jobs:    make(chan Message, cfg.QueueSize),
		abort:   abort,
		baseCtx: baseCtx,
	}
	for range cfg.Workers {
		queue.workers.Add(1)
		go queue.work()
	}
	return queue
}

// Send met le mail en file sans attendre son envoi. Il échoue si la file est saturée ou fermée.
func (q *Queue) Send(_ context.Context, message Message) error {
	if _, err := message.validate(); err != nil {
		return err
	}

	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return ErrQueueClosed
	}
	select {
	case q.jobs <- message:
		return nil
	default:
		return ErrQueueFull
	}
}

// Close arrête d'accepter des mails, laisse les travailleurs vider la file, et attend leur fin
// dans la limite du contexte. Si le délai expire, les envois en cours sont interrompus et les
// mails restants sont abandonnés (et comptés dans l'erreur retournée).
func (q *Queue) Close(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		q.abort()
		<-done
		return ctx.Err()
	}
}

func (q *Queue) work() {
	defer q.workers.Done()
	for message := range q.jobs {
		q.deliver(message)
	}
}

// deliver tente l'envoi jusqu'à MaxAttempts fois, avec un délai qui s'allonge à chaque échec.
// Ni le destinataire ni le contenu ne sont journalisés : ce sont des données personnelles.
func (q *Queue) deliver(message Message) {
	for attempt := 1; attempt <= q.cfg.MaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(q.baseCtx, q.cfg.SendTimeout)
		err := q.mailer.Send(ctx, message)
		cancel()
		if err == nil {
			return
		}

		last := attempt == q.cfg.MaxAttempts || q.baseCtx.Err() != nil || errors.Is(err, ErrInvalidMessage)
		if last {
			q.logger.Error("mail abandonné", "subject", message.Subject, "attempts", attempt, "error", err)
			return
		}
		q.logger.Warn("échec d'envoi de mail, nouvelle tentative", "subject", message.Subject, "attempt", attempt, "error", err)

		timer := time.NewTimer(q.cfg.RetryDelay * time.Duration(attempt))
		select {
		case <-timer.C:
		case <-q.baseCtx.Done():
			timer.Stop()
			return
		}
	}
}
