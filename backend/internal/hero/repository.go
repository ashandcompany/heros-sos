package hero

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
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

type publicProfileRow struct {
	UserID             uuid.UUID `gorm:"column:user_id"`
	Alias              string    `gorm:"column:alias"`
	AvatarPath         *string   `gorm:"column:avatar_path"`
	Bio                string    `gorm:"column:bio"`
	AverageRating      float64   `gorm:"column:average_rating"`
	InterventionsCount int64     `gorm:"column:interventions_count"`
}

func (r publicProfileRow) toPublicProfile() PublicProfile {
	return PublicProfile{
		UserID: r.UserID, Alias: r.Alias, AvatarURL: publicAvatarURL(r.AvatarPath), Bio: r.Bio,
		AverageRating: r.AverageRating, InterventionsCount: r.InterventionsCount,
	}
}

func publicAvatarURL(stored *string) *string {
	if stored == nil {
		return nil
	}
	value := strings.TrimSpace(*stored)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil
	}
	if parsed.IsAbs() {
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
			return nil
		}
	} else if parsed.Host != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") ||
		strings.Contains(value, `\`) || path.Clean(parsed.Path) != parsed.Path {
		return nil
	}
	return &value
}

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
	// ListPublic retourne les profils approuvés des comptes actifs, triés selon l'ordre indiqué.
	ListPublic(ctx context.Context, sort PublicSort, limit, offset int) ([]PublicProfile, int64, error)
	// FindPublic retourne un profil uniquement s'il est approuvé et appartient à un compte actif.
	FindPublic(ctx context.Context, userID uuid.UUID) (*PublicProfile, error)
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

func (r *gormRepository) publicProfiles(ctx context.Context) *gorm.DB {
	ratings := r.conn(ctx).Table("hero_reviews").
		Select("hero_id, AVG(rating)::double precision AS average_rating").
		Where("status = ?", "published").Group("hero_id")
	interventions := r.conn(ctx).Table("intervention_heroes").
		Select("hero_id, COUNT(DISTINCT intervention_id) AS interventions_count").
		Where("participation = ?", "confirmed").Group("hero_id")

	return r.conn(ctx).Table("hero_profiles").
		Select(`hero_profiles.user_id, hero_profiles.alias, users.avatar_path, hero_profiles.bio,
			COALESCE(ratings.average_rating, 0) AS average_rating,
			COALESCE(interventions.interventions_count, 0) AS interventions_count`).
		Joins("JOIN users ON users.id = hero_profiles.user_id").
		Joins("LEFT JOIN (?) AS ratings ON ratings.hero_id = hero_profiles.user_id", ratings).
		Joins("LEFT JOIN (?) AS interventions ON interventions.hero_id = hero_profiles.user_id", interventions).
		Where("hero_profiles.verification_status = ? AND users.role = ? AND users.status = ?",
			string(StatusApproved), "hero", "active")
}

func (r *gormRepository) ListPublic(ctx context.Context, sort PublicSort, limit, offset int) ([]PublicProfile, int64, error) {
	var total int64
	if err := r.publicProfiles(ctx).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : comptage des profils publics : %w", err)
	}
	query := r.publicProfiles(ctx)
	switch sort {
	case PublicSortAlias:
		query = query.Order("hero_profiles.alias ASC, hero_profiles.user_id ASC")
	case PublicSortAverageRating:
		query = query.Order("average_rating DESC, hero_profiles.alias ASC, hero_profiles.user_id ASC")
	default:
		return nil, 0, fmt.Errorf("hero : tri public non pris en charge %q", sort)
	}
	var rows []publicProfileRow
	if err := query.Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : lecture des profils publics : %w", err)
	}
	profiles := make([]PublicProfile, len(rows))
	for i, row := range rows {
		profiles[i] = row.toPublicProfile()
	}
	return profiles, total, nil
}

func (r *gormRepository) FindPublic(ctx context.Context, userID uuid.UUID) (*PublicProfile, error) {
	var row publicProfileRow
	err := r.publicProfiles(ctx).Where("hero_profiles.user_id = ?", userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("hero : lecture du profil public : %w", err)
	}
	profile := row.toPublicProfile()
	return &profile, nil
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
