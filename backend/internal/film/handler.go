package film

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

const (
	CodeFilmNotFound       = "film_not_found"
	CodeFilmAlreadyLinked  = "film_already_linked"
	CodeFilmNotPending     = "film_not_pending"
	CodeCatalogUnavailable = "catalog_unavailable"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type filmPayload struct {
	IMDBID      string `json:"imdbId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	VideoURL    string `json:"videoUrl"`
}

type adminHeroFilmPayload struct {
	HeroID      uuid.UUID `json:"heroId"`
	IMDBID      string    `json:"imdbId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	VideoURL    string    `json:"videoUrl"`
}

type adminVillainFilmPayload struct {
	VillainID uuid.UUID `json:"villainId"`
	FilmID    uuid.UUID `json:"filmId"`
	IMDBID    string    `json:"imdbId"`
}

type HeroSummary struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

type VillainSummary struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

type Response struct {
	ID          uuid.UUID       `json:"id"`
	IMDBID      string          `json:"imdbId,omitempty"`
	VideoURL    string          `json:"videoUrl,omitempty"`
	Title       string          `json:"title"`
	ReleaseYear *int16          `json:"releaseYear,omitempty"`
	PosterURL   string          `json:"posterUrl,omitempty"`
	Plot        string          `json:"plot"`
	AddedAt     time.Time       `json:"addedAt"`
	Moderation  string          `json:"moderation,omitempty"`
	Hero        *HeroSummary    `json:"hero,omitempty"`
	Villain     *VillainSummary `json:"villain,omitempty"`
}

type titleResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Year      *int16 `json:"year,omitempty"`
	Plot      string `json:"plot"`
	PosterURL string `json:"posterUrl,omitempty"`
}

func newTitleResponse(title Title) titleResponse {
	return titleResponse{ID: title.ID, Title: title.Title, Year: title.ReleaseYear, Plot: title.Plot, PosterURL: title.PosterURL}
}

func newHeroResponse(item HeroFilm) Response {
	return Response{
		ID: item.ID, IMDBID: item.IMDBID, VideoURL: item.VideoURL, Title: item.Title,
		ReleaseYear: item.ReleaseYear, PosterURL: item.PosterURL, Plot: item.Plot,
		AddedAt: item.AddedAt, Moderation: item.Status,
		Hero: &HeroSummary{ID: item.HeroID, Alias: item.HeroAlias},
	}
}

func newVillainResponse(item VillainFilm) Response {
	return Response{
		ID: item.ID, IMDBID: item.IMDBID, VideoURL: item.VideoURL, Title: item.Title,
		ReleaseYear: item.ReleaseYear, PosterURL: item.PosterURL, Plot: item.Plot,
		AddedAt: item.AddedAt,
		Villain: &VillainSummary{ID: item.VillainID, Alias: item.VillainAlias},
	}
}

func (h *Handler) PublicFilms(w http.ResponseWriter, r *http.Request) {
	var heroID *uuid.UUID
	if raw := r.URL.Query().Get("hero"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			h.fail(w, r, &httpx.Error{Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
				Message: "Certains paramètres sont invalides.", Fields: map[string]string{"hero": "invalid"}})
			return
		}
		heroID = &parsed
	}
	h.listHeroFilms(w, r, heroID, StatusApproved)
}

func (h *Handler) PublicHeroFilms(w http.ResponseWriter, r *http.Request) {
	heroID, ok := pathID(w, r, "heroId")
	if !ok {
		h.fail(w, r, ErrNotFound)
		return
	}
	h.listHeroFilms(w, r, &heroID, StatusApproved)
}

func (h *Handler) listHeroFilms(w http.ResponseWriter, r *http.Request, heroID *uuid.UUID, status string) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListHeroFilms(r.Context(), heroID, status, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	responses := make([]Response, len(items))
	for index, item := range items {
		responses[index] = newHeroResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(responses, total, page))
}

