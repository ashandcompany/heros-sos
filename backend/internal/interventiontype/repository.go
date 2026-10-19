// Package interventiontype gère le catalogue des types d'intervention. Les modérateurs le
// tiennent à jour ; les utilisateurs y choisissent le type d'un incident déclaré. Un type n'est
// jamais supprimé, seulement désactivé : les incidents existants le référencent toujours.
package interventiontype

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
	tableInterventionTypes = "intervention_types"
	labelUniqueConstraint  = "intervention_types_label_key"
)

var (
	// ErrNotFound indique qu'aucun type ne porte cet identifiant.
	ErrNotFound = errors.New("interventiontype : type introuvable")
	// ErrLabelTaken indique qu'un autre type porte déjà ce libellé.
	ErrLabelTaken = errors.New("interventiontype : libellé déjà utilisé")
)

// InterventionType est une catégorie d'intervention.
type InterventionType struct {
	ID          uuid.UUID
	Label       string
	Description string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type record struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Label       string
	Description string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (record) TableName() string { return tableInterventionTypes }

func (r record) toType() InterventionType { return InterventionType(r) }

// Repository est le stockage des types d'intervention.
type Repository interface {
	Create(ctx context.Context, t *InterventionType) error
	Find(ctx context.Context, id uuid.UUID) (*InterventionType, error)
	// List retourne une page de types, par libellé ; onlyActive masque les types désactivés.
	List(ctx context.Context, onlyActive bool, limit, offset int) ([]InterventionType, int64, error)
	// Deactivate retire un type du choix. Il retourne false si le type était déjà désactivé.
	Deactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Create(ctx context.Context, t *InterventionType) error {
	rec := record(*t)
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsUniqueViolation(err, labelUniqueConstraint) {
			return ErrLabelTaken
		}
		return fmt.Errorf("interventiontype : création : %w", err)
	}
	return nil
}

func (r *gormRepository) Find(ctx context.Context, id uuid.UUID) (*InterventionType, error) {
	var rec record
	err := r.conn(ctx).Where("id = ?", id).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("interventiontype : lecture : %w", err)
	}
	found := rec.toType()
	return &found, nil
}

func (r *gormRepository) List(ctx context.Context, onlyActive bool, limit, offset int) ([]InterventionType, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Model(&record{})
		if onlyActive {
			query = query.Where("active")
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("interventiontype : comptage : %w", err)
	}
	var records []record
	if err := filtered().Order("label ASC, id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("interventiontype : liste : %w", err)
	}
	types := make([]InterventionType, len(records))
	for i, rec := range records {
		types[i] = rec.toType()
	}
	return types, total, nil
}

func (r *gormRepository) Deactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	result := r.conn(ctx).Table(tableInterventionTypes).
		Where("id = ? AND active", id).
		Updates(map[string]any{"active": false, "updated_at": now})
	if result.Error != nil {
		return false, fmt.Errorf("interventiontype : désactivation : %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	if _, err := r.Find(ctx, id); err != nil {
		return false, err
	}
	return false, nil
}
