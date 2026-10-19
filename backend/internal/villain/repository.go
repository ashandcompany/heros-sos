// Package villain gère les vilains, créés par les utilisateurs au fil des déclarations.
package villain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableVillains        = "villains"
	aliasUniqueConstrain = "villains_alias_key"
	likeEscaper          = `\`

	// defaultThreatLevel est le niveau d'un vilain à sa création (valeur autorisée par la base).
	defaultThreatLevel = "faible"
)

var (
	// ErrNotFound indique qu'aucun vilain ne porte cet identifiant.
	ErrNotFound = errors.New("villain : vilain introuvable")
	// ErrAliasTaken indique qu'un autre vilain porte déjà ce pseudo.
	ErrAliasTaken = errors.New("villain : pseudo déjà utilisé")
)

// Villain est un vilain connu de la plateforme.
type Villain struct {
	ID          uuid.UUID
	Alias       string
	Description string
	ThreatLevel string
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
}

type record struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Alias       string
	Description string
	ThreatLevel string
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (record) TableName() string { return tableVillains }

func (r record) toVillain() Villain {
	return Villain{
		ID: r.ID, Alias: r.Alias, Description: r.Description, ThreatLevel: r.ThreatLevel,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
}

// Repository est le stockage des vilains.
type Repository interface {
	Create(ctx context.Context, v *Villain) error
	Find(ctx context.Context, id uuid.UUID) (*Villain, error)
	// List retourne une page de vilains par pseudo ; aliasPrefix, s'il n'est pas vide, restreint
	// la liste aux pseudos qui commencent par ce texte (sans tenir compte de la casse).
	List(ctx context.Context, aliasPrefix string, limit, offset int) ([]Villain, int64, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Create(ctx context.Context, v *Villain) error {
	rec := record{
		ID: v.ID, Alias: v.Alias, Description: v.Description, ThreatLevel: defaultThreatLevel,
		CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, UpdatedAt: v.CreatedAt,
	}
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsUniqueViolation(err, aliasUniqueConstrain) {
			return ErrAliasTaken
		}
		return fmt.Errorf("villain : création : %w", err)
	}
	return nil
}

func (r *gormRepository) Find(ctx context.Context, id uuid.UUID) (*Villain, error) {
	var found record
	err := r.conn(ctx).Where("id = ?", id).Take(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("villain : lecture : %w", err)
	}
	v := found.toVillain()
	return &v, nil
}

func (r *gormRepository) List(ctx context.Context, aliasPrefix string, limit, offset int) ([]Villain, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Model(&record{})
		if aliasPrefix != "" {
			query = query.Where("alias LIKE ?", escapeLike(aliasPrefix)+"%")
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("villain : comptage : %w", err)
	}
	var records []record
	if err := filtered().Order("alias ASC, id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("villain : liste : %w", err)
	}
	villains := make([]Villain, len(records))
	for i, rec := range records {
		villains[i] = rec.toVillain()
	}
	return villains, total, nil
}

// escapeLike neutralise les caractères joker de LIKE (l'échappement par défaut de PostgreSQL
// est l'antislash).
func escapeLike(s string) string {
	return strings.NewReplacer(likeEscaper, likeEscaper+likeEscaper, "%", likeEscaper+"%", "_", likeEscaper+"_").Replace(s)
}
