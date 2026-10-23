package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerHeroLocationRoutes(router *Router, c *components) {
	router.Handle(http.MethodPut, "/hero/location", c.protected(c.locationHandler.Put, user.RoleHero))
	router.Handle(http.MethodDelete, "/hero/location", c.protected(c.locationHandler.Delete, user.RoleHero))
	router.Handle(http.MethodPut, "/hero/location/consent", c.protected(c.locationHandler.PutConsent, user.RoleHero))
	router.Handle(http.MethodDelete, "/hero/location/consent", c.protected(c.locationHandler.DeleteConsent, user.RoleHero))
}
