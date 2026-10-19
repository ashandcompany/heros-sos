package hero

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableHeroProfiles    = "hero_profiles"
	aliasUniqueConstrain = "hero_profiles_alias_key"
)

type profileRecord struct {
	UserID             uuid.UUID `gorm:"primaryKey"`
	Alias              string
	Bio                string
	RequestMessage     string
	VerificationStatus string
	ReviewedBy         *uuid.UUID
	ReviewedAt         *time.Time
	RejectionReason    *string
	CreatedAt          time.Time
}

func (profileRecord) TableName() string { return tableHeroProfiles }

func (r profileRecord) toProfile() Profile {
	profile := Profile{
		UserID: r.UserID, Alias: r.Alias, Bio: r.Bio, RequestMessage: r.RequestMessage,
		Status: VerificationStatus(r.VerificationStatus), ReviewedBy: r.ReviewedBy,
		ReviewedAt: r.ReviewedAt, CreatedAt: r.CreatedAt,
	}
	if r.RejectionReason != nil {
		profile.RejectionReason = *r.RejectionReason
	}
	return profile
}

// Repository est le stockage des profils de super-héros.
type Repository interface {
	// Request dépose une demande (ou la redépose après un refus). Il retourne ErrAlreadyPending,
	// ErrAlreadyHero ou ErrAliasTaken quand la demande ne peut pas être enregistrée.
	Request(ctx context.Context, userID uuid.UUID, alias, bio, message string, now time.Time) error
	Find(ctx context.Context, userID uuid.UUID) (*Profile, error)
	List(ctx context.Context, status VerificationStatus, limit, offset int) ([]Profile, int64, error)
	// Review fait passer une demande de « en attente » à approved ou rejected. Il retourne
	// false si la demande n'était plus en attente : deux modérateurs ne peuvent pas la traiter
	// tous les deux.
	Review(ctx context.Context, userID, reviewer uuid.UUID, to VerificationStatus, reason string, now time.Time) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

// requestSQL insère la demande, ou la remplace si l'utilisateur en avait une refusée. Une demande
// en attente ou acceptée n'est jamais écrasée : la clause WHERE ne laisse passer que « rejected ».
const requestSQL = `
INSERT INTO hero_profiles (user_id, alias, bio, request_message, verification_status, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    alias = EXCLUDED.alias,
    bio = EXCLUDED.bio,
    request_message = EXCLUDED.request_message,
    verification_status = EXCLUDED.verification_status,
    reviewed_by = NULL,
    reviewed_at = NULL,
    rejection_reason = NULL,
    created_at = EXCLUDED.created_at
WHERE hero_profiles.verification_status = ?
RETURNING user_id`

func (r *gormRepository) Request(ctx context.Context, userID uuid.UUID, alias, bio, message string, now time.Time) error {
	var result struct{ UserID uuid.UUID }
	query := r.conn(ctx).Raw(requestSQL, userID, alias, bio, message, string(StatusPending), now, string(StatusRejected)).Scan(&result)
	if query.Error != nil {
		if database.IsUniqueViolation(query.Error, aliasUniqueConstrain) {
			return ErrAliasTaken
		}
		return fmt.Errorf("hero : dépôt de la demande : %w", query.Error)
	}
	if query.RowsAffected > 0 {
		return nil
	}

	// Aucune ligne écrite : l'utilisateur a déjà une demande en attente ou acceptée.
	existing, err := r.Find(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Status == StatusApproved {
		return ErrAlreadyHero
	}
	return ErrAlreadyPending
}

func (r *gormRepository) Find(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var rec profileRecord
	err := r.conn(ctx).Where("user_id = ?", userID).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("hero : lecture du profil : %w", err)
	}
	profile := rec.toProfile()
	return &profile, nil
}

func (r *gormRepository) List(ctx context.Context, status VerificationStatus, limit, offset int) ([]Profile, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Model(&profileRecord{})
		if status != "" {
			query = query.Where("verification_status = ?", string(status))
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : comptage des demandes : %w", err)
	}
	var records []profileRecord
	// Les demandes les plus anciennes d'abord : c'est la file de travail des modérateurs.
	if err := filtered().Order("created_at ASC, user_id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : liste des demandes : %w", err)
	}
	profiles := make([]Profile, len(records))
	for i, rec := range records {
		profiles[i] = rec.toProfile()
	}
	return profiles, total, nil
}

func (r *gormRepository) Review(ctx context.Context, userID, reviewer uuid.UUID, to VerificationStatus, reason string, now time.Time) (bool, error) {
	values := map[string]any{
		"verification_status": string(to),
		"reviewed_by":         reviewer,
		"reviewed_at":         now,
		"rejection_reason":    nil,
	}
	if reason != "" {
		values["rejection_reason"] = reason
	}
	result := r.conn(ctx).Table(tableHeroProfiles).
		Where("user_id = ? AND verification_status = ?", userID, string(StatusPending)).
		Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("hero : examen de la demande : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}
