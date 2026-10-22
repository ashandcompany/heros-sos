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
	Record               record `gorm:"embedded"`
	TypeCode             string
	TypeLabel            string
	ResolvedAt           *time.Time
	LatestInterventionID *uuid.UUID
}

// latestInterventionJoin rattache à chaque incident sa dernière intervention (alias `latest`) : elle donne le lead.
// Elle ne se calcule que pour les lignes de la page (la jointure vient après le tri et la limite).
const latestInterventionJoin = `LEFT JOIN LATERAL (
	SELECT id, ended_at FROM interventions WHERE interventions.incident_id = incidents.id
	ORDER BY started_at DESC NULLS LAST, id DESC LIMIT 1) AS latest ON TRUE`

// resolvedAtExpression est la date de résolution : fin de la dernière intervention, si elle a réussi. C'est une
// colonne dénormalisée, tenue par trigger (migration add_incident_resolved_at) : la calculer pour trier ou filtrer
// obligeait à joindre la dernière intervention de chaque incident de la table.
const resolvedAtExpression = "incidents.resolved_at"

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
	List(ctx context.Context, filter ListFilter, limit, offset int) (ListResult, error)
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
	// ActiveHeroID ne garde que l'incident sur lequel ce héros intervient en ce moment : sa participation
	// confirmée à l'intervention en cours (un héros n'en a jamais plus d'une). Contrairement à HeroIDs, les
	// interventions terminées ne comptent pas. Critère interne, non exposé par l'API.
	ActiveHeroID *uuid.UUID
	// After reprend la liste après cet incident (pagination par curseur) ; il remplace le décalage. Le curseur est celui
	// de l'ordre par date de déclaration : il ne se combine pas avec le tri par résolution.
	After *Cursor
	// SkipTotal dispense de calculer le total (Total reste à zéro) : pour les appelants qui n'affichent pas de
	// pagination (tableau de bord), un total exact peut coûter un comptage de toute la table.
	SkipTotal bool
	Sort      ListSort
}

// ListResult est une page d'incidents : le total correspondant au filtre (hors curseur) et, s'il reste des
// incidents après elle, la position de la page suivante.
type ListResult struct {
	Items []Incident
	Total int64
	Next  *Cursor
}

// ListSort est l'ordre de la liste des incidents (valeur du paramètre `sort`).
type ListSort string

const (
	// SortCreatedAt : les plus récemment déclarés d'abord (ordre par défaut).
	SortCreatedAt ListSort = "-createdAt"
	// SortResolvedAt : les plus récemment résolus d'abord, les incidents non résolus en dernier.
	SortResolvedAt ListSort = "-resolvedAt"
)

// ParseListSort valide un ordre reçu en paramètre ; vide, l'ordre par défaut.
func ParseListSort(raw string) (ListSort, bool) {
	switch sort := ListSort(raw); sort {
	case "":
		return SortCreatedAt, true
	case SortCreatedAt, SortResolvedAt:
		return sort, true
	}
	return "", false
}

