package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const tableEmailChanges = "email_change_requests"

type emailChangeRecord struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	UserID     uuid.UUID
	NewEmail   string
	Status     string
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	CreatedAt  time.Time
}

func (emailChangeRecord) TableName() string { return tableEmailChanges }

func (r emailChangeRecord) toEmailChange() EmailChange {
	return EmailChange{
		ID: r.ID, UserID: r.UserID, NewEmail: r.NewEmail, Status: ChangeStatus(r.Status),
		ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, CreatedAt: r.CreatedAt,
	}
}

// EmailChangeRepository est le stockage des demandes de changement d'adresse.
type EmailChangeRepository interface {
	// Replace dépose la demande d'un compte. Elle remplace sa demande en attente, s'il en a une, par
	// une nouvelle demande portant un nouvel identifiant : un modérateur qui avait ouvert l'ancienne ne
	// peut pas, sans le savoir, valider la nouvelle adresse.
	Replace(ctx context.Context, id, userID uuid.UUID, newEmail string, now time.Time) error
	// FindPending retourne la demande en attente d'un compte (ErrChangeNotFound s'il n'en a pas).
	FindPending(ctx context.Context, userID uuid.UUID) (*EmailChange, error)
	// List retourne les demandes d'un statut (toutes si vide), les plus anciennes d'abord.
	List(ctx context.Context, status ChangeStatus, limit, offset int) ([]EmailChange, int64, error)
	// Review fait passer une demande de « en attente » à approved ou rejected et la retourne. Deux
	// modérateurs ne peuvent pas la traiter tous les deux : le second reçoit ErrChangeNotPending.
	Review(ctx context.Context, id, reviewer uuid.UUID, to ChangeStatus, now time.Time) (*EmailChange, error)
}

type gormEmailChangeRepository struct {
	db *gorm.DB
}

// NewEmailChangeRepository crée le dépôt adossé à PostgreSQL.
func NewEmailChangeRepository(db *gorm.DB) EmailChangeRepository {
	return &gormEmailChangeRepository{db: db}
}

func (r *gormEmailChangeRepository) conn(ctx context.Context) *gorm.DB {
	return database.Conn(ctx, r.db)
}

// replaceSQL s'appuie sur l'index unique partiel « une seule demande en attente par compte » : en cas
// de conflit, la demande en attente est réécrite sur place, nouvel identifiant compris.
const replaceSQL = `
INSERT INTO email_change_requests (id, user_id, new_email, status, created_at)
VALUES (?, ?, ?, 'pending', ?)
ON CONFLICT (user_id) WHERE status = 'pending' DO UPDATE SET
    id = EXCLUDED.id,
    new_email = EXCLUDED.new_email,
    created_at = EXCLUDED.created_at`

func (r *gormEmailChangeRepository) Replace(ctx context.Context, id, userID uuid.UUID, newEmail string, now time.Time) error {
	if err := r.conn(ctx).Exec(replaceSQL, id, userID, newEmail, now).Error; err != nil {
		return fmt.Errorf("account : dépôt de la demande de changement d'adresse : %w", err)
	}
	return nil
}

func (r *gormEmailChangeRepository) FindPending(ctx context.Context, userID uuid.UUID) (*EmailChange, error) {
	var rec emailChangeRecord
	err := r.conn(ctx).Where("user_id = ? AND status = ?", userID, string(ChangePending)).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrChangeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account : lecture de la demande de changement d'adresse : %w", err)
	}
	change := rec.toEmailChange()
	return &change, nil
}

func (r *gormEmailChangeRepository) find(ctx context.Context, id uuid.UUID) (*EmailChange, error) {
	var rec emailChangeRecord
	err := r.conn(ctx).Where("id = ?", id).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrChangeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account : lecture de la demande de changement d'adresse : %w", err)
	}
	change := rec.toEmailChange()
	return &change, nil
}

func (r *gormEmailChangeRepository) List(ctx context.Context, status ChangeStatus, limit, offset int) ([]EmailChange, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Model(&emailChangeRecord{})
		if status != "" {
			query = query.Where("status = ?", string(status))
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("account : comptage des demandes de changement d'adresse : %w", err)
	}
	var records []emailChangeRecord
	// Les plus anciennes d'abord : c'est la file de travail des modérateurs.
	if err := filtered().Order("created_at ASC, id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("account : liste des demandes de changement d'adresse : %w", err)
	}
	changes := make([]EmailChange, len(records))
	for i, rec := range records {
		changes[i] = rec.toEmailChange()
	}
	return changes, total, nil
}

const reviewSQL = `
UPDATE email_change_requests
SET status = ?, reviewed_by = ?, reviewed_at = ?
WHERE id = ? AND status = 'pending'
RETURNING id, user_id, new_email, status, reviewed_by, reviewed_at, created_at`

func (r *gormEmailChangeRepository) Review(ctx context.Context, id, reviewer uuid.UUID, to ChangeStatus, now time.Time) (*EmailChange, error) {
	var rec emailChangeRecord
	query := r.conn(ctx).Raw(reviewSQL, string(to), reviewer, now, id).Scan(&rec)
	if query.Error != nil {
		return nil, fmt.Errorf("account : examen de la demande de changement d'adresse : %w", query.Error)
	}
	if query.RowsAffected > 0 {
		change := rec.toEmailChange()
		return &change, nil
	}

	// Aucune demande en attente : elle n'existe pas, ou quelqu'un l'a déjà traitée.
	if _, err := r.find(ctx, id); err != nil {
		return nil, err
	}
	return nil, ErrChangeNotPending
}
