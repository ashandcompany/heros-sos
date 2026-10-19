package hero

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Codes d'erreur de l'API des demandes de super-héros.
const (
	CodeAlreadyPending = "hero_request_pending"
	CodeAlreadyHero    = "already_hero"
	CodeAliasTaken     = "alias_already_used"
	CodeNotPending     = "hero_request_not_pending"
	CodeNotEligible    = "hero_request_not_eligible"
	CodeRequestMissing = "hero_request_not_found"
)

// Handler expose les endpoints des demandes de super-héros.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

// NewHandler crée les handlers.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type requestPayload struct {
	Alias          string `json:"alias"`
	Bio            string `json:"bio"`
	RequestMessage string `json:"requestMessage"`
}

type rejectPayload struct {
	Reason string `json:"reason"`
}

// RequestStatusResponse est l'état de la demande de l'utilisateur courant.
type RequestStatusResponse struct {
	Status          string `json:"status"`
	Alias           string `json:"alias"`
	RejectionReason string `json:"rejectionReason,omitempty"`
}

// ApplicantResponse est une demande vue par un modérateur.
type ApplicantResponse struct {
	UserID         uuid.UUID `json:"userId"`
	Alias          string    `json:"alias"`
	Bio            string    `json:"bio"`
	RequestMessage string    `json:"requestMessage"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
}

// Request traite POST /heroes/requests (citoyen connecté).
func (h *Handler) Request(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var payload requestPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	err := h.service.Request(r.Context(), principal.UserID,
		RequestInput{Alias: payload.Alias, Bio: payload.Bio, Message: payload.RequestMessage},
		httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, RequestStatusResponse{Status: string(StatusPending), Alias: payload.Alias})
}

// Mine traite GET /heroes/requests/me.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	profile, err := h.service.Mine(r.Context(), principal.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, RequestStatusResponse{
		Status: string(profile.Status), Alias: profile.Alias, RejectionReason: profile.RejectionReason,
	})
}

// List traite GET /admin/hero-requests (modérateur).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := VerificationStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains paramètres sont invalides.", Fields: map[string]string{"status": "invalid"},
		})
		return
	}
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	applicants, total, err := h.service.List(r.Context(), status, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]ApplicantResponse, len(applicants))
	for i, applicant := range applicants {
		items[i] = ApplicantResponse{
			UserID: applicant.UserID, Alias: applicant.Alias, Bio: applicant.Bio,
			RequestMessage: applicant.RequestMessage, Status: string(applicant.Status),
			CreatedAt: applicant.CreatedAt, Email: applicant.Email,
			FirstName: applicant.FirstName, LastName: applicant.LastName,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

// Approve traite POST /admin/hero-requests/{userId}/approve.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	if err := h.service.Approve(r.Context(), principal.UserID, userID, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Reject traite POST /admin/hero-requests/{userId}/reject.
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var payload rejectPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.Reject(r.Context(), principal.UserID, userID, payload.Reason, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		h.fail(w, r, httpx.NotFound("Demande introuvable."))
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NewError(http.StatusNotFound, CodeRequestMissing, "Aucune demande trouvée.")
	case errors.Is(err, ErrAlreadyPending):
		return httpx.Conflict(CodeAlreadyPending, "Une demande est déjà en cours d'examen.")
	case errors.Is(err, ErrAlreadyHero):
		return httpx.Conflict(CodeAlreadyHero, "Vous êtes déjà super-héros.")
	case errors.Is(err, ErrAliasTaken):
		return httpx.Conflict(CodeAliasTaken, "Ce pseudo est déjà utilisé.")
	case errors.Is(err, ErrNotPending):
		return httpx.Conflict(CodeNotPending, "Cette demande a déjà été traitée.")
	case errors.Is(err, ErrNotEligible):
		return httpx.NewError(http.StatusForbidden, CodeNotEligible, "Votre compte ne peut pas faire cette demande.")
	case errors.Is(err, user.ErrNotFound):
		return httpx.NotFound("Utilisateur introuvable.")
	}
	return err
}
