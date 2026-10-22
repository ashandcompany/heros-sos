package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
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
	Moderation        string
	ReporterRole      *string
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

type activeInterventionRecord struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	IncidentID uuid.UUID
	Status     string
	StartedAt  *time.Time
	EndedAt    *time.Time
}

func (activeInterventionRecord) TableName() string { return "interventions" }

type interventionHeroRecord struct {
	InterventionID uuid.UUID `gorm:"primaryKey"`
	HeroID         uuid.UUID `gorm:"primaryKey"`
}

func (interventionHeroRecord) TableName() string { return "intervention_heroes" }

func toRecord(in *Incident) record {
	reporterID := in.ReporterID
	var complement *string
	if in.AddressComplement != "" {
		complement = &in.AddressComplement
	}
	var reporterRole *string
	if in.ReporterRole != nil {
		value := string(*in.ReporterRole)
		reporterRole = &value
	}
	return record{
		ID: in.ID, ReporterID: &reporterID, TypeID: in.TypeID, Title: in.Title, Description: in.Description,
		Status: string(in.Status), Priority: string(in.Priority), Moderation: string(in.Moderation),
		ReporterRole: reporterRole, CitizensRescued: in.CitizensRescued,
		Street: in.Street, PostalCode: in.PostalCode, City: in.City, Country: in.Country,
		AddressComplement: complement, Latitude: in.Latitude, Longitude: in.Longitude,
		OccurredAt: in.OccurredAt, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt,
	}
}

// Repository est le stockage des incidents.
type Repository interface {
	Create(ctx context.Context, in *Incident) error
	Find(ctx context.Context, id uuid.UUID) (*Incident, error)
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]Incident, int64, error)
	SetModeration(ctx context.Context, id uuid.UUID, status ModerationStatus, now time.Time) error
	Update(ctx context.Context, in *Incident, heroID *uuid.UUID, now time.Time) error
	SetPriority(ctx context.Context, id uuid.UUID, priority Priority, now time.Time) error
	PublicHeroExists(ctx context.Context, heroID uuid.UUID) (bool, error)
	ListResolvedByHero(ctx context.Context, heroID uuid.UUID, limit, offset int) ([]Summary, int64, error)
	ListByVillain(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]Summary, int64, error)
}

