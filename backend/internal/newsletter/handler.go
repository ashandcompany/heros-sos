package newsletter

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

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

// Handler expose les endpoints publics de la newsletter.
type Handler struct {
	HandlerDeps

	subscribePerIP    *ratelimit.Limiter
	subscribePerEmail *ratelimit.Limiter
}

// NewHandler crée le handler et ses limiteurs de débit.
func NewHandler(deps HandlerDeps) *Handler {
	limiter := func(rule config.RateRule) *ratelimit.Limiter {
		return ratelimit.New(rule, deps.Limits.MaxTrackedKeys, deps.Clock)
	}
	return &Handler{
		HandlerDeps:       deps,
		subscribePerIP:    limiter(deps.Limits.NewsletterPerIP),
		subscribePerEmail: limiter(deps.Limits.NewsletterPerEmail),
	}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées.
func (h *Handler) SweepLimiters() {
	h.subscribePerIP.Sweep()
	h.subscribePerEmail.Sweep()
}

type subscribeRequest struct {
	Email string `json:"email"`
}

type confirmRequest struct {
	Token string `json:"token"`
}

// Subscribe traite POST /newsletter/subscriptions. La limitation par adresse protège la boîte de
// quelqu'un d'autre : sans elle, le formulaire servirait à lui envoyer des mails en rafale.
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if allowed, retryAfter := h.subscribePerIP.Allow(httpx.ClientKey(r)); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	var body subscribeRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if allowed, retryAfter := h.subscribePerEmail.Allow("email:" + strings.ToLower(strings.TrimSpace(body.Email))); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	if err := h.Service.Subscribe(r.Context(), body.Email); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// Confirm traite POST /newsletter/subscriptions/confirm.
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var body confirmRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.Service.Confirm(r.Context(), body.Token); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrLinkInvalid) {
		err = httpx.NewError(http.StatusBadRequest, auth.CodeInvalidToken, "Ce lien est invalide, expiré ou déjà utilisé.")
	}
	httpx.Fail(w, r, h.Logger, err)
}
