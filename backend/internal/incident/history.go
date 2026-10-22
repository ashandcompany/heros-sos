package incident

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

// Historique public des fiches de l'encyclopédie : incidents résolus par un héros, méfaits d'un
// vilain. Contrairement à GET /incidents, ces listes contiennent les incidents résolus ; elles ne
// donnent ni la description ni l'adresse précise, seulement la ville.

// ErrHeroNotFound indique qu'aucun héros public (approuvé et actif) ne porte cet identifiant.
var ErrHeroNotFound = errors.New("incident : héros public introuvable")

// HeroReference est le héros qui mène une intervention.
type HeroReference struct {
	ID    uuid.UUID
	Alias string
}

// Summary est un incident vu depuis la fiche publique d'un héros ou d'un vilain.
type Summary struct {
	ID        uuid.UUID
	Title     string
	TypeCode  string
	TypeLabel string
	Status    Status
	City      string
	CreatedAt time.Time
	// ResolvedAt est la fin de l'intervention réussie (nil si l'incident n'est pas résolu).
	ResolvedAt *time.Time
	// Lead mène la dernière intervention, nil sans intervention ou si son profil n'est plus public.
	Lead     *HeroReference
	Villains []VillainReference
}

type summaryRow struct {
	ID             uuid.UUID
	Title          string
	TypeCode       string
	TypeLabel      string
	Status         string
	City           string
	CreatedAt      time.Time
	ResolvedAt     *time.Time
	InterventionID *uuid.UUID
}

// --- Stockage ----------------------------------------------------------------------------------

const summaryColumns = `incidents.id, incidents.title, incident_types.code AS type_code,
	incident_types.label AS type_label, incidents.status, incidents.city, incidents.created_at`

// PublicHeroExists indique si l'identifiant est celui d'un héros visible dans l'encyclopédie.
func (r *gormRepository) PublicHeroExists(ctx context.Context, heroID uuid.UUID) (bool, error) {
	var exists bool
	err := r.conn(ctx).Raw(`SELECT EXISTS (
		SELECT 1 FROM hero_profiles JOIN users ON users.id = hero_profiles.user_id
		WHERE hero_profiles.user_id = ? AND hero_profiles.verification_status = 'approved'
		  AND users.role = 'hero' AND users.status = 'active')`, heroID).Scan(&exists).Error
	if err != nil {
		return false, fmt.Errorf("incident : recherche du héros : %w", err)
	}
	return exists, nil
}

// Les deux requêtes de page des historiques, en constantes : le test des plans d'exécution (plans_test.go) les
// explique telles quelles, pour qu'un index supprimé ou une réécriture qui reparcourt la table ne passe pas
// inaperçu. Paramètres : l'identifiant, la taille de la page, le décalage.
const (
	resolvedByHeroSQL = `WITH page AS (
			SELECT intervention_id, resolved_at FROM intervention_heroes
			WHERE hero_id = ? AND resolved_at IS NOT NULL
			ORDER BY resolved_at DESC, intervention_id DESC
			LIMIT ? OFFSET ?)
		SELECT ` + summaryColumns + `, page.resolved_at AS resolved_at, interventions.id AS intervention_id
		FROM page
		JOIN interventions ON interventions.id = page.intervention_id
		JOIN incidents ON incidents.id = interventions.incident_id
		JOIN incident_types ON incident_types.id = incidents.type_id
		ORDER BY page.resolved_at DESC, page.intervention_id DESC`

	byVillainSQL = `WITH page AS (
			SELECT incident_id, incident_created_at FROM incident_villains
			WHERE villain_id = ? AND incident_approved
			ORDER BY incident_created_at DESC, incident_id DESC
			LIMIT ? OFFSET ?)
		SELECT ` + summaryColumns + `,
			CASE WHEN incidents.status = 'succeeded' THEN latest.ended_at END AS resolved_at,
			latest.id AS intervention_id
		FROM page
		JOIN incidents ON incidents.id = page.incident_id
		JOIN incident_types ON incident_types.id = incidents.type_id
		LEFT JOIN LATERAL (
			SELECT id, ended_at FROM interventions WHERE interventions.incident_id = incidents.id
			ORDER BY started_at DESC NULLS LAST, id DESC LIMIT 1) AS latest ON TRUE
		ORDER BY page.incident_created_at DESC, page.incident_id DESC`
)