// ListFilter restreint la liste des incidents. Les valeurs d'un même critère sont combinées
// par OU, les critères entre eux par ET.
type ListFilter struct {
	ReporterID   *uuid.UUID
	Statuses     []Status
	HeroIDs      []uuid.UUID
	VillainIDs   []uuid.UUID
	ApprovedOnly bool
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

func (r *gormRepository) SetModeration(ctx context.Context, id uuid.UUID, status ModerationStatus, now time.Time) error {
	result := r.conn(ctx).Table(tableIncidents).Where("id = ?", id).
		Updates(map[string]any{"moderation": string(status), "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("incident : modération : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) Update(ctx context.Context, in *Incident, heroID *uuid.UUID, now time.Time) error {
	if err := r.syncIntervention(ctx, in.ID, in.Status, heroID, now); err != nil {
		return err
	}
	values := map[string]any{
		"type_id": in.TypeID, "title": in.Title, "description": in.Description, "status": string(in.Status),
		"street": in.Street, "city": in.City, "latitude": in.Latitude, "longitude": in.Longitude,
		"updated_at": now,
	}
	result := r.conn(ctx).Table(tableIncidents).Where("id = ?", in.ID).Updates(values)
	if result.Error != nil {
		return fmt.Errorf("incident : correction : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	if err := r.conn(ctx).Where("incident_id = ?", in.ID).Delete(&incidentVillainRecord{}).Error; err != nil {
		return fmt.Errorf("incident : retrait des vilains : %w", err)
	}
	for _, villainID := range in.VillainIDs {
		link := incidentVillainRecord{IncidentID: in.ID, VillainID: villainID}
		if err := r.conn(ctx).Create(&link).Error; err != nil {
			return fmt.Errorf("incident : association du vilain : %w", err)
		}
	}
	return nil
}

func (r *gormRepository) syncIntervention(ctx context.Context, incidentID uuid.UUID, status Status, heroID *uuid.UUID, now time.Time) error {
	var active activeInterventionRecord
	err := r.conn(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("incident_id = ? AND status = ?", incidentID, "in_progress").Take(&active).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("incident : lecture de l'intervention active : %w", err)
	}
	hasActive := err == nil

	if status == StatusReported {
		if hasActive {
			return r.closeIntervention(ctx, active.ID, "failed", now)
		}
		return nil
	}
	if heroID == nil {
		return &validation.Error{Fields: map[string]string{"heroId": "required"}}
	}

	if !hasActive {
		active = activeInterventionRecord{
			ID: ids.New(), IncidentID: incidentID, Status: "in_progress", StartedAt: &now,
		}
		if status == StatusSucceeded || status == StatusFailed {
			active.Status, active.EndedAt = string(status), &now
		}
		if err := r.conn(ctx).Create(&active).Error; err != nil {
			return fmt.Errorf("incident : création de l'intervention corrigée : %w", err)
		}
	} else if status == StatusSucceeded || status == StatusFailed {
		if err := r.conn(ctx).Model(&active).Updates(map[string]any{
			"status": string(status), "ended_at": now, "proposed_status": nil, "proposed_by": nil,
		}).Error; err != nil {
			return fmt.Errorf("incident : clôture de l'intervention corrigée : %w", err)
		}
	}

	if err := r.conn(ctx).Table("intervention_heroes").
		Where("intervention_id = ? AND role = ? AND hero_id <> ?", active.ID, "lead", *heroID).
		Delete(&interventionHeroRecord{}).Error; err != nil {
		return fmt.Errorf("incident : retrait de l'ancien lead : %w", err)
	}
	if err := r.conn(ctx).Table("intervention_heroes").
		Where("intervention_id = ? AND role = ?", active.ID, "lead").
		Update("role", "member").Error; err != nil {
		return fmt.Errorf("incident : changement du lead : %w", err)
	}
	participant := map[string]any{
		"role": "lead", "participation": "confirmed", "outcome_validation": "pending",
		"is_active": status == StatusInProgress, "joined_at": now,
	}
	result := r.conn(ctx).Table("intervention_heroes").
		Where("intervention_id = ? AND hero_id = ?", active.ID, *heroID).Updates(participant)
	if result.Error != nil {
		if database.IsUniqueViolation(result.Error, "intervention_heroes_one_active_intervention_idx") {
			return ErrNotAvailable
		}
		return fmt.Errorf("incident : mise à jour du lead : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		lead := map[string]any{
			"intervention_id": active.ID, "hero_id": *heroID, "role": "lead",
			"participation": "confirmed", "outcome_validation": "pending",
			"is_active": status == StatusInProgress, "joined_at": now,
		}
		if err := r.conn(ctx).Table("intervention_heroes").Create(lead).Error; err != nil {
			if database.IsUniqueViolation(err, "intervention_heroes_one_active_intervention_idx") {
				return ErrNotAvailable
			}
			return fmt.Errorf("incident : ajout du lead : %w", err)
		}
	}
	if status != StatusInProgress {
		return r.closeIntervention(ctx, active.ID, string(status), now)
	}
	return nil
}

func (r *gormRepository) closeIntervention(ctx context.Context, id uuid.UUID, status string, now time.Time) error {
	if err := r.conn(ctx).Table("interventions").Where("id = ?", id).
		Updates(map[string]any{"status": status, "ended_at": now, "proposed_status": nil, "proposed_by": nil}).Error; err != nil {
		return fmt.Errorf("incident : clôture de l'intervention : %w", err)
	}
	if err := r.conn(ctx).Table("intervention_heroes").Where("intervention_id = ?", id).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("incident : désactivation des participants : %w", err)
	}
	return nil
}

func (r *gormRepository) SetPriority(ctx context.Context, id uuid.UUID, priority Priority, now time.Time) error {
	result := r.conn(ctx).Table(tableIncidents).
		Where(`id = ? AND moderation = ? AND status IN (?, ?) AND NOT EXISTS (
			SELECT 1 FROM interventions WHERE interventions.incident_id = incidents.id AND interventions.status = ?
		)`, id, ModerationApproved, StatusReported, StatusFailed, "in_progress").
		Updates(map[string]any{"priority": string(priority), "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("incident : priorité : %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var exists bool
	if err := r.conn(ctx).Raw("SELECT EXISTS (SELECT 1 FROM incidents WHERE id = ?)", id).Scan(&exists).Error; err != nil {
		return fmt.Errorf("incident : vérification avant priorité : %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrNotAvailable
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

func (r *gormRepository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Incident, int64, error) {
	query := r.incidentQuery(ctx)
	if filter.ReporterID != nil {
		query = query.Where("incidents.reporter_id = ?", *filter.ReporterID)
	}
	if filter.ApprovedOnly {
		query = query.Where("incidents.moderation = ?", ModerationApproved)
	}
	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statuses[i] = string(status)
		}
		query = query.Where("incidents.status IN ?", statuses)
	}
	if len(filter.HeroIDs) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM interventions i
			JOIN intervention_heroes ih ON ih.intervention_id = i.id
			WHERE i.incident_id = incidents.id AND ih.hero_id IN ?)`, filter.HeroIDs)
	}
	if len(filter.VillainIDs) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM incident_villains iv
			WHERE iv.incident_id = incidents.id AND iv.villain_id IN ?)`, filter.VillainIDs)
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
			Moderation:      ModerationStatus(rec.Moderation),
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
		if rec.ReporterRole != nil {
			role := ReporterRole(*rec.ReporterRole)
			item.ReporterRole = &role
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
