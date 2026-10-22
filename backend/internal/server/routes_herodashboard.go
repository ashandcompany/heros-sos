package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerHeroDashboardRoutes(router *Router, c *components) {
	router.Handle(http.MethodGet, "/hero/dashboard", c.protected(c.dashboardHandler.Get, user.RoleHero))
}
