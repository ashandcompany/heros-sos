package mail

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	htmltemplate "html/template"
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
	templateIncident           = "incident_reported"
	templateInterventionInvite = "intervention_invitation"
	templateOutcomeValidation  = "outcome_validation"
	templateOutcomeDisputed    = "outcome_disputed"
	templateReviewRequest      = "hero_review_request"
)

const (
	subjectActivation         = "Activez votre compte %s"
	subjectPasswordReset      = "Réinitialisation de votre mot de passe %s"
	subjectIncident           = "Nouvel incident déclaré sur %s"
	subjectInterventionInvite = "Vous êtes invité à une intervention sur %s"
	subjectOutcomeValidation  = "Validation du résultat d'une intervention sur %s"
	subjectOutcomeDisputed    = "Désaccord sur une intervention sur %s"
	subjectReviewRequest      = "Évaluez l'intervention de %s"
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
}

// Service compose les mails de l'application à partir des modèles et les confie à un Mailer.
type Service struct {
	mailer  Mailer
	appName string
	tokens  config.Tokens
	text    *texttemplate.Template
	html    *htmltemplate.Template
}

// NewService charge les modèles (une erreur de modèle est détectée au démarrage, pas au premier
// envoi). appName est le nom du service dans les mails.
func NewService(mailer Mailer, appName string, tokens config.Tokens) (*Service, error) {
	text, err := texttemplate.ParseFS(templateFS, "templates/*.txt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("mail : modèles texte : %w", err)
	}
	html, err := htmltemplate.ParseFS(templateFS, "templates/*.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("mail : modèles HTML : %w", err)
	}
	return &Service{mailer: mailer, appName: appName, tokens: tokens, text: text, html: html}, nil
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

// SendIncidentReported prévient un destinataire d'un nouvel incident. Le sujet est fixe : le
// titre, saisi par un utilisateur, ne figure que dans le corps.
func (s *Service) SendIncidentReported(ctx context.Context, to, title, interventionType string) error {
	return s.send(ctx, to, fmt.Sprintf(subjectIncident, s.appName), templateIncident, templateData{
		IncidentTitle: title, IncidentType: interventionType,
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

func (s *Service) send(ctx context.Context, to, subject, name string, data templateData) error {
	data.AppName = s.appName

	var text, html bytes.Buffer
	if err := s.text.ExecuteTemplate(&text, name+".txt.tmpl", data); err != nil {
		return fmt.Errorf("mail : modèle texte %s : %w", name, err)
	}
	if err := s.html.ExecuteTemplate(&html, name+".html.tmpl", data); err != nil {
		return fmt.Errorf("mail : modèle HTML %s : %w", name, err)
	}
	return s.mailer.Send(ctx, Message{To: to, Subject: subject, Text: text.String(), HTML: html.String()})
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
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", count, unit)
}
