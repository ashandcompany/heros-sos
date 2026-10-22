package server

import "net/http"

// registerEventRoutes branche le flux temps réel. Tout utilisateur connecté peut l'ouvrir : ce qu'il y reçoit
// dépend de son rôle (voir realtime.visibleTo).
func registerEventRoutes(router *Router, c *components) {
	router.Handle(http.MethodGet, "/events/stream", c.protected(c.streamEvents.Stream))
}
