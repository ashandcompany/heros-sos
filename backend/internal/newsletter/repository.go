package newsletter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const tableSubscriptions = "newsletter_subscriptions"

// Repository est le stockage des abonnements. Chaque changement d'état se fait en une seule
// instruction SQL : deux requêtes simultanées sur la même adresse ne peuvent pas se contredire.
type Repository interface {
	// Request dépose (ou relance) la demande d'un visiteur. Il retourne l'identifiant de
	// l'abonnement et true quand un mail de confirmation doit partir ; false (et aucune écriture) si
	// l'adresse est déjà abonnée. Un abonné qui s'était retiré repasse « en attente » : il doit
	// consentir de nouveau.
	Request(ctx context.Context, id uuid.UUID, email string, now time.Time) (subscriptionID uuid.UUID, confirmationNeeded bool, err error)
	// Confirm valide une demande en attente. C'est idempotent pour un abonnement déjà confirmé ;
	// ErrLinkInvalid pour une demande retirée ou inexistante.
	Confirm(ctx context.Context, id uuid.UUID, now time.Time) error

	// IsSubscribed indique si le titulaire d'un compte reçoit la newsletter, que son abonnement soit
	// rattaché à son compte ou seulement à son adresse (abonné depuis le pied de page).
	IsSubscribed(ctx context.Context, userID uuid.UUID, email string) (bool, error)
	// SetAccountPreference abonne (directement, l'adresse d'un compte est déjà vérifiée) ou
	// désabonne un titulaire de compte. C'est idempotent.
	SetAccountPreference(ctx context.Context, id, userID uuid.UUID, email string, subscribed bool, now time.Time) error
	// FollowEmail aligne l'adresse de l'abonnement d'un compte sur sa nouvelle adresse. Si cette
	// adresse est déjà dans la liste, l'abonnement existant n'est pas modifié.
	FollowEmail(ctx context.Context, userID uuid.UUID, newEmail string) error

	// DeleteStalePending supprime les demandes jamais confirmées, déposées avant la date donnée.
	DeleteStalePending(ctx context.Context, before time.Time) (int64, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

// requestSQL insère la demande, ou la relance si l'adresse est connue sans être abonnée. La clause
// WHERE ne laisse jamais écraser un abonnement confirmé : dans ce cas aucune ligne n'est retournée.
const requestSQL = `
INSERT INTO newsletter_subscriptions (id, email, status, source, created_at)
VALUES (?, ?, 'pending', 'footer', ?)
ON CONFLICT (email) DO UPDATE SET
    status = 'pending',
    source = 'footer',
    confirmed_at = NULL,
    unsubscribed_at = NULL,
    created_at = EXCLUDED.created_at
WHERE newsletter_subscriptions.status <> 'subscribed'
RETURNING id`

func (r *gormRepository) Request(ctx context.Context, id uuid.UUID, email string, now time.Time) (uuid.UUID, bool, error) {
	var result struct{ ID uuid.UUID }
	query := r.conn(ctx).Raw(requestSQL, id, email, now).Scan(&result)
	if query.Error != nil {
		return uuid.Nil, false, fmt.Errorf("newsletter : demande d'abonnement : %w", query.Error)
	}
	if query.RowsAffected == 0 {
		return uuid.Nil, false, nil
	}
	return result.ID, true, nil
}

const confirmSQL = `
UPDATE newsletter_subscriptions
SET status = 'subscribed', confirmed_at = ?, unsubscribed_at = NULL
WHERE id = ? AND status = 'pending'
RETURNING id`

func (r *gormRepository) Confirm(ctx context.Context, id uuid.UUID, now time.Time) error {
	var result struct{ ID uuid.UUID }
	query := r.conn(ctx).Raw(confirmSQL, now, id).Scan(&result)
	if query.Error != nil {
		return fmt.Errorf("newsletter : confirmation : %w", query.Error)
	}
	if query.RowsAffected > 0 {
		return nil
	}

	// Aucune demande en attente : un second clic sur le lien reste un succès, un abonnement retiré non.
	var current struct{ Status string }
	err := r.conn(ctx).Table(tableSubscriptions).Select("status").Where("id = ?", id).Take(&current).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrLinkInvalid
	case err != nil:
		return fmt.Errorf("newsletter : lecture de l'abonnement : %w", err)
	case Status(current.Status) == StatusSubscribed:
		return nil
	}
	return ErrLinkInvalid
}

func (r *gormRepository) IsSubscribed(ctx context.Context, userID uuid.UUID, email string) (bool, error) {
	var subscribed bool
	err := r.conn(ctx).Raw(
		`SELECT EXISTS (SELECT 1 FROM newsletter_subscriptions WHERE status = 'subscribed' AND (user_id = ? OR email = ?))`,
		userID, email,
	).Scan(&subscribed).Error
	if err != nil {
		return false, fmt.Errorf("newsletter : lecture de la préférence : %w", err)
	}
	return subscribed, nil
}

// subscribeAccountSQL abonne un titulaire de compte. S'il était déjà abonné, la date de son
// consentement d'origine est conservée ; sinon (retiré, ou en attente depuis le pied de page) le
// consentement est celui d'aujourd'hui.
const subscribeAccountSQL = `
INSERT INTO newsletter_subscriptions (id, email, user_id, status, source, confirmed_at, created_at)
VALUES (?, ?, ?, 'subscribed', 'account', ?, ?)
ON CONFLICT (email) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    confirmed_at = CASE WHEN newsletter_subscriptions.status = 'subscribed'
        THEN newsletter_subscriptions.confirmed_at ELSE EXCLUDED.confirmed_at END,
    status = 'subscribed',
    unsubscribed_at = NULL`

const unsubscribeAccountSQL = `
UPDATE newsletter_subscriptions
SET status = 'unsubscribed', unsubscribed_at = ?
WHERE (user_id = ? OR email = ?) AND status <> 'unsubscribed'`

func (r *gormRepository) SetAccountPreference(ctx context.Context, id, userID uuid.UUID, email string, subscribed bool, now time.Time) error {
	var err error
	if subscribed {
		err = r.conn(ctx).Exec(subscribeAccountSQL, id, email, userID, now, now).Error
	} else {
		err = r.conn(ctx).Exec(unsubscribeAccountSQL, now, userID, email).Error
	}
	if err != nil {
		return fmt.Errorf("newsletter : enregistrement de la préférence : %w", err)
	}
	return nil
}

const followEmailSQL = `
UPDATE newsletter_subscriptions
SET email = ?
WHERE user_id = ? AND email <> ?
  AND NOT EXISTS (SELECT 1 FROM newsletter_subscriptions other WHERE other.email = ?)`

func (r *gormRepository) FollowEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	if err := r.conn(ctx).Exec(followEmailSQL, newEmail, userID, newEmail, newEmail).Error; err != nil {
		return fmt.Errorf("newsletter : suivi du changement d'adresse : %w", err)
	}
	return nil
}

func (r *gormRepository) DeleteStalePending(ctx context.Context, before time.Time) (int64, error) {
	result := r.conn(ctx).Exec(`DELETE FROM newsletter_subscriptions WHERE status = 'pending' AND created_at < ?`, before)
	if result.Error != nil {
		return 0, fmt.Errorf("newsletter : purge des demandes : %w", result.Error)
	}
	return result.RowsAffected, nil
}
