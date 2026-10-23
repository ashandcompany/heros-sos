package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerIncidentRoutes(router *Router, c *components) {
	h := c.incidentHandler
	router.Handle(http.MethodPost, "/incidents", c.protected(c.idempotent.Wrap(h.Create)))
	router.HandleFunc(http.MethodGet, "/incidents", h.List)
	router.HandleFunc(http.MethodGet, "/incidents/{id}", h.Get)
	router.Handle(http.MethodPut, "/admin/incidents/{id}/moderation", c.protected(h.Moderate, user.RoleModerator))
	router.Handle(http.MethodPatch, "/admin/incidents/{id}", c.protected(h.Update, user.RoleModerator))
	router.Handle(http.MethodPut, "/hero/incidents/{id}/priority", c.protected(h.SetPriority, user.RoleHero))
	router.HandleFunc(http.MethodGet, "/heroes/{heroId}/incidents", h.HeroIncidents)
	router.HandleFunc(http.MethodGet, "/villains/{id}/incidents", h.VillainIncidents)
}
