package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerInterventionTypeRoutes(router *Router, c *components) {
	h := c.typeHandler
	router.Handle(http.MethodGet, "/intervention-types", c.protected(h.ListActive))

	router.Handle(http.MethodGet, "/admin/intervention-types", c.protected(h.ListAll, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/intervention-types", c.protected(h.Create, user.RoleModerator))
}
