package villain

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// CodeAliasTaken est le code d'erreur d'un pseudo de vilain déjà utilisé.
const CodeAliasTaken = "villain_alias_already_used"

// Handler expose les endpoints des vilains.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

// NewHandler crée les handlers.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type createPayload struct {
	Alias       string `json:"alias"`
	Description string `json:"description"`
}

// Response est un vilain.
type Response struct {
	ID          uuid.UUID `json:"id"`
	Alias       string    `json:"alias"`
	Description string    `json:"description"`
	ThreatLevel string    `json:"threatLevel"`
	CreatedAt   time.Time `json:"createdAt"`
}

func newResponse(v Villain) Response {
	return Response{ID: v.ID, Alias: v.Alias, Description: v.Description, ThreatLevel: v.ThreatLevel, CreatedAt: v.CreatedAt}
}

// Create traite POST /villains.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body createPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	created, err := h.service.Create(r.Context(), principal.UserID, CreateInput(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newResponse(*created))
}

// List traite GET /villains?q=&limit=&offset=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	villains, total, err := h.service.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")), page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]Response, len(villains))
	for i, v := range villains {
		items[i] = newResponse(v)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("Vilain introuvable.")
	case errors.Is(err, ErrAliasTaken):
		return httpx.Conflict(CodeAliasTaken, "Un vilain porte déjà ce pseudo.")
	}
	return err
}
