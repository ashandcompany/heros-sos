package incident

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

// CodeAlreadyExists est le code d'erreur d'une déclaration dont l'identifiant existe déjà.
const CodeAlreadyExists = "incident_already_exists"

// HandlerDeps regroupe les dépendances du handler.
type HandlerDeps struct {
	Service *Service
	Logger  *slog.Logger
	HTTP    config.HTTP
	Limits  config.RateLimits
	Clock   clock.Clock
}

// Handler expose les endpoints des incidents.
type Handler struct {
	HandlerDeps
	createPerUser *ratelimit.Limiter
}

// NewHandler crée les handlers et leur limiteur de débit.
func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{
		HandlerDeps:   deps,
		createPerUser: ratelimit.New(deps.Limits.IncidentCreatePerUser, deps.Limits.MaxTrackedKeys, deps.Clock),
	}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées.
func (h *Handler) SweepLimiters() { h.createPerUser.Sweep() }

type createPayload struct {
	ID              *uuid.UUID `json:"id"`
	TypeID          int16      `json:"typeId"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ReporterRole    *string    `json:"reporterRole"`
	CitizensRescued int16      `json:"citizensRescued"`
	Location        location   `json:"location"`
	OccurredAt      *string    `json:"occurredAt"`
	VillainIDs      []string   `json:"villainIds"`
}

type moderationPayload struct {
	Decision ModerationStatus `json:"decision"`
}

type priorityPayload struct {
	Priority Priority `json:"priority"`
}

type updatePayload struct {
	TypeID      int16          `json:"typeId"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      Status         `json:"status"`
	Location    updateLocation `json:"location"`
	VillainID   *string        `json:"villainId"`
	HeroID      *string        `json:"heroId"`
}

type updateLocation struct {
	Address   string   `json:"address"`
	City      string   `json:"city"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type location struct {
	Address           string   `json:"address"`
	City              string   `json:"city"`
	PostalCode        string   `json:"postalCode"`
	Country           string   `json:"country"`
	AddressComplement string   `json:"addressComplement"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
}

