// Package contact reçoit les messages du formulaire de contact et les transmet à l'équipe par mail.
//
// Le message n'est pas conservé en base : le mail est la seule trace. La file d'envoi étant interne
// au processus, un message accepté est perdu si l'API s'arrête brutalement avant son envoi ; un échec
// de mise en file est en revanche annoncé au visiteur (503), qui peut réessayer.
package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Limites du formulaire : les mêmes que celles du frontend (lib/validation/contact.ts).
const (
	MaxNameLength    = 80
	MinMessageLength = 20
	MaxMessageLength = 2000
)

// Objets proposés par le formulaire, et leur libellé dans le sujet du mail envoyé à l'équipe.
const (
	SubjectQuestion    = "question"
	SubjectAccount     = "compte"
	SubjectData        = "donnees"
	SubjectPartnership = "partenariat"
	SubjectOther       = "autre"
)

var subjectLabels = map[string]string{
	SubjectQuestion:    "Question sur le service",
	SubjectAccount:     "Problème avec mon compte",
	SubjectData:        "Mes données personnelles",
	SubjectPartnership: "Presse ou partenariat",
	SubjectOther:       "Autre demande",
}

// ErrDeliveryUnavailable indique que le message n'a pas pu être confié au service d'envoi.
var ErrDeliveryUnavailable = errors.New("contact : envoi du message impossible")

// Mailer transmet un message à l'équipe (implémenté par mail.Service).
type Mailer interface {
	SendContactMessage(ctx context.Context, to string, message mail.ContactMessage) error
}

// Input est la saisie du formulaire (valeurs brutes, non validées).
type Input struct {
	Name    string
	Email   string
	Subject string
	Message string
}

// Service transmet les messages à l'équipe.
type Service struct {
	mailer    Mailer
	recipient string
}

// NewService crée le service. recipient est l'adresse de l'équipe.
func NewService(mailer Mailer, recipient string) *Service {
	return &Service{mailer: mailer, recipient: recipient}
}

// Send valide le message puis le transmet. Toutes les erreurs de saisie sont retournées ensemble
// (*validation.Error). Le mail part à l'équipe seulement, jamais au visiteur : l'adresse saisie ne sert
// que de destinataire des réponses.
func (s *Service) Send(ctx context.Context, in Input) error {
	var fields validation.Collector

	name, code := validation.RequiredText(in.Name, MaxNameLength)
	fields.Add("name", code)
	email, code := validation.Email(in.Email)
	fields.Add("email", code)

	label, known := subjectLabels[in.Subject]
	switch {
	case in.Subject == "":
		fields.Add("subject", validation.CodeRequired)
	case !known:
		fields.Add("subject", validation.CodeInvalid)
	}

	// Seuls les espaces des extrémités sont retirés : paragraphes et retours à la ligne sont conservés.
	body := strings.TrimSpace(in.Message)
	switch length := utf8.RuneCountInString(body); {
	case length == 0:
		fields.Add("message", validation.CodeRequired)
	case length < MinMessageLength:
		fields.Add("message", validation.CodeTooShort)
	case length > MaxMessageLength:
		fields.Add("message", validation.CodeTooLong)
	}
	if err := fields.Err(); err != nil {
		return err
	}

	err := s.mailer.SendContactMessage(ctx, s.recipient, mail.ContactMessage{
		SenderName: name, SenderEmail: email, Subject: label, Body: body,
	})
	if err != nil {
		return fmt.Errorf("%w : %w", ErrDeliveryUnavailable, err)
	}
	return nil
}
