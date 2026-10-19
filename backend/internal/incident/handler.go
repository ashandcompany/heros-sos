package incident

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
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
	Type         string  `json:"interventionTypeId"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	ReporterRole string  `json:"reporterRole"`
	Location     location `json:"location"`
	OccurredAt   *string `json:"occurredAt"`
	VillainID    *string `json:"villainId"`
}

type location struct {
	Address string `json:"address"`
	City    string `json:"city"`
}

// Response est un incident vu par son déclarant ou un modérateur.
type Response struct {
	ID               uuid.UUID        `json:"id"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	InterventionType typeSummary      `json:"interventionType"`
	Status           string           `json:"status"`
	ReporterRole     string           `json:"reporterRole"`
	Location         location         `json:"location"`
	ReportedAt       time.Time        `json:"reportedAt"`
	OccurredAt       *time.Time       `json:"occurredAt,omitempty"`
	Villain          *villainSummary  `json:"villain"`
}

type typeSummary struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
}

type villainSummary struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

func newResponse(in Incident) Response {
	response := Response{
		ID: in.ID, Title: in.Title, Description: in.Description,
		InterventionType: typeSummary{ID: in.InterventionTypeID, Label: in.InterventionType},
		Status:           string(in.Status), ReporterRole: string(in.ReporterRole),
		Location:   location{Address: in.Address, City: in.City},
		ReportedAt: in.CreatedAt, OccurredAt: in.OccurredAt,
	}
	if in.VillainID != nil {
		response.Villain = &villainSummary{ID: *in.VillainID, Alias: in.VillainAlias}
	}
	return response
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
		ReporterRole: body.ReporterRole, InterventionTypeID: body.Type, Title: body.Title,
		Description: body.Description, Address: body.Location.Address, City: body.Location.City,
	}
	if body.OccurredAt != nil {
		in.OccurredAt = *body.OccurredAt
	}
	if body.VillainID != nil {
		in.VillainID = *body.VillainID
	}

	created, err := h.Service.Create(r.Context(), principal.UserID, in, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newResponse(*created))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.Logger, err)
}
