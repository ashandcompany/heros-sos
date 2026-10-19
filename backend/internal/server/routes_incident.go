package server

import "net/http"

func registerIncidentRoutes(router *Router, c *components) {
router.Handle(http.MethodPost, "/incidents", c.protected(c.incidentHandler.Create))
}
