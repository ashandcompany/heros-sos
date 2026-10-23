package contact

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
)

// CodeDeliveryUnavailable est le code d'erreur d'un message qui n'a pas pu être envoyé.
const CodeDeliveryUnavailable = "contact_delivery_unavailable"

// HandlerDeps regroupe les dépendances du handler.
type HandlerDeps struct {
	Service *Service
	Logger  *slog.Logger
	HTTP    config.HTTP
	Limits  config.RateLimits
	Clock   clock.Clock
}

// Handler expose le formulaire de contact.
type Handler struct {
	HandlerDeps
	sendPerIP *ratelimit.Limiter
}

// NewHandler crée le handler et son limiteur de débit.
func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{
		HandlerDeps: deps,
		sendPerIP:   ratelimit.New(deps.Limits.ContactPerIP, deps.Limits.MaxTrackedKeys, deps.Clock),
	}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées.
func (h *Handler) SweepLimiters() { h.sendPerIP.Sweep() }

type sendRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// Send traite POST /contact/messages.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if allowed, retryAfter := h.sendPerIP.Allow(httpx.ClientKey(r)); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	var body sendRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	err := h.Service.Send(r.Context(), Input(body))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrDeliveryUnavailable) {
		// Le détail (file saturée ou fermée) n'intéresse que les journaux.
		h.Logger.ErrorContext(r.Context(), "message de contact non transmis", "error", err)
		err = httpx.NewError(http.StatusServiceUnavailable, CodeDeliveryUnavailable,
			"Le message n'a pas pu être envoyé. Veuillez réessayer plus tard.")
	}
	httpx.Fail(w, r, h.Logger, err)
}
