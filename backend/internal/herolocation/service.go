// Package herolocation gère la position actuelle des héros, partagée volontairement et conservée
// temporairement (voir backend/tables/comptes.md, §« hero_locations »). Elle sert uniquement à choisir les
// héros à prévenir d'un incident : aucune route ne la relit.
package herolocation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// ErrConsentRequired : le héros n'a pas consenti au partage de sa position.
var ErrConsentRequired = errors.New("herolocation : consentement requis")

// ErrNotHero : aucun profil de héros pour ce compte.
var ErrNotHero = errors.New("herolocation : profil de héros introuvable")

const (
	tableName = "hero_locations"
	// coordinateScale arrondit à 3 décimales (~110 m) : suffisant pour un rayon de plusieurs kilomètres,
	// et on ne conserve pas plus de précision que nécessaire.
	coordinateScale = 1000
	// kmPerDegree est la longueur d'un degré de latitude.
	kmPerDegree = 111.0
	// maxAccuracyM écarte les valeurs aberrantes de précision.
	maxAccuracyM = 1_000_000
)

// Input est une position déclarée par un héros (valeurs brutes, non validées).
type Input struct {
	Latitude  float64
	Longitude float64
	AccuracyM *int
}

type record struct {
	HeroID    uuid.UUID `gorm:"primaryKey;column:hero_id"`
	Latitude  float64
	Longitude float64
	AccuracyM *int      `gorm:"column:accuracy_m"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (record) TableName() string { return tableName }

// Service porte les règles de la localisation des héros.
type Service struct {
	db    *gorm.DB
	clock clock.Clock
	ttl   time.Duration
	// radiusKm est le rayon d'alerte autour d'un incident.
	radiusKm int
}

// NewService crée le service.
func NewService(db *gorm.DB, clk clock.Clock, ttl time.Duration, radiusKm int) *Service {
	return &Service{db: db, clock: clk, ttl: ttl, radiusKm: radiusKm}
}

// Set enregistre la position du héros : elle remplace la précédente (aucun historique).
func (s *Service) Set(ctx context.Context, heroID uuid.UUID, in Input) error {
	var fields validation.Collector
	if math.IsNaN(in.Latitude) || in.Latitude < -90 || in.Latitude > 90 {
		fields.Add("latitude", validation.CodeInvalid)
	}
	if math.IsNaN(in.Longitude) || in.Longitude < -180 || in.Longitude > 180 {
		fields.Add("longitude", validation.CodeInvalid)
	}
	if in.AccuracyM != nil && (*in.AccuracyM < 0 || *in.AccuracyM > maxAccuracyM) {
		fields.Add("accuracyM", validation.CodeInvalid)
	}
	if err := fields.Err(); err != nil {
		return err
	}

	row := record{
		HeroID:    heroID,
		Latitude:  math.Round(in.Latitude*coordinateScale) / coordinateScale,
		Longitude: math.Round(in.Longitude*coordinateScale) / coordinateScale,
		AccuracyM: in.AccuracyM,
		UpdatedAt: s.clock.Now(),
	}
	// Le consentement est lu et verrouillé (FOR SHARE) dans la même requête que l'écriture : un retrait
	// concurrent attend la fin de l'écriture, puis supprime la position. Une position ne survit donc jamais
	// à un retrait, quel que soit l'ordre d'arrivée des requêtes.
	result := database.Conn(ctx, s.db).Exec(`
		INSERT INTO hero_locations (hero_id, latitude, longitude, accuracy_m, updated_at)
		SELECT user_id, ?, ?, ?, ? FROM hero_profiles
		WHERE user_id = ? AND location_consent_at IS NOT NULL FOR SHARE
		ON CONFLICT (hero_id) DO UPDATE SET
			latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
			accuracy_m = EXCLUDED.accuracy_m, updated_at = EXCLUDED.updated_at`,
		row.Latitude, row.Longitude, row.AccuracyM, row.UpdatedAt, heroID)
	if result.Error != nil {
		return fmt.Errorf("herolocation : enregistrement : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConsentRequired
	}
	return nil
}

// Consent enregistre le consentement du héros au partage de sa position. Déjà consenti : la date d'origine
// est conservée.
func (s *Service) Consent(ctx context.Context, heroID uuid.UUID) error {
	result := database.Conn(ctx, s.db).Table("hero_profiles").
		Where("user_id = ?", heroID).
		Update("location_consent_at", gorm.Expr("COALESCE(location_consent_at, ?)", s.clock.Now()))
	if result.Error != nil {
		return fmt.Errorf("herolocation : consentement : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotHero
	}
	return nil
}

// Revoke retire le consentement et supprime la position, dans la même transaction.
func (s *Service) Revoke(ctx context.Context, heroID uuid.UUID) error {
	err := database.Conn(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("hero_profiles").Where("user_id = ?", heroID).
			Update("location_consent_at", nil).Error; err != nil {
			return err
		}
		return tx.Where("hero_id = ?", heroID).Delete(&record{}).Error
	})
	if err != nil {
		return fmt.Errorf("herolocation : retrait du consentement : %w", err)
	}
	return nil
}

// Clear supprime la position du héros en gardant son consentement. Sans position, ne fait rien.
func (s *Service) Clear(ctx context.Context, heroID uuid.UUID) error {
	if err := database.Conn(ctx, s.db).Where("hero_id = ?", heroID).Delete(&record{}).Error; err != nil {
		return fmt.Errorf("herolocation : suppression : %w", err)
	}
	return nil
}

// PurgeStale supprime les positions plus anciennes que le délai de péremption.
func (s *Service) PurgeStale(ctx context.Context) (int64, error) {
	result := database.Conn(ctx, s.db).Where("updated_at < ?", s.clock.Now().Add(-s.ttl)).Delete(&record{})
	if result.Error != nil {
		return 0, fmt.Errorf("herolocation : purge : %w", result.Error)
	}
	return result.RowsAffected, nil
}

// EmailsNear retourne les adresses des héros actifs dont la position est fraîche et à moins du rayon
// d'alerte du point donné.
func (s *Service) EmailsNear(ctx context.Context, latitude, longitude float64) ([]string, error) {
	radius := float64(s.radiusKm)
	latDelta := radius / kmPerDegree
	query := database.Conn(ctx, s.db).Table(tableName+" AS hl").
		Joins("JOIN users u ON u.id = hl.hero_id").
		Joins("JOIN hero_profiles hp ON hp.user_id = hl.hero_id AND hp.location_consent_at IS NOT NULL").
		Where("u.role = 'hero' AND u.status = 'active' AND hl.updated_at >= ?", s.clock.Now().Add(-s.ttl)).
		Where("hl.latitude BETWEEN ? AND ?", latitude-latDelta, latitude+latDelta)
	// Le pré-filtre en longitude est ignoré près des pôles et de l'antiméridien, où le rectangle n'a plus de sens.
	if lonDelta := radius / (kmPerDegree * math.Cos(latitude*math.Pi/180)); math.Abs(latitude) < 85 &&
		longitude-lonDelta > -180 && longitude+lonDelta < 180 {
		query = query.Where("hl.longitude BETWEEN ? AND ?", longitude-lonDelta, longitude+lonDelta)
	}
	var emails []string
	err := query.
		Where(`2 * 6371 * asin(least(1, sqrt(
			power(sin(radians(hl.latitude::float8 - ?::float8) / 2), 2) +
			cos(radians(?::float8)) * cos(radians(hl.latitude::float8)) *
			power(sin(radians(hl.longitude::float8 - ?::float8) / 2), 2)))) <= ?::float8`, latitude, latitude, longitude, radius).
		Order("u.email").
		Pluck("u.email", &emails).Error
	if err != nil {
		return nil, fmt.Errorf("herolocation : héros à proximité : %w", err)
	}
	return emails, nil
}
