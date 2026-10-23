package account

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/avatar"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Codes d'erreur de l'API du profil, stables et destinés au frontend. L'adresse déjà prise réutilise
// auth.CodeEmailAlreadyUsed.
const (
	CodeEmailUnchanged     = "email_unchanged"
	CodeChangeNotFound     = "email_change_not_found"
	CodeChangeNotPending   = "email_change_not_pending"
	CodeChangeNotConfirmed = "email_change_not_confirmed"
	CodeNotHero            = "hero_profile_required"
	messageValidationError = "Certains paramètres sont invalides."
)

// HandlerDeps regroupe les dépendances du handler.
type HandlerDeps struct {
	Service *Service
	Logger  *slog.Logger
	HTTP    config.HTTP
	Limits  config.RateLimits
	Clock   clock.Clock
	Avatar  config.Avatar
	// AvatarURLPrefix est le début de l'adresse publique d'une photo, avant son nom de fichier
	// (par exemple « /api/v1/avatars/ »).
	AvatarURLPrefix string
}

// Handler expose les endpoints du profil. Il traduit HTTP vers le service et inversement ; les règles
// métier restent dans Service.
type Handler struct {
	HandlerDeps

	avatarPerUser        *ratelimit.Limiter
	emailChangePerUser   *ratelimit.Limiter
	exportPerUser        *ratelimit.Limiter
	passwordCheckPerUser *ratelimit.Limiter
}

// NewHandler crée le handler et ses limiteurs de débit.
func NewHandler(deps HandlerDeps) *Handler {
	limiter := func(rule config.RateRule) *ratelimit.Limiter {
		return ratelimit.New(rule, deps.Limits.MaxTrackedKeys, deps.Clock)
	}
	return &Handler{
		HandlerDeps:        deps,
		avatarPerUser:      limiter(deps.Limits.AvatarPerUser),
		emailChangePerUser: limiter(deps.Limits.EmailChangePerUser),
		exportPerUser:      limiter(deps.Limits.ExportPerUser),
		// Distinct de celui de POST /me/password (paquet auth) : chacun compte ses propres essais.
		passwordCheckPerUser: limiter(deps.Limits.PasswordCheckPerUser),
	}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées.
func (h *Handler) SweepLimiters() {
	h.avatarPerUser.Sweep()
	h.emailChangePerUser.Sweep()
	h.exportPerUser.Sweep()
	h.passwordCheckPerUser.Sweep()
}

// Get traite GET /me.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	account, err := h.Service.Get(r.Context(), principal.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.newResponse(account))
}

// UpdateContact traite PATCH /me.
func (h *Handler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body contactRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	in := ContactInput{Phone: body.MobilePhone, Alias: body.Alias, Bio: body.Bio}
	if address := body.Address; address != nil {
		in.Address = &AddressInput{
			Street: address.Street, Complement: address.Complement, PostalCode: address.PostalCode,
			City: address.City, Country: address.Country,
		}
	}

	account, err := h.Service.UpdateContact(r.Context(), principal.UserID, in, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.newResponse(account))
}

// UpdateAvatar traite PUT /me/avatar : le corps est l'image elle-même.
func (h *Handler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if allowed, retryAfter := h.avatarPerUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	if !avatar.IsAcceptedContentType(r.Header.Get("Content-Type")) {
		h.fail(w, r, httpx.NewError(http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType,
			"Le corps de la requête doit être une image JPEG, PNG ou WebP."))
		return
	}

	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.Avatar.MaxBytes))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		h.fail(w, r, httpx.NewError(http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, "Le corps de la requête est trop volumineux."))
		return
	case err != nil:
		h.fail(w, r, httpx.BadRequest("Le corps de la requête n'a pas pu être lu."))
		return
	case len(data) == 0:
		h.fail(w, r, &validation.Error{Fields: map[string]string{"avatar": validation.CodeRequired}})
		return
	}

	name, err := h.Service.SetAvatar(r.Context(), principal.UserID, data)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, avatarResponse{AvatarURL: h.AvatarURLPrefix + name})
}

