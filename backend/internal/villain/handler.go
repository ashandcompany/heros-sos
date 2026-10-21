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
	Alias       string  `json:"alias"`
	LastName    string  `json:"lastName"`
	FirstName   string  `json:"firstName"`
	MainSector  string  `json:"mainSector"`
	Description string  `json:"description"`
	TypeIDs     []int16 `json:"typeIds"`
}

type updatePayload struct {
	Alias       string   `json:"alias"`
	LastName    *string  `json:"lastName"`
	FirstName   *string  `json:"firstName"`
	MainSector  *string  `json:"mainSector"`
	Description *string  `json:"description"`
	TypeIDs     *[]int16 `json:"typeIds"`
	ThreatLevel *string  `json:"threatLevel"`
	Status      *string  `json:"status"`
}

// Response est un vilain.
type Response struct {
	ID          uuid.UUID `json:"id"`
	Alias       string    `json:"alias"`
	LastName    string    `json:"lastName,omitempty"`
	FirstName   string    `json:"firstName,omitempty"`
	MainSector  string    `json:"mainSector,omitempty"`
	Description string    `json:"description,omitempty"`
	TypeIDs     []int16   `json:"typeIds"`
	ReportedIncidentsCount int64 `json:"reportedIncidentsCount"`
	ThreatLevel string    `json:"threatLevel"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func newResponse(v Villain) Response {
	return Response{
		ID: v.ID, Alias: v.Alias, LastName: v.LastName, FirstName: v.FirstName,
		MainSector: v.MainSector, Description: v.Description, TypeIDs: v.TypeIDs,
		ReportedIncidentsCount: v.ReportedIncidentsCount,
		ThreatLevel: string(v.ThreatLevel), Status: string(v.Status), CreatedAt: v.CreatedAt,
	}
}

// Update traite PATCH /villains/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return
	}
	var body updatePayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.TypeIDs == nil {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains champs sont invalides.", Fields: map[string]string{"typeIds": "required"},
		})
		return
	}
	updated, err := h.service.Update(r.Context(), principal.UserID, principal.Role, id, UpdateInput{
		Alias: body.Alias, LastName: body.LastName, FirstName: body.FirstName,
		MainSector: body.MainSector, Description: body.Description, TypeIDs: *body.TypeIDs,
		ThreatLevel: body.ThreatLevel, Status: body.Status,
	}, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newResponse(*updated))
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
	case errors.Is(err, ErrNotAllowed):
		return httpx.Forbidden("Vous ne pouvez pas modifier cette fiche de vilain.")
	}
	return err
}
