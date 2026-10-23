package mail

import (
	"fmt"
	"log/slog"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// NewTransport crée l'expéditeur qui remet réellement les mails : le serveur SMTP, ou le journal en
// développement. Il est partagé par l'API (file en mémoire) et par le worker (file RabbitMQ).
func NewTransport(cfg config.Mail, logger *slog.Logger) (Mailer, error) {
	switch cfg.Driver {
	case config.MailDriverSMTP:
		return NewSMTPMailer(cfg.SMTP, Sender{Address: cfg.FromAddress, Name: cfg.FromName}, clock.System{}), nil
	case config.MailDriverLog:
		return NewLogMailer(logger), nil
	default:
		return nil, fmt.Errorf("mail : driver de mail inconnu %q", cfg.Driver)
	}
}
