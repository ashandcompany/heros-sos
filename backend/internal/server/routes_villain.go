package server

import "net/http"

func registerVillainRoutes(router *Router, c *components) {
	h := c.villainHandler
	router.Handle(http.MethodPost, "/villains", c.protected(h.Create))
	router.HandleFunc(http.MethodGet, "/villains", h.List)
	router.HandleFunc(http.MethodGet, "/villains/{id}", h.Get)
	router.Handle(http.MethodPatch, "/villains/{id}", c.protected(h.Update))
}