// RemoveAvatar traite DELETE /me/avatar.
func (h *Handler) RemoveAvatar(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.Service.RemoveAvatar(r.Context(), principal.UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// RequestEmailChange traite POST /me/email-change-requests.
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if allowed, retryAfter := h.emailChangePerUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	var body emailChangeRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	change, err := h.Service.RequestEmailChange(r.Context(), principal.UserID, body.Email, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, EmailChangeStatusResponse{NewEmail: change.NewEmail, Status: string(change.Status)})
}

// SetNewsletter traite PUT /me/newsletter.
func (h *Handler) SetNewsletter(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body newsletterRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.Subscribed == nil {
		h.fail(w, r, &validation.Error{Fields: map[string]string{"subscribed": validation.CodeRequired}})
		return
	}
	if err := h.Service.SetNewsletter(r.Context(), principal.UserID, *body.Subscribed); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ConfirmEmailChange traite POST /email-change-requests/confirm : le titulaire de la nouvelle adresse a
// cliqué sur le lien reçu. La route est publique, comme le lien : il s'ouvre souvent sur un autre appareil
// que celui où la demande a été faite, et il porte sa propre preuve (signature).
func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var body confirmRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.Service.ConfirmEmailChange(r.Context(), body.Token, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ListEmailChanges traite GET /admin/email-change-requests (modérateur).
func (h *Handler) ListEmailChanges(w http.ResponseWriter, r *http.Request) {
	status := ChangeStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: messageValidationError, Fields: map[string]string{"status": validation.CodeInvalid},
		})
		return
	}
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	views, total, err := h.Service.ListEmailChanges(r.Context(), status, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]EmailChangeResponse, len(views))
	for i, view := range views {
		items[i] = EmailChangeResponse{
			ID: view.ID, UserID: view.UserID, FirstName: view.FirstName, LastName: view.LastName,
			CurrentEmail: view.CurrentEmail, NewEmail: view.NewEmail, Status: string(view.Status),
			CreatedAt: view.CreatedAt, ConfirmedAt: view.ConfirmedAt, ReviewedAt: view.ReviewedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

// ApproveEmailChange traite POST /admin/email-change-requests/{id}/approve.
func (h *Handler) ApproveEmailChange(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, h.Service.ApproveEmailChange)
}

// RejectEmailChange traite POST /admin/email-change-requests/{id}/reject.
func (h *Handler) RejectEmailChange(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, h.Service.RejectEmailChange)
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, moderatorID, requestID uuid.UUID, ip netip.Addr) error) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrChangeNotFound)
		return
	}
	if err := action(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.Logger, toHTTPError(err))
}

// toHTTPError traduit les erreurs métier en erreurs HTTP. Les autres (validation, interne) sont
// laissées à httpx.Fail.
func toHTTPError(err error) error {
	switch {
	case errors.Is(err, user.ErrNotFound):
		return httpx.NotFound("Compte introuvable.")
	case errors.Is(err, user.ErrEmailTaken):
		return httpx.Conflict(auth.CodeEmailAlreadyUsed, "Cette adresse email est déjà utilisée.")
	case errors.Is(err, ErrEmailUnchanged):
		return httpx.Conflict(CodeEmailUnchanged, "C'est déjà l'adresse email de ce compte.")
	case errors.Is(err, ErrChangeNotFound):
		return httpx.NewError(http.StatusNotFound, CodeChangeNotFound, "Demande de changement d'adresse introuvable.")
	case errors.Is(err, ErrChangeNotPending):
		return httpx.Conflict(CodeChangeNotPending, "Cette demande a déjà été traitée.")
	case errors.Is(err, ErrChangeNotConfirmed):
		return httpx.Conflict(CodeChangeNotConfirmed, "Le titulaire de la nouvelle adresse ne l'a pas encore confirmée.")
	case errors.Is(err, ErrLinkInvalid):
		return httpx.NewError(http.StatusBadRequest, auth.CodeInvalidToken, "Ce lien est invalide, expiré ou déjà utilisé.")
	case errors.Is(err, ErrNotHero):
		return httpx.NewError(http.StatusForbidden, CodeNotHero, "Le pseudo et la description sont réservés aux super-héros.")
	case errors.Is(err, hero.ErrAliasTaken):
		return httpx.Conflict(hero.CodeAliasTaken, "Ce pseudo est déjà utilisé.")
	case errors.Is(err, ErrNotErasable):
		return httpx.Conflict(CodeNotErasable, "Un modérateur ne peut anonymiser ni son propre compte, ni celui d'un autre modérateur.")
	case errors.Is(err, ErrAlreadyErased):
		return httpx.Conflict(CodeAlreadyErased, "Ce compte a déjà été supprimé.")
	}
	var locked *auth.LockedError
	if errors.As(err, &locked) {
		apiErr := httpx.RateLimited(max(time.Until(locked.Until), time.Second))
		apiErr.Code, apiErr.Message = auth.CodeAccountLocked, "Compte temporairement verrouillé après trop d'échecs."
		return apiErr
	}
	return err
}