// ListResolvedByHero retourne les incidents approuvés résolus par une intervention à laquelle le héros a
// participé, les plus récemment résolus d'abord (resolved_at n'est renseigné que pour un incident approuvé). L'index (hero_id, resolved_at DESC, intervention_id DESC) donne la page sans
// aucun tri : seules les interventions de la page sont jointes à leurs incidents. Le total est le compteur
// hero_stats (voir la migration create_read_counters).
func (r *gormRepository) ListResolvedByHero(ctx context.Context, heroID uuid.UUID, limit, offset int) ([]Summary, int64, error) {
	var total int64
	err := r.conn(ctx).Table("hero_stats").Select("COALESCE(SUM(resolved), 0)").Where("hero_id = ?", heroID).Scan(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("incident : comptage des incidents résolus : %w", err)
	}
	var rows []summaryRow
	err = r.conn(ctx).Raw(resolvedByHeroSQL, heroID, limit, offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("incident : liste des incidents résolus : %w", err)
	}
	items, err := r.hydrateSummaries(ctx, rows)
	return items, total, err
}

// ListByVillain retourne les incidents approuvés auxquels le vilain est associé, quel que soit leur état, les plus
// récents d'abord. Le lead est celui de la dernière intervention. La page est choisie dans l'index
// (villain_id, incident_created_at DESC, incident_id DESC) avant toute jointure : sans cela, la jointure latérale
// vers la dernière intervention était calculée pour tous les incidents du vilain, pas pour les 20 affichés.
func (r *gormRepository) ListByVillain(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]Summary, int64, error) {
	var total int64
	err := r.conn(ctx).Table("villain_stats").Select("COALESCE(SUM(incidents), 0)").Where("villain_id = ?", villainID).Scan(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("incident : comptage des incidents du vilain : %w", err)
	}
	var rows []summaryRow
	err = r.conn(ctx).Raw(byVillainSQL, villainID, limit, offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("incident : liste des incidents du vilain : %w", err)
	}
	items, err := r.hydrateSummaries(ctx, rows)
	return items, total, err
}

// hydrateSummaries ajoute à chaque incident ses vilains et le lead de l'intervention retenue.
func (r *gormRepository) hydrateSummaries(ctx context.Context, rows []summaryRow) ([]Summary, error) {
	items := make([]Summary, len(rows))
	incidentIDs := make([]uuid.UUID, len(rows))
	byIncident := make(map[uuid.UUID]int, len(rows))
	byIntervention := make(map[uuid.UUID]int, len(rows))
	for i, row := range rows {
		items[i] = Summary{
			ID: row.ID, Title: row.Title, TypeCode: row.TypeCode, TypeLabel: row.TypeLabel,
			Status: Status(row.Status), City: row.City, CreatedAt: row.CreatedAt, ResolvedAt: row.ResolvedAt,
			Villains: []VillainReference{},
		}
		incidentIDs[i], byIncident[row.ID] = row.ID, i
		if row.InterventionID != nil {
			byIntervention[*row.InterventionID] = i
		}
	}
	if len(rows) == 0 {
		return items, nil
	}

	var villains []struct {
		IncidentID uuid.UUID
		VillainID  uuid.UUID
		Alias      string
	}
	if err := r.conn(ctx).Table("incident_villains").
		Select("incident_villains.incident_id, villains.id AS villain_id, villains.alias").
		Joins("JOIN villains ON villains.id = incident_villains.villain_id").
		Where("incident_villains.incident_id IN ?", incidentIDs).Order("villains.alias").Scan(&villains).Error; err != nil {
		return nil, fmt.Errorf("incident : lecture des vilains : %w", err)
	}
	for _, link := range villains {
		index := byIncident[link.IncidentID]
		items[index].Villains = append(items[index].Villains, VillainReference{ID: link.VillainID, Alias: link.Alias})
	}

	leads, err := r.publicLeads(ctx, byIntervention)
	if err != nil {
		return nil, err
	}
	for interventionID, lead := range leads {
		items[byIntervention[interventionID]].Lead = lead
	}
	return items, nil
}

