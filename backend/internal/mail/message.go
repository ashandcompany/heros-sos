// Package mail envoie les mails de l'application : construction MIME, envoi SMTP, file d'envoi
// asynchrone et modèles. Les autres paquets ne dépendent que de l'interface Mailer.
package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// fallbackMessageIDDomain complète le Message-ID quand l'adresse d'expédition n'a pas de domaine.
const fallbackMessageIDDomain = "localhost"

// Message est un mail à envoyer, avec une version texte et une version HTML.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer envoie un mail. Les implémentations sont sûres pour un usage concurrent.
type Mailer interface {
	Send(ctx context.Context, message Message) error
}

// ErrInvalidMessage indique un message refusé avant tout envoi.
var ErrInvalidMessage = errors.New("mail : message invalide")

// Sender est l'expéditeur des mails.
type Sender struct {
	Address string
	Name    string
}

func (s Sender) header() string {
	return (&mail.Address{Name: s.Name, Address: s.Address}).String()
}

// validate refuse un message qui permettrait une injection d'en-têtes : saut de ligne dans le
// destinataire ou le sujet, adresse qui n'est pas une adresse simple.
func (m Message) validate() (recipient *mail.Address, err error) {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return nil, fmt.Errorf("%w : saut de ligne dans un en-tête", ErrInvalidMessage)
	}
	recipient, err = mail.ParseAddress(m.To)
	if err != nil || recipient.Name != "" {
		return nil, fmt.Errorf("%w : destinataire", ErrInvalidMessage)
	}
	if strings.TrimSpace(m.Subject) == "" || (m.Text == "" && m.HTML == "") {
		return nil, fmt.Errorf("%w : sujet ou contenu vide", ErrInvalidMessage)
	}
	return recipient, nil
}

// build écrit le message au format MIME multipart/alternative : texte puis HTML, encodés en
// quoted-printable, avec les en-têtes encodés pour les caractères accentués.
func build(sender Sender, message Message, now time.Time) ([]byte, error) {
	recipient, err := message.validate()
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	parts := multipart.NewWriter(&body)

	headers := []struct{ name, value string }{
		{"From", sender.header()},
		{"To", recipient.String()},
		{"Subject", mime.QEncoding.Encode("utf-8", message.Subject)},
		{"Date", now.Format(time.RFC1123Z)},
		{"Message-ID", fmt.Sprintf("<%s@%s>", ids.New(), domainOf(sender.Address))},
		{"MIME-Version", "1.0"},
		{"Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", parts.Boundary())},
	}
	var output bytes.Buffer
	for _, header := range headers {
		fmt.Fprintf(&output, "%s: %s\r\n", header.name, header.value)
	}
	output.WriteString("\r\n")

	for _, part := range []struct{ contentType, content string }{
		{"text/plain", message.Text},
		{"text/html", message.HTML},
	} {
		if part.content == "" {
			continue
		}
		if err := writePart(parts, part.contentType, part.content); err != nil {
			return nil, err
		}
	}
	if err := parts.Close(); err != nil {
		return nil, fmt.Errorf("mail : fermeture du message : %w", err)
	}
	output.Write(body.Bytes())
	return output.Bytes(), nil
}

func writePart(parts *multipart.Writer, contentType, content string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType+"; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := parts.CreatePart(header)
	if err != nil {
		return fmt.Errorf("mail : création d'une partie : %w", err)
	}
	encoder := quotedprintable.NewWriter(part)
	if _, err := io.WriteString(encoder, content); err != nil {
		return fmt.Errorf("mail : encodage d'une partie : %w", err)
	}
	return encoder.Close()
}

func domainOf(address string) string {
	if at := strings.LastIndex(address, "@"); at >= 0 {
		return address[at+1:]
	}
	return fallbackMessageIDDomain
}
