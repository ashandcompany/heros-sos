package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerFilmRoutes(router *Router, c *components) {
	h := c.filmHandler
	router.HandleFunc(http.MethodGet, "/films", h.PublicFilms)
	router.HandleFunc(http.MethodGet, "/heroes/{heroId}/films", h.PublicHeroFilms)
	router.HandleFunc(http.MethodGet, "/villains/{villainId}/films", h.PublicVillainFilms)
	router.Handle(http.MethodGet, "/imdb/titles", c.protected(h.SearchTitles, user.RoleHero, user.RoleModerator))
	router.Handle(http.MethodGet, "/imdb/titles/{id}", c.protected(h.GetTitle, user.RoleHero, user.RoleModerator))

	router.Handle(http.MethodGet, "/hero/films", c.protected(h.MyFilms, user.RoleHero))
	router.Handle(http.MethodPost, "/hero/films", c.protected(h.AddMyFilm, user.RoleHero))
	router.Handle(http.MethodDelete, "/hero/films/{filmId}", c.protected(h.DeleteMyFilm, user.RoleHero))

	router.Handle(http.MethodGet, "/admin/hero-films", c.protected(h.AdminHeroFilms, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/hero-films", c.protected(h.AddAdminHeroFilm, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/hero-films/{heroId}/{filmId}/approve", c.protected(h.ApproveHeroFilm, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/hero-films/{heroId}/{filmId}/reject", c.protected(h.RejectHeroFilm, user.RoleModerator))
	router.Handle(http.MethodDelete, "/admin/hero-films/{heroId}/{filmId}", c.protected(h.DeleteAdminHeroFilm, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/villain-films", c.protected(h.AddAdminVillainFilm, user.RoleModerator))
	router.Handle(http.MethodDelete, "/admin/villains/{villainId}/films/{filmId}", c.protected(h.DeleteAdminVillainFilm, user.RoleModerator))
}
