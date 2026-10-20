package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerIncidentRoutes(router *Router, c *components) {
	h := c.incidentHandler
	router.Handle(http.MethodPost, "/incidents", c.protected(h.Create))
	router.Handle(http.MethodGet, "/incidents", c.protected(h.List, user.RoleCitizen, user.RoleHero, user.RoleModerator))
	router.Handle(http.MethodGet, "/incidents/{id}", c.protected(h.Get, user.RoleHero, user.RoleModerator))
}