func (h *Handler) PublicVillainFilms(w http.ResponseWriter, r *http.Request) {
	villainID, ok := pathID(w, r, "villainId")
	if !ok {
		h.fail(w, r, ErrNotFound)
		return
	}
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListVillainFilms(r.Context(), villainID, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	responses := make([]Response, len(items))
	for index, item := range items {
		responses[index] = newVillainResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(responses, total, page))
}

func (h *Handler) SearchTitles(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if principal.Role != user.RoleHero && principal.Role != user.RoleModerator {
		h.fail(w, r, ErrNotAllowed)
		return
	}
	items, err := h.service.Search(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	responses := make([]titleResponse, len(items))
	for index, item := range items {
		responses[index] = newTitleResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) GetTitle(w http.ResponseWriter, r *http.Request) {
	title, err := h.service.FindTitle(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newTitleResponse(*title))
}

func (h *Handler) MyFilms(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items, total, err := h.service.ListHeroFilms(r.Context(), &principal.UserID, "", page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	responses := make([]Response, len(items))
	for index, item := range items {
		responses[index] = newHeroResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(responses, total, page))
}

func (h *Handler) AddMyFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body filmPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	film, err := h.service.AddToHero(r.Context(), principal.UserID, principal.Role, uuid.Nil,
		FilmInput{IMDBID: body.IMDBID, Title: body.Title, Description: body.Description, VideoURL: body.VideoURL},
		httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, Response{ID: film.ID, IMDBID: film.IMDBID, VideoURL: film.VideoURL,
		Title: film.Title, ReleaseYear: film.ReleaseYear, PosterURL: film.PosterURL, Plot: film.Plot, AddedAt: film.CreatedAt})
}

func (h *Handler) DeleteMyFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	filmID, ok := pathID(w, r, "filmId")
	if !ok {
		h.fail(w, r, ErrNotFound)
		return
	}
	if err := h.service.DeleteHeroFilm(r.Context(), principal.UserID, uuid.Nil, filmID, principal.Role, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) AdminHeroFilms(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	items, total, err := h.service.ListHeroFilms(r.Context(), nil, status, page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	responses := make([]Response, len(items))
	for index, item := range items {
		responses[index] = newHeroResponse(item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(responses, total, page))
}

func (h *Handler) AddAdminHeroFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body adminHeroFilmPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.HeroID == uuid.Nil {
		h.fail(w, r, &httpx.Error{Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains champs sont invalides.", Fields: map[string]string{"heroId": "required"}})
		return
	}
	film, err := h.service.AddApprovedToHero(r.Context(), principal.UserID, body.HeroID,
		FilmInput{IMDBID: body.IMDBID, Title: body.Title, Description: body.Description, VideoURL: body.VideoURL}, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, Response{ID: film.ID, IMDBID: film.IMDBID, VideoURL: film.VideoURL,
		Title: film.Title, ReleaseYear: film.ReleaseYear, PosterURL: film.PosterURL, Plot: film.Plot,
		AddedAt: film.CreatedAt, Moderation: StatusPending})
}

func (h *Handler) ApproveHeroFilm(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, StatusApproved)
}
func (h *Handler) RejectHeroFilm(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, StatusRejected)
}

func (h *Handler) moderate(w http.ResponseWriter, r *http.Request, status string) {
	principal, _ := auth.PrincipalFrom(r.Context())
	heroID, filmID, ok := h.linkIDs(w, r)
	if !ok {
		return
	}
	if err := h.service.Moderate(r.Context(), principal.UserID, heroID, filmID, status, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) DeleteAdminHeroFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	heroID, filmID, ok := h.linkIDs(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteHeroFilm(r.Context(), principal.UserID, heroID, filmID, user.RoleModerator, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) AddAdminVillainFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body adminVillainFilmPayload
	if err := httpx.DecodeJSON(w, r, h.http.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	film, err := h.service.AddToVillain(r.Context(), principal.UserID, body.VillainID,
		FilmInput{FilmID: body.FilmID, IMDBID: body.IMDBID}, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, Response{ID: film.ID, IMDBID: film.IMDBID, VideoURL: film.VideoURL,
		Title: film.Title, ReleaseYear: film.ReleaseYear, PosterURL: film.PosterURL, Plot: film.Plot,
		AddedAt: film.CreatedAt, Moderation: StatusApproved})
}

func (h *Handler) DeleteAdminVillainFilm(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	villainID, ok := pathID(w, r, "villainId")
	if !ok {
		h.fail(w, r, ErrNotFound)
		return
	}
	filmID, ok := pathID(w, r, "filmId")
	if !ok {
		h.fail(w, r, ErrNotFound)
		return
	}
	if err := h.service.DeleteVillainFilm(r.Context(), principal.UserID, villainID, filmID, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) linkIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	heroID, err := uuid.Parse(r.PathValue("heroId"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return uuid.Nil, uuid.Nil, false
	}
	filmID, err := uuid.Parse(r.PathValue("filmId"))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return uuid.Nil, uuid.Nil, false
	}
	return heroID, filmID, true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	return id, err == nil
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrTitleNotFound):
		return httpx.NewError(http.StatusNotFound, CodeFilmNotFound, "Film introuvable dans le catalogue.")
	case errors.Is(err, ErrNotFound), errors.Is(err, hero.ErrNotFound), errors.Is(err, villain.ErrNotFound):
		return httpx.NotFound("Film ou fiche associée introuvable.")
	case errors.Is(err, ErrAlreadyLinked), errors.Is(err, ErrIMDBExists):
		return httpx.Conflict(CodeFilmAlreadyLinked, "Ce film est déjà associé.")
	case errors.Is(err, ErrAlreadyReview):
		return httpx.Conflict(CodeFilmNotPending, "Ce film a déjà été modéré.")
	case errors.Is(err, ErrNotAllowed):
		return httpx.Forbidden("Cette action n'est pas autorisée.")
	case errors.Is(err, ErrCatalogUnavailable):
		return httpx.NewError(http.StatusBadGateway, CodeCatalogUnavailable, "Le catalogue de films est indisponible.")
	}
	return err
}
