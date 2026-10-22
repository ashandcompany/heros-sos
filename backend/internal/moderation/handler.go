package moderation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Codes d'erreur de l'API de modération.
const (
	CodeSelfAction       = "self_action_forbidden"
	CodeProtectedAccount = "protected_account"
)

// Handler expose les endpoints de modération des comptes.
type Handler struct {
	service *Service
	summary *SummaryStore
	logger  *slog.Logger
}

// NewHandler crée les handlers.
func NewHandler(service *Service, summary *SummaryStore, logger *slog.Logger) *Handler {
	return &Handler{service: service, summary: summary, logger: logger}
}

// UserSummary est un compte vu par un modérateur.
type UserSummary struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	FirstName  string     `json:"firstName"`
	LastName   string     `json:"lastName"`
	Alias      string     `json:"alias,omitempty"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// ListUsers traite GET /admin/users?role=&status=&email=&limit=&offset=.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := user.Filter{
		Role:        user.Role(query.Get("role")),
		Status:      user.Status(query.Get("status")),
		EmailPrefix: strings.TrimSpace(query.Get("email")),
	}
	invalid := make(map[string]string)
	if filter.Role != "" && !filter.Role.Valid() {
		invalid["role"] = "invalid"
	}
	switch filter.Status {
	case "", user.StatusPendingActivation, user.StatusActive, user.StatusDeactivated:
	default:
		invalid["status"] = "invalid"
	}
	page, err := httpx.ParsePage(r)
	if len(invalid) > 0 || err != nil {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains paramètres sont invalides.", Fields: mergeInvalid(invalid, err),
		})
		return
	}
	filter.Limit, filter.Offset = page.Limit, page.Offset

	users, total, err := h.service.ListUsers(r.Context(), filter)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]UserSummary, len(users))
	for i, account := range users {
		items[i] = UserSummary{
			ID: account.ID, Email: account.Email, FirstName: account.FirstName, LastName: account.LastName,
			Alias: account.Alias, Role: string(account.Role), Status: string(account.Status),
			VerifiedAt: account.VerifiedAt, CreatedAt: account.CreatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

func mergeInvalid(invalid map[string]string, pageErr error) map[string]string {
	var apiErr *httpx.Error
	if errors.As(pageErr, &apiErr) {
		for field, code := range apiErr.Fields {
			invalid[field] = code
		}
	}
	return invalid
}

// Deactivate traite POST /admin/users/{id}/deactivate.
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.act(w, r, h.service.Deactivate)
}

// Reactivate traite POST /admin/users/{id}/reactivate.
func (h *Handler) Reactivate(w http.ResponseWriter, r *http.Request) {
	h.act(w, r, h.service.Reactivate)
}

// Verify traite POST /admin/users/{id}/verify.
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	h.act(w, r, h.service.Verify)
}

func (h *Handler) act(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, actor, target uuid.UUID, ip netip.Addr) error) {
	principal, _ := auth.PrincipalFrom(r.Context())
	targetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, httpx.NotFound("Utilisateur introuvable."))
		return
	}
	if err := action(r.Context(), principal.UserID, targetID, httpx.ClientIPFrom(r.Context())); err != nil {
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
	case errors.Is(err, user.ErrNotFound):
		return httpx.NotFound("Utilisateur introuvable.")
	case errors.Is(err, ErrSelfAction):
		return httpx.NewError(http.StatusConflict, CodeSelfAction, "Vous ne pouvez pas effectuer cette action sur votre propre compte.")
	case errors.Is(err, ErrProtectedAccount):
		return httpx.NewError(http.StatusForbidden, CodeProtectedAccount, "Les comptes de modérateur ne peuvent pas être désactivés par l'API.")
	}
	return err
}
