package mail

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Noms des modèles : <nom>.txt.tmpl et <nom>.html.tmpl dans templates/.
const (
	templateActivation         = "activation"
	templatePasswordReset      = "password_reset"
	templatePasswordChanged    = "password_changed"
	templateIncident           = "incident_reported"
	templateInterventionInvite = "intervention_invitation"
	templateOutcomeValidation  = "outcome_validation"
	templateOutcomeDisputed    = "outcome_disputed"
	templateReviewRequest      = "hero_review_request"
	templateContactMessage     = "contact_message"
	templateNewsletterConfirm  = "newsletter_confirmation"
	templateNewsletter         = "newsletter"
	// templateLayout enveloppe le fragment HTML de chaque mail : fond, logo, carte et pied.
	templateLayout             = "layout"
	templateEmailChangeConfirm = "email_change_confirmation"
	templateEmailChangeApprove = "email_change_approved"
	templateEmailChangeNotice  = "email_change_notice"
	templateEmailChangeReject  = "email_change_rejected"
)

const (
	subjectActivation         = "Activez votre compte %s"
	subjectPasswordReset      = "Réinitialisation de votre mot de passe %s"
	subjectPasswordChanged    = "Le mot de passe de votre compte %s a été modifié"
	subjectIncident           = "Nouvel incident déclaré sur %s"
	subjectInterventionInvite = "Vous êtes invité à une intervention sur %s"
	subjectOutcomeValidation  = "Validation du résultat d'une intervention sur %s"
	subjectOutcomeDisputed    = "Désaccord sur une intervention sur %s"
	subjectReviewRequest      = "Évaluez l'intervention de %s"
	// Le sujet du message de contact ne porte que l'objet choisi dans une liste fixe : le nom et le
	// texte, saisis par le visiteur, ne figurent que dans le corps.
	subjectContactMessage    = "[%s] Message de contact : %s"
	subjectNewsletterConfirm = "Confirmez votre abonnement à la newsletter %s"

	subjectEmailChangeConfirm = "Confirmez votre nouvelle adresse email %s"
	subjectEmailChangeApprove = "Votre nouvelle adresse email est active sur %s"
	subjectEmailChangeNotice  = "L'adresse email de votre compte %s a été remplacée"
	subjectEmailChangeReject  = "Votre demande de changement d'adresse email a été refusée"
)

// templateData est ce que voient les modèles.
type templateData struct {
	AppName        string
	FirstName      string
	Link           string
	ExpiresIn      string
	IncidentTitle  string
	IncidentType   string
	InterventionID string
	SenderName     string
	SenderEmail    string
	ContactSubject string
	Body           string
	// NewEmail est la nouvelle adresse d'un compte, citée dans l'avertissement envoyé à l'ancienne.
	NewEmail string
	// UnsubscribeLink est le lien de désinscription que porte chaque newsletter.
	UnsubscribeLink string
	// NearbyRadiusKm est le rayon d'alerte d'un héros prévenu pour sa proximité (0 sinon).
	NearbyRadiusKm int
	// BodyHTML est le corps d'une newsletter, déjà nettoyé par l'appelant (liste blanche de balises).
	BodyHTML htmltemplate.HTML

	// Renseignés par deliver pour le gabarit commun.
	LogoURL string
	Content htmltemplate.HTML
}

// Service compose les mails de l'application à partir des modèles et les confie à un Mailer.
type Service struct {
	mailer  Mailer
	appName string
	logoURL string
	tokens  config.Tokens
	text    *texttemplate.Template
	html    *htmltemplate.Template
}

// Option ajuste un Service à sa création.
type Option func(*Service)

// WithLogoURL donne l'adresse absolue du logo affiché en tête de chaque mail.
func WithLogoURL(logoURL string) Option {
	return func(s *Service) { s.logoURL = logoURL }
}

// NewService charge les modèles (une erreur de modèle est détectée au démarrage, pas au premier
// envoi). appName est le nom du service dans les mails.
func NewService(mailer Mailer, appName string, tokens config.Tokens, options ...Option) (*Service, error) {
	text, err := texttemplate.ParseFS(templateFS, "templates/*.txt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("mail : modèles texte : %w", err)
	}
	html, err := htmltemplate.ParseFS(templateFS, "templates/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("mail : modèles HTML : %w", err)
	}
	service := &Service{mailer: mailer, appName: appName, tokens: tokens, text: text, html: html}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

// SendActivation envoie le lien d'activation d'un compte.
func (s *Service) SendActivation(ctx context.Context, to, firstName, link string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectActivation, s.appName), templateActivation, templateData{
		FirstName: firstName, Link: link, ExpiresIn: formatDuration(s.tokens.ActivationTTL),
	})
}

