package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Codes d'erreur de l'API d'authentification, stables et destinés au frontend.
const (
	CodeInvalidCredentials = "invalid_credentials"
	CodeNotActivated       = "account_not_activated"
	CodeDeactivated        = "account_deactivated"
	CodeAccountLocked      = "account_locked"
	CodeInvalidToken       = "invalid_token"
	CodeEmailAlreadyUsed   = "email_already_used"
)

const (
	unknownClient = "unknown"
	// minRetryAfter évite d'annoncer « réessayez dans 0 seconde » à un client verrouillé.
	minRetryAfter = time.Second
)

// HandlerDeps regroupe les dépendances des handlers d'authentification.
type HandlerDeps struct {
	Service *Service
	Cookies CookieManager
	CSRF    *CSRF
	Clock   clock.Clock
	Logger  *slog.Logger
	HTTP    config.HTTP
	Limits  config.RateLimits
}

// Handler expose les endpoints /auth/*. Il traduit HTTP vers le service et inversement ; les
// règles métier restent dans Service.
type Handler struct {
	HandlerDeps

	registerPerIP         *ratelimit.Limiter
	loginPerIP            *ratelimit.Limiter
	loginPerIdentifier    *ratelimit.Limiter
	passwordResetPerIP    *ratelimit.Limiter
	passwordResetPerEmail *ratelimit.Limiter
}

// NewHandler crée les handlers et leurs limiteurs de débit.
func NewHandler(deps HandlerDeps) *Handler {
	limiter := func(rule config.RateRule) *ratelimit.Limiter {
		return ratelimit.New(rule, deps.Limits.MaxTrackedKeys, deps.Clock)
	}
	return &Handler{
		HandlerDeps:           deps,
		registerPerIP:         limiter(deps.Limits.RegisterPerIP),
		loginPerIP:            limiter(deps.Limits.LoginPerIP),
		loginPerIdentifier:    limiter(deps.Limits.LoginPerIdentifier),
		passwordResetPerIP:    limiter(deps.Limits.PasswordResetPerIP),
		passwordResetPerEmail: limiter(deps.Limits.PasswordResetPerEmail),
	}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées ; le serveur l'appelle
// périodiquement.
func (h *Handler) SweepLimiters() {
	for _, limiter := range []*ratelimit.Limiter{
		h.registerPerIP, h.loginPerIP, h.loginPerIdentifier, h.passwordResetPerIP, h.passwordResetPerEmail,
	} {
		limiter.Sweep()
	}
}

// Register traite POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if err := h.allow(h.registerPerIP, clientKey(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	var req registerRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	created, err := h.Service.Register(r.Context(), RegisterInput{
		Civility: req.Civility, FirstName: req.FirstName, LastName: req.LastName, BirthDate: req.BirthDate,
		Email: req.Email, Phone: req.MobilePhone, Password: req.Password,
		Address: user.Address{
			Street: req.Address.Street, Complement: req.Address.Complement, PostalCode: req.Address.PostalCode,
			City: req.Address.City, Country: req.Address.Country,
		},
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, registerResponse{User: NewUserResponse(created), ActivationRequired: true})
}

// Activate traite POST /auth/activate.
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.Service.Activate(r.Context(), req.Token); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ResendActivation traite POST /auth/resend-activation.
func (h *Handler) ResendActivation(w http.ResponseWriter, r *http.Request) {
	h.handleEmailRequest(w, r, h.Service.ResendActivation)
}

// RequestPasswordReset traite POST /auth/password-reset.
func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	h.handleEmailRequest(w, r, h.Service.RequestPasswordReset)
}