// --- Service -----------------------------------------------------------------------------------

// ResolvedByHero retourne les incidents résolus par un héros public, ErrHeroNotFound sinon.
func (s *Service) ResolvedByHero(ctx context.Context, heroID uuid.UUID, limit, offset int) ([]Summary, int64, error) {
	exists, err := s.Repo.PublicHeroExists(ctx, heroID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, ErrHeroNotFound
	}
	return s.Repo.ListResolvedByHero(ctx, heroID, limit, offset)
}

// ByVillain retourne les incidents d'un vilain, villain.ErrNotFound s'il n'existe pas.
func (s *Service) ByVillain(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]Summary, int64, error) {
	if _, err := s.Villains.Find(ctx, villainID); err != nil {
		return nil, 0, err
	}
	return s.Repo.ListByVillain(ctx, villainID, limit, offset)
}

// --- HTTP --------------------------------------------------------------------------------------

// SummaryResponse est un incident de l'historique public d'un héros ou d'un vilain.
type SummaryResponse struct {
	ID         uuid.UUID        `json:"id"`
	Title      string           `json:"title"`
	TypeCode   string           `json:"typeCode"`
	TypeLabel  string           `json:"typeLabel"`
	Status     string           `json:"status"`
	City       string           `json:"city"`
	CreatedAt  time.Time        `json:"createdAt"`
	ResolvedAt *time.Time       `json:"resolvedAt"`
	Lead       *heroSummary     `json:"lead"`
	Villains   []villainSummary `json:"villains"`
}

type heroSummary struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

func newSummaryResponse(in Summary) SummaryResponse {
	villains := make([]villainSummary, len(in.Villains))
	for i, v := range in.Villains {
		villains[i] = villainSummary{ID: v.ID, Alias: v.Alias}
	}
	response := SummaryResponse{
		ID: in.ID, Title: in.Title, TypeCode: in.TypeCode, TypeLabel: in.TypeLabel, Status: string(in.Status),
		City: in.City, CreatedAt: in.CreatedAt, ResolvedAt: in.ResolvedAt, Villains: villains,
	}
	if in.Lead != nil {
		response.Lead = &heroSummary{ID: in.Lead.ID, Alias: in.Lead.Alias}
	}
	return response
}

// HeroIncidents traite GET /heroes/{heroId}/incidents.
func (h *Handler) HeroIncidents(w http.ResponseWriter, r *http.Request) {
	h.listSummaries(w, r, "heroId", ErrHeroNotFound, h.Service.ResolvedByHero)
}

// VillainIncidents traite GET /villains/{id}/incidents.
func (h *Handler) VillainIncidents(w http.ResponseWriter, r *http.Request) {
	h.listSummaries(w, r, "id", villain.ErrNotFound, h.Service.ByVillain)
}

func (h *Handler) listSummaries(
	w http.ResponseWriter, r *http.Request, param string, notFound error,
	list func(context.Context, uuid.UUID, int, int) ([]Summary, int64, error),
) {
	id, err := uuid.Parse(r.PathValue(param))
	if err != nil {
		h.fail(w, r, notFound)
		return
	}
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	summaries, total, err := list(r.Context(), id, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]SummaryResponse, len(summaries))
	for i, item := range summaries {
		items[i] = newSummaryResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}
