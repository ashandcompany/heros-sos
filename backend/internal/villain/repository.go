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
	LastName    string
	FirstName   string
	MainSector  string
	Description string
	ThreatLevel ThreatLevel
	Status      Status
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TypeIDs     []int16
}

type record struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Alias       string
	LastName    *string
	FirstName   *string
	MainSector  *string
	Description *string
	ThreatLevel string
	Status      string
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (record) TableName() string { return tableVillains }

type specialtyRecord struct {
	VillainID uuid.UUID `gorm:"primaryKey"`
	TypeID    int16     `gorm:"primaryKey"`
}

func (specialtyRecord) TableName() string { return "villain_specialties" }

func (r record) toVillain() Villain {
	v := Villain{
		ID: r.ID, Alias: r.Alias, ThreatLevel: ThreatLevel(r.ThreatLevel), Status: Status(r.Status),
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.LastName != nil {
		v.LastName = *r.LastName
	}
	if r.FirstName != nil {
		v.FirstName = *r.FirstName
	}
	if r.MainSector != nil {
		v.MainSector = *r.MainSector
	}
	if r.Description != nil {
		v.Description = *r.Description
	}
	return v
}

// Repository est le stockage des vilains.
type Repository interface {
	Create(ctx context.Context, v *Villain) error
	Find(ctx context.Context, id uuid.UUID) (*Villain, error)
	Update(ctx context.Context, v *Villain) error
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
		ID: v.ID, Alias: v.Alias, LastName: optional(v.LastName), FirstName: optional(v.FirstName),
		MainSector: optional(v.MainSector), Description: optional(v.Description),
		ThreatLevel: string(v.ThreatLevel), Status: string(v.Status),
		CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsUniqueViolation(err, aliasUniqueConstrain) {
			return ErrAliasTaken
		}
		return fmt.Errorf("villain : création : %w", err)
	}
	for _, typeID := range v.TypeIDs {
		link := specialtyRecord{VillainID: v.ID, TypeID: typeID}
		if err := r.conn(ctx).Create(&link).Error; err != nil {
			return fmt.Errorf("villain : spécialité %d : %w", typeID, err)
		}
	}
	return nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
	typeIDs, err := r.typeIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	v.TypeIDs = typeIDs[id]
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
	ids := make([]uuid.UUID, len(records))
	for i, rec := range records {
		ids[i] = rec.ID
	}
	typeIDs, err := r.typeIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range villains {
		villains[i].TypeIDs = typeIDs[villains[i].ID]
	}
	return villains, total, nil
}

func (r *gormRepository) typeIDs(ctx context.Context, villainIDs []uuid.UUID) (map[uuid.UUID][]int16, error) {
	result := make(map[uuid.UUID][]int16, len(villainIDs))
	for _, id := range villainIDs {
		result[id] = []int16{}
	}
	if len(villainIDs) == 0 {
		return result, nil
	}
	var rows []specialtyRecord
	if err := r.conn(ctx).Where("villain_id IN ?", villainIDs).Order("villain_id, type_id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("villain : lecture des spécialités : %w", err)
	}
	for _, row := range rows {
		result[row.VillainID] = append(result[row.VillainID], row.TypeID)
	}
	return result, nil
}

func (r *gormRepository) Update(ctx context.Context, v *Villain) error {
	result := r.conn(ctx).Table(tableVillains).Where("id = ?", v.ID).Updates(map[string]any{
		"alias": v.Alias, "last_name": optional(v.LastName), "first_name": optional(v.FirstName),
		"main_sector": optional(v.MainSector), "description": optional(v.Description),
		"threat_level": string(v.ThreatLevel), "status": string(v.Status), "updated_at": v.UpdatedAt,
	})
	if result.Error != nil {
		if database.IsUniqueViolation(result.Error, aliasUniqueConstrain) {
			return ErrAliasTaken
		}
		return fmt.Errorf("villain : mise à jour : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if err := r.conn(ctx).Where("villain_id = ?", v.ID).Delete(&specialtyRecord{}).Error; err != nil {
		return fmt.Errorf("villain : remplacement des spécialités : %w", err)
	}
	for _, typeID := range v.TypeIDs {
		link := specialtyRecord{VillainID: v.ID, TypeID: typeID}
		if err := r.conn(ctx).Create(&link).Error; err != nil {
			return fmt.Errorf("villain : spécialité %d : %w", typeID, err)
		}
	}
	return nil
}

// escapeLike neutralise les caractères joker de LIKE (l'échappement par défaut de PostgreSQL
// est l'antislash).
func escapeLike(s string) string {
	return strings.NewReplacer(likeEscaper, likeEscaper+likeEscaper, "%", likeEscaper+"%", "_", likeEscaper+"_").Replace(s)
}
