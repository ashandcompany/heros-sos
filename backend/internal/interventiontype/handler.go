package interventiontype

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// CodeCodeTaken est le code d'erreur d'un code déjà utilisé.
const CodeCodeTaken = "incident_type_code_already_used"

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
	Code          string `json:"code"`
	Label         string `json:"label"`
	EndangersLife bool   `json:"endangersLife"`
}

// Response est un type d'intervention.
type Response struct {
	ID            int16  `json:"id"`
	Code          string `json:"code"`
	Label         string `json:"label"`
	EndangersLife bool   `json:"endangersLife"`
}

func newResponse(t InterventionType) Response {
	return Response{ID: t.ID, Code: t.Code, Label: t.Label, EndangersLife: t.EndangersLife}
}

// ListActive traite GET /intervention-types : les types que l'on peut choisir.
func (h *Handler) ListActive(w http.ResponseWriter, r *http.Request) { h.list(w, r) }

// ListAll traite GET /admin/intervention-types : tous les types, désactivés compris.
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) { h.list(w, r) }

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	types, total, err := h.service.List(r.Context(), page.Limit, page.Offset)
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

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("Type d'intervention introuvable.")
	case errors.Is(err, ErrCodeTaken):
		return httpx.Conflict(CodeCodeTaken, "Ce code de type d'incident est déjà utilisé.")
	}
	return err
}