// handleEmailRequest factorise les demandes qui envoient un mail à une adresse : limitées par IP
// et par adresse (contre l'envoi massif de mails), et toujours acceptées avec la même réponse.
func (h *Handler) handleEmailRequest(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, email string) error) {
	if err := h.allow(h.passwordResetPerIP, clientKey(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	var req emailRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.allow(h.passwordResetPerEmail, "email:"+normalizeKey(req.Email)); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := action(r.Context(), req.Email); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// ConfirmPasswordReset traite POST /auth/password-reset/confirm.
func (h *Handler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.Service.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Login traite POST /auth/login et pose le cookie de session.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := h.allow(h.loginPerIP, clientKey(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identifierKey := "identifier:" + normalizeKey(req.Identifier)
	if err := h.allow(h.loginPerIdentifier, identifierKey); err != nil {
		h.fail(w, r, err)
		return
	}

	result, err := h.Service.Login(r.Context(), LoginInput{
		Identifier: req.Identifier, Password: req.Password, RememberMe: req.RememberMe,
		IP: httpx.ClientIPFrom(r.Context()), UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.loginPerIdentifier.Reset(identifierKey)
	h.Cookies.Set(w, result.Session.Token, result.Session.ExpiresAt, h.Clock.Now())
	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		User:      NewUserResponse(result.User),
		CSRFToken: h.CSRF.Token(result.Session.Token),
		ExpiresAt: result.Session.ExpiresAt,
	})
}

// Logout traite POST /auth/logout (route protégée : session et jeton CSRF exigés).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := h.Cookies.Read(r); ok {
		if err := h.Service.Logout(r.Context(), token); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	h.Cookies.Clear(w)
	httpx.WriteNoContent(w)
}

// LogoutAll traite POST /auth/logout-all (route protégée).
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	if err := h.Service.LogoutAll(r.Context(), principal.UserID, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	h.Cookies.Clear(w)
	httpx.WriteNoContent(w)
}

// Me traite GET /auth/me (route protégée) : l'utilisateur courant et son jeton CSRF.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	current, err := h.Service.Me(r.Context(), principal.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	sessionToken, _ := h.Cookies.Read(r)
	httpx.WriteJSON(w, http.StatusOK, meResponse{User: NewUserResponse(current), CSRFToken: h.CSRF.Token(sessionToken)})
}

// allow compte une requête pour une clé et traduit un dépassement en erreur 429.
func (h *Handler) allow(limiter *ratelimit.Limiter, key string) error {
	if allowed, retryAfter := limiter.Allow(key); !allowed {
		return httpx.RateLimited(retryAfter)
	}
	return nil
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.Logger, h.toHTTPError(err))
}

// toHTTPError traduit les erreurs métier en erreurs HTTP. Les autres (validation, interne)
// sont laissées à httpx.Fail.
func (h *Handler) toHTTPError(err error) error {
	var locked *LockedError
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return httpx.NewError(http.StatusUnauthorized, CodeInvalidCredentials, "Identifiant ou mot de passe incorrect.")
	case errors.Is(err, ErrAccountNotActivated):
		return httpx.NewError(http.StatusForbidden, CodeNotActivated, "Votre compte n'est pas encore activé : consultez le mail d'activation.")
	case errors.Is(err, ErrAccountDeactivated):
		return httpx.NewError(http.StatusForbidden, CodeDeactivated, "Ce compte a été désactivé.")
	case errors.As(err, &locked):
		apiErr := httpx.RateLimited(max(locked.Until.Sub(h.Clock.Now()), minRetryAfter))
		apiErr.Code, apiErr.Message = CodeAccountLocked, "Compte temporairement verrouillé après trop d'échecs de connexion."
		return apiErr
	case errors.Is(err, ErrTokenInvalid):
		return httpx.NewError(http.StatusBadRequest, CodeInvalidToken, "Ce lien est invalide, expiré ou déjà utilisé.")
	case errors.Is(err, user.ErrEmailTaken):
		return httpx.Conflict(CodeEmailAlreadyUsed, "Cette adresse email est déjà utilisée.")
	}
	return err
}

// clientKey identifie le client pour la limitation de débit.
func clientKey(r *http.Request) string {
	if ip := httpx.ClientIPFrom(r.Context()); ip.IsValid() {
		return "ip:" + ip.String()
	}
	return "ip:" + unknownClient
}

// normalizeKey met une saisie sous une forme stable pour servir de clé de limitation.
func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
