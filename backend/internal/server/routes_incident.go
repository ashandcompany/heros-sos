package server

import "net/http"

func registerIncidentRoutes(router *Router, c *components) {
	h := c.incidentHandler
	router.Handle(http.MethodPost, "/incidents", c.protected(h.Create))
	router.HandleFunc(http.MethodGet, "/incidents", h.List)
	router.HandleFunc(http.MethodGet, "/incidents/{id}", h.Get)
	router.HandleFunc(http.MethodGet, "/heroes/{heroId}/incidents", h.HeroIncidents)
	router.HandleFunc(http.MethodGet, "/villains/{id}/incidents", h.VillainIncidents)
}
