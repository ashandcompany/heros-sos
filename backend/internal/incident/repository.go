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

const (
	tableIncidents = "incidents"
	// incidentPrimaryKey est la contrainte violée quand l'identifiant fourni par le client existe déjà.
	incidentPrimaryKey = "incidents_pkey"
)

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
	List(ctx context.Context, filter ListFilter, limit, offset int) (ListResult, error)
	PublicHeroExists(ctx context.Context, heroID uuid.UUID) (bool, error)
	ListResolvedByHero(ctx context.Context, heroID uuid.UUID, limit, offset int) ([]Summary, int64, error)
	ListByVillain(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]Summary, int64, error)
}

// ListFilter restreint la liste des incidents. Les valeurs d'un même critère sont combinées
// par OU, les critères entre eux par ET.
type ListFilter struct {
	ReporterID *uuid.UUID
	Statuses   []Status
	HeroIDs    []uuid.UUID
	VillainIDs []uuid.UUID
	// After reprend la liste après cet incident (pagination par curseur) ; il remplace le décalage.
	After *Cursor
}

// ListResult est une page d'incidents : le total correspondant au filtre (hors curseur) et, s'il reste des
// incidents après elle, la position de la page suivante.
type ListResult struct {
	Items []Incident
	Total int64
	Next  *Cursor
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
		if database.IsUniqueViolation(err, incidentPrimaryKey) {
			return ErrAlreadyExists
		}
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

func (r *gormRepository) List(ctx context.Context, filter ListFilter, limit, offset int) (ListResult, error) {
	total, err := r.total(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}
	query := r.filteredQuery(ctx, filter)
	if filter.After != nil {
		// Comparaison de ligne : PostgreSQL la lit directement dans l'index (created_at, id).
		query = query.Where("(incidents.created_at, incidents.id) < (?, ?)", filter.After.CreatedAt, filter.After.ID)
	}
	var rows []joinedRecord
	// Une ligne de plus que demandé : si elle existe, il y a une page suivante, sans compter pour le savoir.
	err = query.Order("incidents.created_at DESC, incidents.id DESC").Limit(limit + 1).Offset(offset).Find(&rows).Error
	if err != nil {
		return ListResult{}, fmt.Errorf("incident : liste : %w", err)
	}
	var next *Cursor
	if len(rows) > limit {
		rows = rows[:limit]
		last := Cursor{CreatedAt: rows[limit-1].Record.CreatedAt, ID: rows[limit-1].Record.ID}
		next = &last
	}
	items, err := r.hydrate(ctx, rows)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total, Next: next}, nil
}

// filteredQuery applique les critères du filtre (sans le curseur, qui ne restreint que la page).
func (r *gormRepository) filteredQuery(ctx context.Context, filter ListFilter) *gorm.DB {
	query := r.incidentQuery(ctx)
	if filter.ReporterID != nil {
		query = query.Where("incidents.reporter_id = ?", *filter.ReporterID)
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("incidents.status IN ?", statusNames(filter.Statuses))
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
	return query
}

func statusNames(statuses []Status) []string {
	names := make([]string, len(statuses))
	for i, status := range statuses {
		names[i] = string(status)
	}
	return names
}

// total retourne le nombre d'incidents du filtre. Compter 5 millions de lignes coûte plusieurs centaines de
// millisecondes, et un citoyen très actif en a des centaines de milliers : sans héros ni vilain dans le filtre, le
// total se lit dans les compteurs (incident_stats, ou reporter_stats pour un déclarant, somme des statuts
// demandés), et un vilain seul dans villain_stats. Les autres filtres se comptent directement.
func (r *gormRepository) total(ctx context.Context, filter ListFilter) (int64, error) {
	var total int64
	switch {
	case len(filter.HeroIDs) == 0 && len(filter.VillainIDs) == 0:
		// Sans héros ni vilain : un compteur par statut, global (incident_stats) ou celui du déclarant (reporter_stats).
		query := r.conn(ctx).Table("incident_stats").Select("COALESCE(SUM(incidents), 0)")
		if filter.ReporterID != nil {
			query = r.conn(ctx).Table("reporter_stats").Select("COALESCE(SUM(incidents), 0)").Where("reporter_id = ?", *filter.ReporterID)
		}
		if len(filter.Statuses) > 0 {
			query = query.Where("status IN ?", statusNames(filter.Statuses))
		}
		if err := query.Scan(&total).Error; err != nil {
			return 0, fmt.Errorf("incident : total : %w", err)
		}
	case filter.ReporterID == nil && len(filter.HeroIDs) == 0 && len(filter.Statuses) == 0 && len(filter.VillainIDs) == 1:
		err := r.conn(ctx).Table("villain_stats").Select("COALESCE(SUM(incidents), 0)").
			Where("villain_id = ?", filter.VillainIDs[0]).Scan(&total).Error
		if err != nil {
			return 0, fmt.Errorf("incident : total du vilain : %w", err)
		}
	default:
		if err := r.filteredQuery(ctx, filter).Count(&total).Error; err != nil {
			return 0, fmt.Errorf("incident : comptage : %w", err)
		}
	}
	return total, nil
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
