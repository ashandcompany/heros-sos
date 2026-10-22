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
	TypeID          int16    `json:"typeId"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	CitizensRescued int16    `json:"citizensRescued"`
	Location        location `json:"location"`
	OccurredAt      *string  `json:"occurredAt"`
	VillainIDs      []string `json:"villainIds"`
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
	CitizensRescued int16            `json:"citizensRescued"`
	Location        locationResponse `json:"location"`
	OccurredAt      *time.Time       `json:"occurredAt,omitempty"`
	VillainIDs      []string         `json:"villainIds"`
	Villains        []villainSummary `json:"villains"`
	InterventionID  *uuid.UUID       `json:"interventionId,omitempty"`
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

func newResponse(in Incident) Response {
	villainIDs := make([]string, len(in.VillainIDs))
	for i, id := range in.VillainIDs {
		villainIDs[i] = id.String()
	}
	villains := make([]villainSummary, len(in.Villains))
	for i, villain := range in.Villains {
		villains[i] = villainSummary{ID: villain.ID, Alias: villain.Alias}
	}
	return Response{
		ID: in.ID, TypeID: in.TypeID, TypeCode: in.TypeCode, TypeLabel: in.TypeLabel, Title: in.Title,
		Description: in.Description, Status: string(in.Status), Priority: string(in.Priority),
		CitizensRescued: in.CitizensRescued,
		Location: locationResponse{
			Address: in.Street, City: in.City, PostalCode: in.PostalCode, Country: in.Country,
			AddressComplement: in.AddressComplement, Latitude: in.Latitude, Longitude: in.Longitude,
		}, OccurredAt: in.OccurredAt,
		VillainIDs: villainIDs, Villains: villains, InterventionID: in.InterventionID, CreatedAt: in.CreatedAt,
	}
}

// List traite GET /incidents.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var reporterID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("reporterId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			h.fail(w, r, &httpx.Error{
				Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
				Message: "Certains paramètres sont invalides.", Fields: map[string]string{"reporterId": "invalid"},
			})
			return
		}
		reporterID = &parsed
	}
	if principal.Role == user.RoleCitizen {
		if reporterID != nil && *reporterID != principal.UserID {
			h.fail(w, r, httpx.Forbidden("Vous ne pouvez consulter que vos propres signalements."))
			return
		}
		reporterID = &principal.UserID
	}
	incidents, total, err := h.Service.List(r.Context(), reporterID, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]Response, len(incidents))
	for i, item := range incidents {
		items[i] = newResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
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
	httpx.WriteJSON(w, http.StatusOK, newResponse(*item))
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
		CitizensRescued: body.CitizensRescued, Street: body.Location.Address, PostalCode: body.Location.PostalCode,
		City: body.Location.City, Country: body.Location.Country, AddressComplement: body.Location.AddressComplement,
		Latitude: body.Location.Latitude, Longitude: body.Location.Longitude, VillainIDs: body.VillainIDs,
	}
	if body.OccurredAt != nil {
		in.OccurredAt = *body.OccurredAt
	}

	created, err := h.Service.Create(r.Context(), principal.UserID, in, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newResponse(*created))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		err = httpx.NotFound("Incident introuvable.")
	case errors.Is(err, ErrHeroNotFound):
		err = httpx.NotFound("Super-héros introuvable.")
	case errors.Is(err, villain.ErrNotFound):
		err = httpx.NotFound("Vilain introuvable.")
	}
	httpx.Fail(w, r, h.Logger, err)
}
