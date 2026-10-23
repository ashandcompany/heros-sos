package account

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

// Les lignes ci-dessous sont celles de l'export des données d'un compte (GET /me/export), décrites dans
// openapi/openapi.yaml (ExportedData). Elles ne contiennent que ce qui concerne le titulaire : un avis reçu
// par un héros n'y porte pas l'identifiant du citoyen qui l'a donné.

// ExportedIncident est un incident déclaré par le titulaire.
type ExportedIncident struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Street      string     `json:"street,omitempty"`
	PostalCode  string     `json:"postalCode,omitempty"`
	City        string     `json:"city,omitempty"`
	Country     string     `json:"country,omitempty"`
	OccurredAt  *time.Time `json:"occurredAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ExportedIntervention est une intervention à laquelle le titulaire, super-héros, a participé.
type ExportedIntervention struct {
	ID         uuid.UUID  `json:"id"`
	IncidentID uuid.UUID  `json:"incidentId"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
}

// ExportedReview est un avis donné par le titulaire (citoyen) ou reçu par lui (super-héros), avec sa réponse.
type ExportedReview struct {
	ID             uuid.UUID `json:"id"`
	InterventionID uuid.UUID `json:"interventionId"`
	HeroID         uuid.UUID `json:"heroId"`
	Rating         int       `json:"rating"`
	Comment        string    `json:"comment,omitempty"`
	Status         string    `json:"status"`
	Reply          string    `json:"reply,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ExportedFilm est un film de la vidéothèque du titulaire, super-héros.
type ExportedFilm struct {
	Title    string    `json:"title"`
	IMDBID   string    `json:"imdbId,omitempty"`
	VideoURL string    `json:"videoUrl,omitempty"`
	Status   string    `json:"status"`
	AddedAt  time.Time `json:"addedAt"`
}

// ExportedEmailChange est une demande de changement d'adresse.
type ExportedEmailChange struct {
	NewEmail    string     `json:"newEmail"`
	Status      string     `json:"status"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ExportedNewsletter est l'abonnement à la newsletter, avec la preuve du consentement.
type ExportedNewsletter struct {
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	Source         string     `json:"source"`
	ConfirmedAt    *time.Time `json:"confirmedAt,omitempty"`
	UnsubscribedAt *time.Time `json:"unsubscribedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// ExportedSession est une session ouverte (le jeton n'est jamais exporté).
type ExportedSession struct {
	CreatedAt   time.Time `json:"createdAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	DeviceLabel string    `json:"deviceLabel,omitempty"`
	IPAddress   string    `json:"ipAddress,omitempty"`
}

// Activity est ce que l'export ajoute au profil : tout ce que le titulaire a laissé dans le service.
type Activity struct {
	Incidents           []ExportedIncident     `json:"incidents"`
	Interventions       []ExportedIntervention `json:"interventions"`
	ReviewsGiven        []ExportedReview       `json:"reviewsGiven"`
	ReviewsReceived     []ExportedReview       `json:"reviewsReceived"`
	Films               []ExportedFilm         `json:"films"`
	EmailChangeRequests []ExportedEmailChange  `json:"emailChangeRequests"`
	Newsletter          *ExportedNewsletter    `json:"newsletter,omitempty"`
	Sessions            []ExportedSession      `json:"sessions"`
	MFAEnabled          bool                   `json:"mfaEnabled"`
}

// PrivacyRepository lit et efface les données d'un compte en dehors de la table users.
type PrivacyRepository interface {
	// Activity rassemble l'activité d'un compte pour son export.
	Activity(ctx context.Context, userID uuid.UUID, now time.Time) (*Activity, error)
	// EraseRelated supprime ce qui ne sert qu'au titulaire (sessions, jetons, 2FA, demandes de changement
	// d'adresse, abonnement à la newsletter, position, vidéothèque) et rend anonyme son profil de héros. Les
	// incidents, interventions et avis restent : ils citent un compte devenu anonyme.
	EraseRelated(ctx context.Context, userID uuid.UUID, email string) error
}

type gormPrivacyRepository struct {
	db *gorm.DB
}

// NewPrivacyRepository crée le dépôt adossé à PostgreSQL.
func NewPrivacyRepository(db *gorm.DB) PrivacyRepository { return &gormPrivacyRepository{db: db} }

func (r *gormPrivacyRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

const (
	exportIncidentsSQL = `
SELECT id, title, description, coalesce(status, 'reported') AS status, coalesce(street, '') AS street,
       coalesce(postal_code, '') AS postal_code, coalesce(city, '') AS city, coalesce(country, '') AS country,
       occurred_at, created_at
  FROM incidents WHERE reporter_id = ? ORDER BY created_at, id`
	exportInterventionsSQL = `
SELECT i.id, i.incident_id, ih.role, i.status, i.started_at, i.ended_at
  FROM intervention_heroes ih JOIN interventions i ON i.id = ih.intervention_id
 WHERE ih.hero_id = ? ORDER BY i.started_at, i.id`
	exportReviewsSQL = `
SELECT r.id, r.intervention_id, r.hero_id, r.rating, coalesce(r.comment, '') AS comment, r.status,
       coalesce(rp.content, '') AS reply, r.created_at
  FROM hero_reviews r LEFT JOIN hero_review_replies rp ON rp.review_id = r.id
 WHERE r.%s = ? ORDER BY r.created_at, r.id`
	exportFilmsSQL = `
SELECT f.title, coalesce(f.imdb_id, '') AS imdb_id, coalesce(f.video_url, '') AS video_url, hf.status,
       hf.created_at AS added_at
  FROM hero_films hf JOIN films f ON f.id = hf.film_id
 WHERE hf.hero_id = ? ORDER BY hf.created_at`
	exportEmailChangesSQL = `
SELECT new_email, status, confirmed_at, created_at
  FROM email_change_requests WHERE user_id = ? ORDER BY created_at`
	exportNewsletterSQL = `
SELECT email, status, source, confirmed_at, unsubscribed_at, created_at
  FROM newsletter_subscriptions WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`
	exportSessionsSQL = `
SELECT created_at, last_seen_at, expires_at, device_label, coalesce(host(ip_address), '') AS ip_address
  FROM sessions WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ? ORDER BY created_at`
	exportMFASQL = `SELECT EXISTS (SELECT 1 FROM user_mfa WHERE user_id = ? AND confirmed_at IS NOT NULL)`
)

func (r *gormPrivacyRepository) Activity(ctx context.Context, userID uuid.UUID, now time.Time) (*Activity, error) {
	activity := &Activity{}
	queries := []struct {
		what   string
		sql    string
		args   []any
		target any
	}{
		{"incidents", exportIncidentsSQL, []any{userID}, &activity.Incidents},
		{"interventions", exportInterventionsSQL, []any{userID}, &activity.Interventions},
		{"avis donnés", fmt.Sprintf(exportReviewsSQL, "citizen_id"), []any{userID}, &activity.ReviewsGiven},
		{"avis reçus", fmt.Sprintf(exportReviewsSQL, "hero_id"), []any{userID}, &activity.ReviewsReceived},
		{"films", exportFilmsSQL, []any{userID}, &activity.Films},
		{"changements d'adresse", exportEmailChangesSQL, []any{userID}, &activity.EmailChangeRequests},
		{"sessions", exportSessionsSQL, []any{userID, now}, &activity.Sessions},
		{"double authentification", exportMFASQL, []any{userID}, &activity.MFAEnabled},
	}
	for _, query := range queries {
		if err := r.conn(ctx).Raw(query.sql, query.args...).Scan(query.target).Error; err != nil {
			return nil, fmt.Errorf("account : export (%s) : %w", query.what, err)
		}
	}
	var newsletter []ExportedNewsletter
	if err := r.conn(ctx).Raw(exportNewsletterSQL, userID).Scan(&newsletter).Error; err != nil {
		return nil, fmt.Errorf("account : export (newsletter) : %w", err)
	}
	if len(newsletter) > 0 {
		activity.Newsletter = &newsletter[0]
	}
	return activity, nil
}

// eraseStatements suppriment ce qui ne sert qu'au titulaire. L'ordre compte : hero_films et hero_locations
// dépendent du profil de héros, qui reste (anonyme) pour les interventions et les avis.
var eraseStatements = []struct {
	what string
	sql  string
}{
	{"sessions", `DELETE FROM sessions WHERE user_id = ?`},
	{"jetons", `DELETE FROM user_tokens WHERE user_id = ?`},
	{"codes de secours", `DELETE FROM mfa_recovery_codes WHERE user_id = ?`},
	{"double authentification", `DELETE FROM user_mfa WHERE user_id = ?`},
	{"changements d'adresse", `DELETE FROM email_change_requests WHERE user_id = ?`},
	{"commandes traitées", `DELETE FROM processed_commands WHERE user_id = ?`},
	{"position", `DELETE FROM hero_locations WHERE hero_id = ?`},
	{"vidéothèque", `DELETE FROM hero_films WHERE hero_id = ?`},
}

// anonymizeHeroSQL donne au profil de héros un pseudo neutre, unique (il porte le début de l'identifiant) et
// conforme aux règles d'un pseudo, et efface tout ce que le héros avait écrit sur lui-même.
const anonymizeHeroSQL = `
UPDATE hero_profiles SET alias = ?, bio = '', request_message = '', rejection_reason = NULL,
       location_consent_at = NULL
 WHERE user_id = ?`

// AnonymizedAlias est le pseudo d'un super-héros dont le compte a été supprimé.
func AnonymizedAlias(userID uuid.UUID) string { return "Ancien héros " + userID.String()[:8] }

func (r *gormPrivacyRepository) EraseRelated(ctx context.Context, userID uuid.UUID, email string) error {
	for _, statement := range eraseStatements {
		if err := r.conn(ctx).Exec(statement.sql, userID).Error; err != nil {
			return fmt.Errorf("account : effacement (%s) : %w", statement.what, err)
		}
	}
	// L'abonnement porte l'adresse : celui du compte, et celui qu'un visiteur aurait pris avec la même adresse.
	if err := r.conn(ctx).Exec(`DELETE FROM newsletter_subscriptions WHERE user_id = ? OR email = ?`, userID, email).Error; err != nil {
		return fmt.Errorf("account : effacement (newsletter) : %w", err)
	}
	if err := r.conn(ctx).Exec(anonymizeHeroSQL, AnonymizedAlias(userID), userID).Error; err != nil {
		return fmt.Errorf("account : effacement (profil de héros) : %w", err)
	}
	return nil
}