// SendPasswordReset envoie le lien de réinitialisation d'un mot de passe.
func (s *Service) SendPasswordReset(ctx context.Context, to, firstName, link string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectPasswordReset, s.appName), templatePasswordReset, templateData{
		FirstName: firstName, Link: link, ExpiresIn: formatDuration(s.tokens.PasswordResetTTL),
	})
}

// SendPasswordChanged prévient le titulaire d'un compte que son mot de passe vient d'être changé : si ce n'est pas
// lui, il est le premier à pouvoir réagir.
func (s *Service) SendPasswordChanged(ctx context.Context, to, firstName string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectPasswordChanged, s.appName), templatePasswordChanged, templateData{
		FirstName: firstName,
	})
}

// SendIncidentReported prévient un destinataire d'un nouvel incident. Le sujet est fixe : le
// titre, saisi par un utilisateur, ne figure que dans le corps. nearbyRadiusKm, pour un héros prévenu
// parce qu'il est proche, est le rayon d'alerte cité dans le mail (0 sinon : rien n'est dit).
func (s *Service) SendIncidentReported(ctx context.Context, to, title, interventionType string, nearbyRadiusKm int) error {
	return s.send(ctx, to, fmt.Sprintf(subjectIncident, s.appName), templateIncident, templateData{
		IncidentTitle: title, IncidentType: interventionType, NearbyRadiusKm: nearbyRadiusKm,
	})
}

// SendInterventionInvitation notifie un héros qu'il a été invité à une intervention.
func (s *Service) SendInterventionInvitation(ctx context.Context, to string, interventionID uuid.UUID) error {
	return s.send(ctx, to, fmt.Sprintf(subjectInterventionInvite, s.appName), templateInterventionInvite, templateData{
		InterventionID: interventionID.String(),
	})
}

// SendOutcomeValidation demande à un participant de valider le résultat proposé.
func (s *Service) SendOutcomeValidation(ctx context.Context, to string, interventionID uuid.UUID) error {
	return s.send(ctx, to, fmt.Sprintf(subjectOutcomeValidation, s.appName), templateOutcomeValidation, templateData{
		InterventionID: interventionID.String(),
	})
}

// SendOutcomeDisputed prévient un modérateur qu'un participant a refusé le résultat proposé.
func (s *Service) SendOutcomeDisputed(ctx context.Context, to string, interventionID uuid.UUID) error {
	return s.send(ctx, to, fmt.Sprintf(subjectOutcomeDisputed, s.appName), templateOutcomeDisputed, templateData{
		InterventionID: interventionID.String(),
	})
}

// SendReviewRequest demande au déclarant de noter un héros après une intervention clôturée.
func (s *Service) SendReviewRequest(ctx context.Context, to, link string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectReviewRequest, s.appName), templateReviewRequest, templateData{Link: link})
}

// ContactMessage est un message du formulaire de contact, destiné à l'équipe.
type ContactMessage struct {
	SenderName  string
	SenderEmail string
	// Subject est le libellé de l'objet choisi par le visiteur (liste fixe).
	Subject string
	Body    string
}

// SendContactMessage transmet un message du formulaire de contact à l'équipe. L'adresse du visiteur
// sert d'adresse de réponse : répondre au mail répond au visiteur. Le mail n'est jamais envoyé au
// visiteur lui-même, pour que le formulaire ne serve pas à écrire à des tiers.
func (s *Service) SendContactMessage(ctx context.Context, to string, message ContactMessage) error {
	return s.deliver(ctx, Message{
		To: to, ReplyTo: message.SenderEmail, Subject: fmt.Sprintf(subjectContactMessage, s.appName, message.Subject),
	}, templateContactMessage, templateData{
		SenderName: message.SenderName, SenderEmail: message.SenderEmail,
		ContactSubject: message.Subject, Body: message.Body,
	})
}

// SendNewsletterConfirmation envoie le lien qui confirme un abonnement à la newsletter.
func (s *Service) SendNewsletterConfirmation(ctx context.Context, to, link string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectNewsletterConfirm, s.appName), templateNewsletterConfirm, templateData{
		Link: link, ExpiresIn: formatDuration(s.tokens.NewsletterConfirmTTL),
	})
}

