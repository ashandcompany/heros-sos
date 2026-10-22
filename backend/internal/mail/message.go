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
	"net/url"
	"strings"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// fallbackMessageIDDomain complète le Message-ID quand l'adresse d'expédition n'a pas de domaine.
const fallbackMessageIDDomain = "localhost"

// Message est un mail à envoyer, avec une version texte et une version HTML.
type Message struct {
	To string
	// ReplyTo, facultatif, est l'adresse qui reçoit la réponse du destinataire (formulaire de contact).
	ReplyTo string
	Subject string
	Text    string
	HTML    string
	// ListUnsubscribe, facultatif, est l'adresse (http ou https) où se désabonner : elle devient l'en-tête
	// List-Unsubscribe (RFC 2369), que les messageries affichent comme un lien « Se désabonner ». Elle
	// est obligatoire pour une newsletter (voir Service.SendNewsletter).
	ListUnsubscribe string
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
// destinataire, l'adresse de réponse ou le sujet, adresse qui n'est pas une adresse simple.
func (m Message) validate() (recipient *mail.Address, err error) {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.ReplyTo, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return nil, fmt.Errorf("%w : saut de ligne dans un en-tête", ErrInvalidMessage)
	}
	recipient, ok := bareAddress(m.To)
	if !ok {
		return nil, fmt.Errorf("%w : destinataire", ErrInvalidMessage)
	}
	if _, ok := bareAddress(m.ReplyTo); m.ReplyTo != "" && !ok {
		return nil, fmt.Errorf("%w : adresse de réponse", ErrInvalidMessage)
	}
	if strings.TrimSpace(m.Subject) == "" || (m.Text == "" && m.HTML == "") {
		return nil, fmt.Errorf("%w : sujet ou contenu vide", ErrInvalidMessage)
	}
	if m.ListUnsubscribe != "" && !isWebLink(m.ListUnsubscribe) {
		return nil, fmt.Errorf("%w : adresse de désinscription", ErrInvalidMessage)
	}
	return recipient, nil
}

// bareAddress lit une adresse simple : « a@b.c », sans nom affiché.
func bareAddress(raw string) (*mail.Address, bool) {
	address, err := mail.ParseAddress(raw)
	return address, err == nil && address.Name == ""
}

// isWebLink dit si raw est un lien http(s) sûr à placer entre chevrons dans un en-tête : ni chevron, ni espace,
// ni saut de ligne, qui casseraient l'en-tête.
func isWebLink(raw string) bool {
	link, err := url.Parse(raw)
	return err == nil && (link.Scheme == "http" || link.Scheme == "https") && link.Host != "" &&
		!strings.ContainsAny(raw, "\r\n<> ")
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
	if message.ReplyTo != "" {
		headers = append(headers, struct{ name, value string }{"Reply-To", (&mail.Address{Address: message.ReplyTo}).String()})
	}
	if message.ListUnsubscribe != "" {
		headers = append(headers, struct{ name, value string }{"List-Unsubscribe", "<" + message.ListUnsubscribe + ">"})
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
