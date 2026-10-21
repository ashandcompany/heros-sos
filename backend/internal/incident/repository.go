package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const tableIncidents = "incidents"

type record struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	ReporterID        *uuid.UUID
	TypeID            int16
	Title             string
	Description       string
	Status            string
	Priority          string
	CitizensRescued   int16
	Street            string
	PostalCode        string
	City              string
	Country           string
	AddressComplement *string
	Latitude          *float64
	Longitude         *float64
	OccurredAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type joinedRecord struct {
	Record    record `gorm:"embedded"`
	TypeCode  string
	TypeLabel string
}

func (record) TableName() string { return tableIncidents }

type incidentVillainRecord struct {
	IncidentID uuid.UUID `gorm:"primaryKey"`
	VillainID  uuid.UUID `gorm:"primaryKey"`
}

func (incidentVillainRecord) TableName() string { return "incident_villains" }

func toRecord(in *Incident) record {
	reporterID := in.ReporterID
	var complement *string
	if in.AddressComplement != "" {
		complement = &in.AddressComplement
	}
	return record{
		ID: in.ID, ReporterID: &reporterID, TypeID: in.TypeID, Title: in.Title, Description: in.Description,
		Status: string(in.Status), Priority: string(in.Priority), CitizensRescued: in.CitizensRescued,
		Street: in.Street, PostalCode: in.PostalCode, City: in.City, Country: in.Country,
		AddressComplement: complement, Latitude: in.Latitude, Longitude: in.Longitude,
		OccurredAt: in.OccurredAt, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt,
	}
}

// Repository est le stockage des incidents.
type Repository interface {
	Create(ctx context.Context, in *Incident) error
	Find(ctx context.Context, id uuid.UUID) (*Incident, error)
	List(ctx context.Context, reporterID *uuid.UUID, limit, offset int) ([]Incident, int64, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Create(ctx context.Context, in *Incident) error {
	rec := toRecord(in)
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("incident : création : %w", err)
	}
	for _, villainID := range in.VillainIDs {
		link := incidentVillainRecord{IncidentID: in.ID, VillainID: villainID}
		if err := r.conn(ctx).Create(&link).Error; err != nil {
			return fmt.Errorf("incident : liaison du vilain %s : %w", villainID, err)
		}
	}
	return nil
}

func (r *gormRepository) Find(ctx context.Context, id uuid.UUID) (*Incident, error) {
	var joined joinedRecord
	err := r.incidentQuery(ctx).Where("incidents.id = ?", id).Take(&joined).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("incident : lecture : %w", err)
	}
	items, err := r.hydrate(ctx, []joinedRecord{joined})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *gormRepository) List(ctx context.Context, reporterID *uuid.UUID, limit, offset int) ([]Incident, int64, error) {
	query := r.incidentQuery(ctx)
	if reporterID != nil {
		query = query.Where("incidents.reporter_id = ?", *reporterID)
	} else {
		query = query.Where("incidents.status IN ?", []string{"reported", "in_progress", "failed"})
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("incident : comptage : %w", err)
	}
	var rows []joinedRecord
	if err := query.Order("incidents.created_at DESC, incidents.id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("incident : liste : %w", err)
	}
	items, err := r.hydrate(ctx, rows)
	return items, total, err
}

func (r *gormRepository) incidentQuery(ctx context.Context) *gorm.DB {
	return r.conn(ctx).Table(tableIncidents).
		Select("incidents.*, incident_types.code AS type_code, incident_types.label AS type_label").
		Joins("JOIN incident_types ON incident_types.id = incidents.type_id")
}

func (r *gormRepository) hydrate(ctx context.Context, rows []joinedRecord) ([]Incident, error) {
	items := make([]Incident, len(rows))
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		rec := row.Record
		item := Incident{
			ID: rec.ID, TypeID: rec.TypeID, TypeCode: row.TypeCode, TypeLabel: row.TypeLabel, Title: rec.Title,
			Description: rec.Description, Status: Status(rec.Status), Priority: Priority(rec.Priority),
			CitizensRescued: rec.CitizensRescued, Street: rec.Street, PostalCode: rec.PostalCode,
			City: rec.City, Country: rec.Country, Latitude: rec.Latitude, Longitude: rec.Longitude,
			OccurredAt: rec.OccurredAt, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
		}
		if rec.ReporterID != nil {
			item.ReporterID = *rec.ReporterID
		}
		if rec.AddressComplement != nil {
			item.AddressComplement = *rec.AddressComplement
		}
		items[i], ids[i] = item, rec.ID
	}
	if len(ids) == 0 {
		return items, nil
	}
	var links []struct {
		IncidentID uuid.UUID
		VillainID  uuid.UUID
		Alias      string
	}
	if err := r.conn(ctx).Table("incident_villains").
		Select("incident_villains.incident_id, villains.id AS villain_id, villains.alias").
		Joins("JOIN villains ON villains.id = incident_villains.villain_id").
		Where("incident_villains.incident_id IN ?", ids).Order("villains.alias").Scan(&links).Error; err != nil {
		return nil, fmt.Errorf("incident : lecture des vilains : %w", err)
	}
	byIncident := make(map[uuid.UUID]int, len(items))
	for i := range items {
		byIncident[items[i].ID] = i
		items[i].VillainIDs = []uuid.UUID{}
		items[i].Villains = []VillainReference{}
	}
	for _, link := range links {
		index := byIncident[link.IncidentID]
		items[index].VillainIDs = append(items[index].VillainIDs, link.VillainID)
		items[index].Villains = append(items[index].Villains, VillainReference{ID: link.VillainID, Alias: link.Alias})
	}
	var active []struct {
		IncidentID uuid.UUID
		ID         uuid.UUID
	}
	if err := r.conn(ctx).Table("interventions").
		Select("incident_id, id").Where("incident_id IN ? AND status = ?", ids, "in_progress").
		Find(&active).Error; err != nil {
		return nil, fmt.Errorf("incident : lecture de l'intervention active : %w", err)
	}
	for _, row := range active {
		interventionID := row.ID
		items[byIncident[row.IncidentID]].InterventionID = &interventionID
	}
	return items, nil
}
