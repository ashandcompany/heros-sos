package blog

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

// CodeSlugTaken est le code d'erreur d'un slug qu'on n'a pas pu rendre unique.
const CodeSlugTaken = "article_slug_already_used"

// Handler expose les endpoints du blog.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

// NewHandler crée les handlers.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type payload struct {
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// SummaryResponse est un article sans son contenu (listes).
type SummaryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Excerpt     string     `json:"excerpt"`
	Status      string     `json:"status"`
	AuthorName  string     `json:"authorName"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	PublishedAt *time.Time `json:"publishedAt"`
}

// Response est un article complet.
type Response struct {
	SummaryResponse
	Content string `json:"content"`
}

func newSummary(a Article) SummaryResponse {
	return SummaryResponse{
		ID: a.ID, Title: a.Title, Slug: a.Slug, Excerpt: a.Excerpt, Status: string(a.Status),
		AuthorName: a.AuthorName, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, PublishedAt: a.PublishedAt,
	}
}

func newResponse(a Article) Response { return Response{SummaryResponse: newSummary(a), Content: a.Content} }

func writeSummaries(w http.ResponseWriter, articles []Article, total int64, page httpx.Page) {
	items := make([]SummaryResponse, len(articles))
	for i, a := range articles {
		items[i] = newSummary(a)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

// PublicList traite GET /articles.
func (h *Handler) PublicList(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	articles, total, err := h.service.ListPublished(r.Context(), page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeSummaries(w, articles, total, page)
}

// PublicGet traite GET /articles/{slug}.
func (h *Handler) PublicGet(w http.ResponseWriter, r *http.Request) {
	article, err := h.service.GetPublished(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newResponse(*article))
}

// AdminList traite GET /admin/articles?status=&limit=&offset=.
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	articles, total, err := h.service.ListAll(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")), page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeSummaries(w, articles, total, page)
}

// AdminGet traite GET /admin/articles/{id}.
func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	article, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newResponse(*article))
}

// Create traite POST /admin/articles.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body payload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	created, err := h.service.Create(r.Context(), principal.UserID, Input(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newResponse(*created))
}

// Update traite PUT /admin/articles/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var body payload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	updated, err := h.service.Update(r.Context(), principal.UserID, id, Input(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newResponse(*updated))
}

// Delete traite DELETE /admin/articles/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
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
		return httpx.NotFound("Article introuvable.")
	case errors.Is(err, ErrSlugTaken):
		return httpx.Conflict(CodeSlugTaken, "Impossible de générer une adresse unique pour cet article.")
	}
	return err
}