func (s ListSort) orderClause() string {
	if s == SortResolvedAt {
		return resolvedAtExpression + " DESC NULLS LAST, incidents.created_at DESC, incidents.id DESC"
	}
	return "incidents.created_at DESC, incidents.id DESC"
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
	err := withResolution(r.incidentQuery(ctx)).Where("incidents.id = ?", id).Take(&joined).Error
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
		// Le curseur est (date de déclaration, identifiant) : il n'a pas de sens pour un autre ordre.
		if filter.Sort == SortResolvedAt {
			return ListResult{}, ErrInvalidCursor
		}
		// Comparaison de ligne : PostgreSQL la lit directement dans l'index (created_at, id).
		query = query.Where("(incidents.created_at, incidents.id) < (?, ?)", filter.After.CreatedAt, filter.After.ID)
	}
	var rows []joinedRecord
	// Une ligne de plus que demandé : si elle existe, il y a une page suivante, sans compter pour le savoir.
	err = withResolution(query).Order(filter.Sort.orderClause()).Limit(limit + 1).Offset(offset).Find(&rows).Error
	if err != nil {
		return ListResult{}, fmt.Errorf("incident : liste : %w", err)
	}
	var next *Cursor
	if len(rows) > limit {
		rows = rows[:limit]
		// Pas de curseur de page suivante pour le tri par résolution : il se pagine par décalage (offset).
		if filter.Sort != SortResolvedAt {
			last := Cursor{CreatedAt: rows[limit-1].Record.CreatedAt, ID: rows[limit-1].Record.ID}
			next = &last
		}
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
	if filter.ApprovedOnly {
		// En toutes lettres, pas en paramètre : seul un littéral permet d'utiliser les index partiels
		// incidents_approved_*_idx (voir la migration add_incident_perf_indexes).
		query = query.Where("incidents.moderation = 'approved'")
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("incidents.status IN ?", statusNames(filter.Statuses))
	}
	if len(filter.HeroIDs) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM interventions i
			JOIN intervention_heroes ih ON ih.intervention_id = i.id
			WHERE i.incident_id = incidents.id AND ih.hero_id IN ?)`, filter.HeroIDs)
	}
	if filter.ActiveHeroID != nil {
		query = query.Where(`EXISTS (SELECT 1 FROM interventions i
			JOIN intervention_heroes ih ON ih.intervention_id = i.id
			WHERE i.incident_id = incidents.id AND i.status = 'in_progress'
			AND ih.hero_id = ? AND ih.participation = 'confirmed' AND ih.is_active)`, *filter.ActiveHeroID)
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
// millisecondes, et un citoyen très actif en a des centaines de milliers : quand un compteur a exactement la
// définition du filtre, le total s'y lit.
//
//   - incident_stats (par modération, statut et type) : sans déclarant, héros ni vilain ; la modération restreint
//     aux incidents approuvés (listes publiques et héros) ou les compte tous (modérateur) ;
//   - reporter_stats (par déclarant et statut) : « mes incidents », donc tous les états de modération ;
//   - villain_stats : un vilain seul, approuvés seulement, comme son historique public.
//
// Les autres combinaisons (héros, vilain avec autre critère, déclarant restreint aux approuvés, intervention en
// cours d'un héros) se comptent directement.
func (r *gormRepository) total(ctx context.Context, filter ListFilter) (int64, error) {
	if filter.SkipTotal {
		return 0, nil
	}
	plain := len(filter.HeroIDs) == 0 && len(filter.VillainIDs) == 0 && filter.ActiveHeroID == nil
	var total int64
	switch {
	case plain && filter.ReporterID == nil:
		query := r.conn(ctx).Table("incident_stats").Select("COALESCE(SUM(incidents), 0)")
		if filter.ApprovedOnly {
			query = query.Where("moderation = ?", ModerationApproved)
		}
		if len(filter.Statuses) > 0 {
			query = query.Where("status IN ?", statusNames(filter.Statuses))
		}
		if err := query.Scan(&total).Error; err != nil {
			return 0, fmt.Errorf("incident : total : %w", err)
		}
	case plain && !filter.ApprovedOnly:
		query := r.conn(ctx).Table("reporter_stats").Select("COALESCE(SUM(incidents), 0)").Where("reporter_id = ?", *filter.ReporterID)
		if len(filter.Statuses) > 0 {
			query = query.Where("status IN ?", statusNames(filter.Statuses))
		}
		if err := query.Scan(&total).Error; err != nil {
			return 0, fmt.Errorf("incident : total du déclarant : %w", err)
		}
	case filter.ApprovedOnly && filter.ReporterID == nil && filter.ActiveHeroID == nil &&
		len(filter.HeroIDs) == 0 && len(filter.Statuses) == 0 && len(filter.VillainIDs) == 1:
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

// withResolution ajoute la dernière intervention (date de résolution, lead) à une requête
// d'incidents. Appliquée après le comptage : la jointure ne change pas le nombre de lignes.
func withResolution(query *gorm.DB) *gorm.DB {
	return query.Joins(latestInterventionJoin).
		Select(`incidents.*, incident_types.code AS type_code, incident_types.label AS type_label, ` +
			`latest.id AS latest_intervention_id`)
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

	byLatest := make(map[uuid.UUID]int, len(rows))
	for i, row := range rows {
		items[i].ResolvedAt = row.ResolvedAt
		if row.LatestInterventionID != nil {
			byLatest[*row.LatestInterventionID] = i
		}
	}
	leads, err := r.publicLeads(ctx, byLatest)
	if err != nil {
		return nil, err
	}
	for interventionID, lead := range leads {
		items[byLatest[interventionID]].Lead = lead
	}
	return items, nil
}

// publicLeads retourne le lead de chaque intervention demandée (clés de `interventions`). Seuls
// les héros encore publics sont nommés : leur fiche doit exister.
func (r *gormRepository) publicLeads(ctx context.Context, interventions map[uuid.UUID]int) (map[uuid.UUID]*HeroReference, error) {
	if len(interventions) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(interventions))
	for id := range interventions {
		ids = append(ids, id)
	}
	var rows []struct {
		InterventionID uuid.UUID
		HeroID         uuid.UUID
		Alias          string
	}
	if err := r.conn(ctx).Table("intervention_heroes").
		Select("intervention_heroes.intervention_id, hero_profiles.user_id AS hero_id, hero_profiles.alias").
		Joins("JOIN hero_profiles ON hero_profiles.user_id = intervention_heroes.hero_id").
		Joins("JOIN users ON users.id = hero_profiles.user_id").
		Where(`intervention_heroes.intervention_id IN ? AND intervention_heroes.role = 'lead'
			AND hero_profiles.verification_status = 'approved' AND users.role = 'hero' AND users.status = 'active'`, ids).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("incident : lecture des leads : %w", err)
	}
	leads := make(map[uuid.UUID]*HeroReference, len(rows))
	for _, row := range rows {
		leads[row.InterventionID] = &HeroReference{ID: row.HeroID, Alias: row.Alias}
	}
	return leads, nil
}
