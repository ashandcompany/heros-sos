package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// SMTPMailer envoie les mails par un serveur SMTP. Chaque envoi ouvre sa propre connexion :
// le débit des mails de l'application ne justifie pas un pool, et c'est sûr pour la concurrence.
type SMTPMailer struct {
	cfg    config.SMTP
	sender Sender
	clock  clock.Clock
	// tlsConfig est surchargeable par les tests (certificat auto-signé).
	tlsConfig *tls.Config
}

// NewSMTPMailer crée l'expéditeur SMTP.
func NewSMTPMailer(cfg config.SMTP, sender Sender, clk clock.Clock) *SMTPMailer {
	return &SMTPMailer{cfg: cfg, sender: sender, clock: clk}
}

// Send construit puis envoie le message. Le délai total est borné par le contexte et par
// le délai de la configuration.
func (m *SMTPMailer) Send(ctx context.Context, message Message) error {
	recipient, err := message.validate()
	if err != nil {
		return err
	}
	payload, err := build(m.sender, message, m.clock.Now())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timeout)
	defer cancel()

	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if err := m.authenticate(client); err != nil {
		return err
	}
	if err := client.Mail(m.sender.Address); err != nil {
		return fmt.Errorf("mail : MAIL FROM : %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("mail : RCPT TO : %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail : DATA : %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("mail : écriture du message : %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("mail : fin du message : %w", err)
	}
	return client.Quit()
}

func (m *SMTPMailer) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("mail : connexion au serveur SMTP : %w", err)
	}
	// Le délai du contexte couvre aussi les échanges, pas seulement la connexion.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	tlsConfig := m.tlsConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}
	}

	if m.cfg.TLSMode == config.TLSModeImplicit {
		conn = tls.Client(conn, tlsConfig)
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("mail : dialogue SMTP : %w", err)
	}
	if m.cfg.TLSMode == config.TLSModeStartTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("mail : STARTTLS : %w", err)
		}
	}
	return client, nil
}

// authenticate s'identifie si des identifiants sont configurés. smtp.PlainAuth refuse d'envoyer
// le mot de passe sur une connexion non chiffrée (sauf vers localhost).
func (m *SMTPMailer) authenticate(client *smtp.Client) error {
	if m.cfg.Username == "" {
		return nil
	}
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mail : authentification SMTP : %w", err)
	}
	return nil
}