// Response est un incident vu par son déclarant ou un modérateur.
type Response struct {
	ID              uuid.UUID        `json:"id"`
	TypeID          int16            `json:"typeId"`
	TypeCode        string           `json:"typeCode"`
	TypeLabel       string           `json:"typeLabel"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Status          string           `json:"status"`
	Priority        string           `json:"priority"`
	Moderation      string           `json:"moderation"`
	ReporterRole    *string          `json:"reporterRole,omitempty"`
	CitizensRescued int16            `json:"citizensRescued"`
	Location        locationResponse `json:"location"`
	OccurredAt      *time.Time       `json:"occurredAt,omitempty"`
	VillainIDs      []string         `json:"villainIds"`
	Villains        []villainSummary `json:"villains"`
	InterventionID  *uuid.UUID       `json:"interventionId,omitempty"`
	ResolvedAt      *time.Time       `json:"resolvedAt"`
	Lead            *heroSummary     `json:"lead"`
	CreatedAt       time.Time        `json:"createdAt"`
}

type locationResponse struct {
	Address           string   `json:"address"`
	City              string   `json:"city"`
	PostalCode        string   `json:"postalCode,omitempty"`
	Country           string   `json:"country,omitempty"`
	AddressComplement string   `json:"addressComplement,omitempty"`
	Latitude          *float64 `json:"latitude,omitempty"`
	Longitude         *float64 `json:"longitude,omitempty"`
}

type villainSummary struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

// NewResponse convertit un incident en sa représentation JSON (partagée avec le tableau de bord héros).
func NewResponse(in Incident) Response {
	villainIDs := make([]string, len(in.VillainIDs))
	for i, id := range in.VillainIDs {
		villainIDs[i] = id.String()
	}
	villains := make([]villainSummary, len(in.Villains))
	for i, villain := range in.Villains {
		villains[i] = villainSummary{ID: villain.ID, Alias: villain.Alias}
	}
	var lead *heroSummary
	if in.Lead != nil {
		lead = &heroSummary{ID: in.Lead.ID, Alias: in.Lead.Alias}
	}
	var reporterRole *string
	if in.ReporterRole != nil {
		value := string(*in.ReporterRole)
		reporterRole = &value
	}
	return Response{
		ID: in.ID, TypeID: in.TypeID, TypeCode: in.TypeCode, TypeLabel: in.TypeLabel, Title: in.Title,
		Description: in.Description, Status: string(in.Status), Priority: string(in.Priority),
		Moderation: string(in.Moderation), ReporterRole: reporterRole,
		CitizensRescued: in.CitizensRescued,
		Location: locationResponse{
			Address: in.Street, City: in.City, PostalCode: in.PostalCode, Country: in.Country,
			AddressComplement: in.AddressComplement, Latitude: in.Latitude, Longitude: in.Longitude,
		}, OccurredAt: in.OccurredAt,
		VillainIDs: villainIDs, Villains: villains, InterventionID: in.InterventionID,
		ResolvedAt: in.ResolvedAt, Lead: lead, CreatedAt: in.CreatedAt,
	}
}

// maxFilterValues borne le nombre d'identifiants acceptés par filtre (route publique).
const maxFilterValues = 20

// parseUUIDs lit un paramètre répété (?name=a&name=b).
func parseUUIDs(values []string, name string, invalid map[string]string) []uuid.UUID {
	if len(values) > maxFilterValues {
		invalid[name] = "invalid"
		return nil
	}
	ids := make([]uuid.UUID, 0, len(values))
	for _, raw := range values {
		id, err := uuid.Parse(raw)
		if err != nil {
			invalid[name] = "invalid"
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	query := r.URL.Query()
	invalid := map[string]string{}
	filter := ListFilter{
		HeroIDs:    parseUUIDs(query["heroId"], "heroId", invalid),
		VillainIDs: parseUUIDs(query["villainId"], "villainId", invalid),
	}
	if raw := strings.TrimSpace(query.Get("reporterId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			invalid["reporterId"] = "invalid"
		}
		filter.ReporterID = &id
	}
	for _, raw := range query["status"] {
		status, ok := ParseStatus(raw)
		if !ok {
			invalid["status"] = "invalid"
			break
		}
		filter.Statuses = append(filter.Statuses, status)
	}
	sort, ok := ParseListSort(strings.TrimSpace(query.Get("sort")))
	if !ok {
		invalid["sort"] = "invalid"
	}
	filter.Sort = sort
	if len(invalid) > 0 {
		return ListFilter{}, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains paramètres sont invalides.", Fields: invalid,
		}
	}
	return filter, nil
}

// parseCursorParam lit le paramètre cursor. Il remplace offset : les deux ensemble n'ont pas de sens. Le curseur est
// celui de l'ordre par date de déclaration : le tri par résolution se pagine par décalage (offset).
func parseCursorParam(r *http.Request, sort ListSort) (*Cursor, error) {
	query := r.URL.Query()
	raw := query.Get("cursor")
	if raw == "" {
		return nil, nil
	}
	invalid := func(field string) error {
		return &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains paramètres sont invalides.", Fields: map[string]string{field: "invalid"},
		}
	}
	if query.Has("offset") {
		return nil, invalid("offset")
	}
	cursor, err := ParseCursor(raw)
	if err != nil || sort == SortResolvedAt {
		return nil, invalid("cursor")
	}
	return &cursor, nil
}

// List traite GET /incidents.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	filter, err := parseListFilter(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if filter.After, err = parseCursorParam(r, filter.Sort); err != nil {
		h.fail(w, r, err)
		return
	}
	if principal.Role == user.RoleCitizen {
		if filter.ReporterID != nil && *filter.ReporterID != principal.UserID {
			h.fail(w, r, httpx.Forbidden("Vous ne pouvez consulter que vos propres signalements."))
			return
		}
		filter.ReporterID = &principal.UserID
		filter.ApprovedOnly = false
	} else if principal.Role != user.RoleModerator {
		if filter.ReporterID != nil && *filter.ReporterID != principal.UserID {
			h.fail(w, r, httpx.Forbidden("Vous ne pouvez consulter que vos propres signalements."))
			return
		}
		filter.ApprovedOnly = true
		if filter.ReporterID != nil {
			filter.ApprovedOnly = false
		}
	}
	result, err := h.Service.List(r.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]Response, len(result.Items))
	for i, item := range result.Items {
		items[i] = NewResponse(item)
	}
	var next string
	if result.Next != nil {
		next = result.Next.String()
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewCursorPageResponse(items, result.Total, page, next))
}

// Get traite GET /incidents/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	item, err := h.Service.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	principal, _ := auth.PrincipalFrom(r.Context())
	if item.Moderation != ModerationApproved &&
		principal.Role != user.RoleModerator &&
		((principal.Role != user.RoleCitizen && principal.Role != user.RoleHero) || item.ReporterID != principal.UserID) {
		h.fail(w, r, ErrNotFound)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewResponse(*item))
}

// Create traite POST /incidents.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if allowed, retryAfter := h.createPerUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	var body createPayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	in := CreateInput{
		TypeID: body.TypeID, Title: body.Title, Description: body.Description,
		ReporterRole:    body.ReporterRole,
		CitizensRescued: body.CitizensRescued, Street: body.Location.Address, PostalCode: body.Location.PostalCode,
		City: body.Location.City, Country: body.Location.Country, AddressComplement: body.Location.AddressComplement,
		Latitude: body.Location.Latitude, Longitude: body.Location.Longitude, VillainIDs: body.VillainIDs,
	}
	if body.OccurredAt != nil {
		in.OccurredAt = *body.OccurredAt
	}
	if body.ID != nil {
		if *body.ID == uuid.Nil {
			h.fail(w, r, &httpx.Error{
				Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
				Message: "Certains champs sont invalides.", Fields: map[string]string{"id": "invalid"},
			})
			return
		}
		in.ID = *body.ID
	}

	created, err := h.Service.Create(r.Context(), principal.UserID, principal.Role, in, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, NewResponse(*created))
}

func (h *Handler) Moderate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	var body moderationPayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.Service.Moderate(r.Context(), principal.UserID, id, body.Decision, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	var body updatePayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	var heroID *uuid.UUID
	if body.HeroID != nil && strings.TrimSpace(*body.HeroID) != "" {
		parsed, parseErr := uuid.Parse(strings.TrimSpace(*body.HeroID))
		if parseErr != nil {
			h.fail(w, r, &httpx.Error{
				Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
				Message: "Certaines informations sont invalides.", Fields: map[string]string{"heroId": "invalid"},
			})
			return
		}
		heroID = &parsed
	}
	input := UpdateInput{
		TypeID: body.TypeID, Title: body.Title, Description: body.Description, Status: body.Status,
		Street: body.Location.Address, City: body.Location.City,
		Latitude: body.Location.Latitude, Longitude: body.Location.Longitude,
		VillainID: body.VillainID, HeroID: heroID,
	}
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.Service.Update(r.Context(), principal.UserID, id, input, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	item, err := h.Service.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewResponse(*item))
}

func (h *Handler) SetPriority(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	var body priorityPayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.Service.SetPriority(r.Context(), principal.UserID, id, body.Priority, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		err = httpx.NotFound("Incident introuvable.")
	case errors.Is(err, ErrAlreadyExists):
		err = httpx.Conflict(CodeAlreadyExists, "Un incident porte déjà cet identifiant.")
	case errors.Is(err, ErrHeroNotFound):
		err = httpx.NotFound("Super-héros introuvable.")
	case errors.Is(err, ErrNotAvailable):
		err = httpx.Conflict("incident_not_available", "Cet incident ne peut plus être modifié.")
	case errors.Is(err, villain.ErrNotFound):
		err = httpx.NotFound("Vilain introuvable.")
	}
	httpx.Fail(w, r, h.Logger, err)
}
