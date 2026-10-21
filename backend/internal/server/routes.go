package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// registerRoutes branche les routes de chaque domaine. Pour ajouter un domaine, écrire sa
// fonction d'enregistrement dans un fichier routes_<domaine>.go et l'appeler ici : les autres
// domaines ne sont pas touchés (pas de conflit de fusion entre développeurs).
func registerRoutes(router *Router, c *components) {
	registerHealthRoutes(router, c)
	registerAuthRoutes(router, c)
	registerHeroRoutes(router, c)
	registerModerationRoutes(router, c)
	registerDocsRoutes(router, c)
	registerInterventionTypeRoutes(router, c)
	registerVillainRoutes(router, c)
	registerIncidentRoutes(router, c)
	registerInterventionRoutes(router, c)
	registerReputationRoutes(router, c)
	registerFilmRoutes(router, c)
	registerAccountRoutes(router, c)
	registerPublicMailRoutes(router, c)
	registerBlogRoutes(router, c)
}

// protected exige une session valide (et le jeton CSRF sur les écritures) ; avec des rôles, il
// restreint en plus l'accès à ces rôles.
func (c *components) protected(h http.HandlerFunc, roles ...user.Role) http.Handler {
	return c.authMiddleware.Protected(roles...)(h)
}

func registerHealthRoutes(router *Router, c *components) {
	router.HandleFunc(http.MethodGet, "/health", c.health.live)
	router.HandleFunc(http.MethodGet, "/ready", c.health.ready)
}

func registerAuthRoutes(router *Router, c *components) {
	h := c.authHandler
	router.HandleFunc(http.MethodPost, "/auth/register", h.Register)
	router.HandleFunc(http.MethodPost, "/auth/activate", h.Activate)
	router.HandleFunc(http.MethodPost, "/auth/resend-activation", h.ResendActivation)
	router.HandleFunc(http.MethodPost, "/auth/login", h.Login)
	router.HandleFunc(http.MethodPost, "/auth/password-reset", h.RequestPasswordReset)
	router.HandleFunc(http.MethodPost, "/auth/password-reset/confirm", h.ConfirmPasswordReset)

	router.Handle(http.MethodPost, "/auth/logout", c.protected(h.Logout))
	router.Handle(http.MethodPost, "/auth/logout-all", c.protected(h.LogoutAll))
	router.Handle(http.MethodGet, "/auth/me", c.protected(h.Me))
}

func registerHeroRoutes(router *Router, c *components) {
	h := c.heroHandler
	router.HandleFunc(http.MethodGet, "/heroes", h.PublicList)
	router.HandleFunc(http.MethodGet, "/heroes/{heroId}", h.PublicGet)

	// Tous les rôles connectés passent : c'est le service qui répond « déjà héros » (409) ou
	// « compte non éligible » (403, modérateur), avec un code d'erreur explicite.
	router.Handle(http.MethodPost, "/heroes/requests", c.protected(h.Request))
	router.Handle(http.MethodGet, "/heroes/requests/me", c.protected(h.Mine))

	router.Handle(http.MethodGet, "/admin/hero-requests", c.protected(h.List, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/hero-requests/{userId}/approve", c.protected(h.Approve, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/hero-requests/{userId}/reject", c.protected(h.Reject, user.RoleModerator))
}

func registerModerationRoutes(router *Router, c *components) {
	h := c.moderationHandler
	router.Handle(http.MethodGet, "/admin/users", c.protected(h.ListUsers, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/users/{id}/deactivate", c.protected(h.Deactivate, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/users/{id}/reactivate", c.protected(h.Reactivate, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/users/{id}/verify", c.protected(h.Verify, user.RoleModerator))
}