// SendEmailChangeConfirmation envoie, à la NOUVELLE adresse, le lien qui prouve qu'elle appartient bien
// au titulaire du compte.
func (s *Service) SendEmailChangeConfirmation(ctx context.Context, to, firstName, link string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectEmailChangeConfirm, s.appName), templateEmailChangeConfirm, templateData{
		FirstName: firstName, Link: link, ExpiresIn: formatDuration(s.tokens.EmailChangeConfirmTTL),
	})
}

// SendEmailChangeApproved prévient, à la nouvelle adresse, que la modération a accepté le changement.
func (s *Service) SendEmailChangeApproved(ctx context.Context, to, firstName string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectEmailChangeApprove, s.appName), templateEmailChangeApprove, templateData{
		FirstName: firstName,
	})
}

// SendEmailChangeNotice prévient l'ANCIENNE adresse qu'elle a été remplacée : si ce n'est pas voulu, son
// titulaire est le premier à pouvoir réagir.
func (s *Service) SendEmailChangeNotice(ctx context.Context, to, firstName, newEmail string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectEmailChangeNotice, s.appName), templateEmailChangeNotice, templateData{
		FirstName: firstName, NewEmail: newEmail,
	})
}

// SendEmailChangeRejected prévient, à l'adresse actuelle, que la modération a refusé le changement.
func (s *Service) SendEmailChangeRejected(ctx context.Context, to, firstName string) error {
	return s.send(ctx, to, subjectEmailChangeReject, templateEmailChangeReject, templateData{FirstName: firstName})
}

// ErrMissingUnsubscribeLink indique une newsletter sans lien de désinscription : elle n'est jamais envoyée.
var ErrMissingUnsubscribeLink = errors.New("mail : une newsletter doit porter son lien de désinscription")

// SendNewsletter envoie un numéro de la newsletter à un abonné. C'est la seule façon d'en envoyer une :
// le gabarit ajoute le lien de désinscription au pied du message et en en-tête List-Unsubscribe, et un
// envoi sans ce lien est refusé. bodyHTML doit déjà être nettoyé (liste blanche de balises) : il est
// inséré tel quel dans la carte du message, et la version texte en est dérivée.
func (s *Service) SendNewsletter(ctx context.Context, to, subject, bodyHTML, unsubscribeLink string) error {
	if strings.TrimSpace(unsubscribeLink) == "" {
		return ErrMissingUnsubscribeLink
	}
	return s.deliver(ctx, Message{To: to, Subject: subject, ListUnsubscribe: unsubscribeLink}, templateNewsletter, templateData{
		Body: plainText(bodyHTML), BodyHTML: htmltemplate.HTML(bodyHTML), UnsubscribeLink: unsubscribeLink, //nolint:gosec // nettoyé par l'appelant
	})
}

func (s *Service) send(ctx context.Context, to, subject, name string, data templateData) error {
	return s.deliver(ctx, Message{To: to, Subject: subject}, name, data)
}

// deliver compose le corps du message à partir du modèle, puis le confie au Mailer. La version HTML
// passe par le gabarit commun : le modèle du mail n'en est que le contenu.
func (s *Service) deliver(ctx context.Context, message Message, name string, data templateData) error {
	data.AppName = s.appName
	data.LogoURL = s.logoURL

	var text, fragment, page bytes.Buffer
	if err := s.text.ExecuteTemplate(&text, name+".txt.tmpl", data); err != nil {
		return fmt.Errorf("mail : modèle texte %s : %w", name, err)
	}
	if err := s.html.ExecuteTemplate(&fragment, name+".html.tmpl", data); err != nil {
		return fmt.Errorf("mail : modèle HTML %s : %w", name, err)
	}
	data.Content = htmltemplate.HTML(fragment.String()) //nolint:gosec // produit par nos modèles, déjà échappé
	if err := s.html.ExecuteTemplate(&page, templateLayout+".html.tmpl", data); err != nil {
		return fmt.Errorf("mail : gabarit HTML : %w", err)
	}
	message.Text, message.HTML = text.String(), page.String()
	return s.mailer.Send(ctx, message)
}

// formatDuration écrit une durée de validité en français : « 24 heures », « 30 minutes ».
func formatDuration(d time.Duration) string {
	if d >= time.Hour && d%time.Hour == 0 {
		return plural(int(d/time.Hour), "heure")
	}
	return plural(int(d.Round(time.Minute)/time.Minute), "minute")
}

func plural(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(count) + " " + unit + "s"
}
