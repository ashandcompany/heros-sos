package reputation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

const (
	CodeAlreadyReviewed  = "hero_already_reviewed"
	CodeNotEligible      = "hero_review_not_eligible"
	CodeReplyUnavailable = "hero_review_reply_unavailable"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type ratePayload struct {
	HeroID  uuid.UUID `json:"heroId"`
	Rating  int16     `json:"rating"`
	Comment string    `json:"comment"`
}

type replyPayload struct {
	Content string `json:"content"`
}

type statusPayload struct {
	Status string `json:"status"`
}

func (h *Handler) ListCitizenInterventions(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListCitizenInterventions(r.Context(), principal.UserID, Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

func (h *Handler) Rate(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	interventionID, ok := h.pathID(w, r, "interventionId")
	if !ok {
		return
	}
	var payload ratePayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.Rate(r.Context(), principal.UserID, interventionID, payload.HeroID,
		payload.Rating, payload.Comment, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) ListHeroReviews(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	result, err := h.service.ListHeroReviews(r.Context(), principal.UserID, Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	reviewID, ok := h.pathID(w, r, "reviewId")
	if !ok {
		return
	}
	var payload replyPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.Reply(r.Context(), principal.UserID, reviewID, payload.Content, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) PublicReviews(w http.ResponseWriter, r *http.Request) {
	heroID, ok := h.pathID(w, r, "heroId")
	if !ok {
		return
	}
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	result, err := h.service.PublicReviews(r.Context(), heroID, Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ListModerationReviews(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListModerationReviews(r.Context(), r.URL.Query().Get("status"), Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

func (h *Handler) ListModerationReplies(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListModerationReplies(r.Context(), r.URL.Query().Get("status"), Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

func (h *Handler) ModerateReview(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	reviewID, ok := h.pathID(w, r, "reviewId")
	if !ok {
		return
	}
	var payload statusPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.ModerateReview(r.Context(), principal.UserID, reviewID, payload.Status, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) ModerateReply(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	reviewID, ok := h.pathID(w, r, "reviewId")
	if !ok {
		return
	}
	var payload statusPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &payload); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.ModerateReply(r.Context(), principal.UserID, reviewID, payload.Status, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) pathID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(key))
	if err != nil {
		h.fail(w, r, httpx.NotFound("Ressource introuvable."))
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
		return httpx.NotFound("Avis ou profil héros introuvable.")
	case errors.Is(err, ErrNotEligible):
		return httpx.NewError(http.StatusForbidden, CodeNotEligible, "Vous ne pouvez pas noter ce héros ou répondre à cet avis.")
	case errors.Is(err, ErrAlreadyReviewed):
		return httpx.Conflict(CodeAlreadyReviewed, "Ce héros a déjà été noté pour cette intervention.")
	case errors.Is(err, ErrReplyUnavailable):
		return httpx.Conflict(CodeReplyUnavailable, "Ce commentaire ne peut pas recevoir de réponse.")
	}
	return err
}
