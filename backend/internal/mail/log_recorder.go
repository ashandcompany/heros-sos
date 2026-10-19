package mail

import (
	"context"
	"log/slog"
	"sync"
)

// LogMailer écrit les mails dans les logs au lieu de les envoyer. Réservé au développement : les
// liens d'activation et de réinitialisation y figurent en clair (la production l'interdit).
type LogMailer struct {
	logger *slog.Logger
}

// NewLogMailer crée un expéditeur qui journalise.
func NewLogMailer(logger *slog.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// Send journalise le mail.
func (m *LogMailer) Send(ctx context.Context, message Message) error {
	if _, err := message.validate(); err != nil {
		return err
	}
	m.logger.InfoContext(ctx, "mail (non envoyé, driver log)", "to", message.To, "subject", message.Subject, "text", message.Text)
	return nil
}

// Recorder mémorise les mails « envoyés », pour les tests. Il est sûr pour un usage concurrent.
type Recorder struct {
	mu       sync.Mutex
	messages []Message
	// Err, s'il est renseigné, est retourné par Send (simulation de panne).
	err error
}

// NewRecorder crée un enregistreur vide.
func NewRecorder() *Recorder { return &Recorder{} }

// FailWith fait échouer les envois suivants avec err (nil pour rétablir le fonctionnement).
func (r *Recorder) FailWith(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// Send mémorise le message.
func (r *Recorder) Send(_ context.Context, message Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.messages = append(r.messages, message)
	return nil
}

// Messages retourne une copie des mails reçus.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.messages...)
}

// Last retourne le dernier mail reçu.
func (r *Recorder) Last() (Message, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.messages) == 0 {
		return Message{}, false
	}
	return r.messages[len(r.messages)-1], true
}
