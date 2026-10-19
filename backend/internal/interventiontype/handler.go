package interventiontype

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// CodeLabelTaken est le code d'erreur d'un libellé déjà utilisé.
const CodeLabelTaken = "intervention_type_label_already_used"

// Handler expose les endpoints des types d'intervention.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

// NewHandler crée les handlers.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type payload struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Response est un type d'intervention.
type Response struct {
	ID          uuid.UUID `json:"id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"createdAt"`
}

func newResponse(t InterventionType) Response {
	return Response{ID: t.ID, Label: t.Label, Description: t.Description, Active: t.Active, CreatedAt: t.CreatedAt}
}

// ListActive traite GET /intervention-types : les types que l'on peut choisir.
func (h *Handler) ListActive(w http.ResponseWriter, r *http.Request) { h.list(w, r, true) }

// ListAll traite GET /admin/intervention-types : tous les types, désactivés compris.
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) { h.list(w, r, false) }

func (h *Handler) list(w http.ResponseWriter, r *http.Request, onlyActive bool) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	types, total, err := h.service.List(r.Context(), onlyActive, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]Response, len(types))
	for i, t := range types {
		items[i] = newResponse(t)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

// Create traite POST /admin/intervention-types.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body payload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	created, err := h.service.Create(r.Context(), principal.UserID, Input(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newResponse(*created))
}

// Deactivate traite POST /admin/intervention-types/{id}/deactivate.
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	if err := h.service.Deactivate(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("Type d'intervention introuvable.")
	case errors.Is(err, ErrLabelTaken):
		return httpx.Conflict(CodeLabelTaken, "Un type d'intervention porte déjà ce libellé.")
	}
	return err
}
