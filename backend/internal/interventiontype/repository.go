// Package interventiontype gère le catalogue des types d'intervention. Les modérateurs le
// tiennent à jour ; les utilisateurs y choisissent le type d'un incident déclaré. Un type n'est
// jamais supprimé, seulement désactivé : les incidents existants le référencent toujours.
package interventiontype

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableIncidentTypes   = "incident_types"
	codeUniqueConstraint = "incident_types_code_key"
)

var (
	// ErrNotFound indique qu'aucun type ne porte cet identifiant.
	ErrNotFound = errors.New("interventiontype : type introuvable")
	// ErrCodeTaken indique qu'un autre type porte déjà ce code.
	ErrCodeTaken = errors.New("interventiontype : code déjà utilisé")
)

// InterventionType est une catégorie d'intervention.
type InterventionType struct {
	ID            int16
	Code          string
	Label         string
	EndangersLife bool
}

type record struct {
	ID            int16 `gorm:"primaryKey;autoIncrement"`
	Code          string
	Label         string
	EndangersLife bool
}

func (record) TableName() string { return tableIncidentTypes }

func (r record) toType() InterventionType { return InterventionType(r) }

// Repository est le stockage des types d'intervention.
type Repository interface {
	Create(ctx context.Context, t *InterventionType) error
	Find(ctx context.Context, id int16) (*InterventionType, error)
	List(ctx context.Context, limit, offset int) ([]InterventionType, int64, error)
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
		if database.IsUniqueViolation(err, codeUniqueConstraint) {
			return ErrCodeTaken
		}
		return fmt.Errorf("interventiontype : création : %w", err)
	}
	*t = rec.toType()
	return nil
}

func (r *gormRepository) Find(ctx context.Context, id int16) (*InterventionType, error) {
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

func (r *gormRepository) List(ctx context.Context, limit, offset int) ([]InterventionType, int64, error) {
	filtered := r.conn(ctx).Model(&record{})
	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("interventiontype : comptage : %w", err)
	}
	var records []record
	if err := filtered.Order("label ASC, id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("interventiontype : liste : %w", err)
	}
	types := make([]InterventionType, len(records))
	for i, rec := range records {
		types[i] = rec.toType()
	}
	return types, total, nil
}
