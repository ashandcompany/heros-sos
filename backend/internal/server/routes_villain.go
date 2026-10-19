package server

import "net/http"

func registerVillainRoutes(router *Router, c *components) {
h := c.villainHandler
router.Handle(http.MethodPost, "/villains", c.protected(h.Create))
router.Handle(http.MethodGet, "/villains", c.protected(h.List))
}
