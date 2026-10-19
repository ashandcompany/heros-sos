package incident

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const tableIncidents = "incidents"

// Colonnes chiffrées : leur nom entre dans le contexte authentifié (AAD) de chaque valeur.
const (
	columnAddress = "address"
	columnCity    = "city"
)

// Sealer chiffre une valeur pour un contexte donné (*crypto.Sealer).
type Sealer interface {
	Seal(plaintext []byte, aad crypto.AAD) ([]byte, error)
}

type record struct {
	ID                 uuid.UUID `gorm:"primaryKey"`
	ReporterID         uuid.UUID
	ReporterRole       string
	InterventionTypeID uuid.UUID
	VillainID          *uuid.UUID
	Title              string
	Description        string
	Address            []byte
	City               []byte
	OccurredAt         *time.Time
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (record) TableName() string { return tableIncidents }

func aad(column string, id uuid.UUID) crypto.AAD {
	return crypto.AAD{Table: tableIncidents, Column: column, RecordID: id}
}

func toRecord(in *Incident, sealer Sealer) (record, error) {
	address, err := sealer.Seal([]byte(in.Address), aad(columnAddress, in.ID))
	if err != nil {
		return record{}, fmt.Errorf("incident : chiffrement de %s : %w", columnAddress, err)
	}
	city, err := sealer.Seal([]byte(in.City), aad(columnCity, in.ID))
	if err != nil {
		return record{}, fmt.Errorf("incident : chiffrement de %s : %w", columnCity, err)
	}
	return record{
		ID: in.ID, ReporterID: in.ReporterID, ReporterRole: string(in.ReporterRole),
		InterventionTypeID: in.InterventionTypeID, VillainID: in.VillainID, Title: in.Title,
		Description: in.Description, Address: address, City: city, OccurredAt: in.OccurredAt,
		Status: string(in.Status), CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt,
	}, nil
}

// Repository est le stockage des incidents.
type Repository interface {
	Create(ctx context.Context, in *Incident) error
}

type gormRepository struct {
	db     *gorm.DB
	sealer Sealer
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB, sealer Sealer) Repository {
	return &gormRepository{db: db, sealer: sealer}
}

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Create(ctx context.Context, in *Incident) error {
	rec, err := toRecord(in, r.sealer)
	if err != nil {
		return err
	}
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("incident : création : %w", err)
	}
	return nil
}
