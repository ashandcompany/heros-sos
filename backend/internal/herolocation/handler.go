package herolocation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Handler expose le partage de position des héros.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
	// perUser limite les envois de position d'un même héros.
	perUser *ratelimit.Limiter
}

// NewHandler crée le handler.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP, perUser *ratelimit.Limiter) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig, perUser: perUser}
}

// SweepLimiters libère la mémoire des fenêtres de limitation expirées.
func (h *Handler) SweepLimiters() {
	h.perUser.Sweep()
}

type payload struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	AccuracyM *int     `json:"accuracyM"`
}

// CodeConsentRequired : la position ne peut être envoyée qu'après consentement explicite.
const CodeConsentRequired = "hero_location_consent_required"

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrConsentRequired):
		err = httpx.Conflict(CodeConsentRequired, "Le partage de position n'est pas activé.")
	case errors.Is(err, ErrNotHero):
		err = httpx.NotFound("Profil de héros introuvable.")
	}
	httpx.Fail(w, r, h.logger, err)
}

// PutConsent traite PUT /hero/location/consent : le héros accepte de partager sa position.
func (h *Handler) PutConsent(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.service.Consent(r.Context(), principal.UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// DeleteConsent traite DELETE /hero/location/consent : retire le consentement et supprime la position.
func (h *Handler) DeleteConsent(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.service.Revoke(r.Context(), principal.UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Put traite PUT /hero/location : remplace la position du héros connecté.
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if allowed, retryAfter := h.perUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	var body payload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.Latitude == nil || body.Longitude == nil {
		httpx.Fail(w, r, h.logger, &validation.Error{Fields: map[string]string{"location": validation.CodeRequired}})
		return
	}
	in := Input{Latitude: *body.Latitude, Longitude: *body.Longitude, AccuracyM: body.AccuracyM}
	if err := h.service.Set(r.Context(), principal.UserID, in); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Delete traite DELETE /hero/location : supprime la position, le consentement est conservé.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if err := h.service.Clear(r.Context(), principal.UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}
