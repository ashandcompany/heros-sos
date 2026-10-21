package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerReputationRoutes(router *Router, c *components) {
	h := c.reputationHandler
	router.Handle(http.MethodGet, "/me/interventions", c.protected(h.ListCitizenInterventions, user.RoleCitizen, user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{interventionId}/reviews", c.protected(h.Rate, user.RoleCitizen, user.RoleHero))
	router.Handle(http.MethodGet, "/hero/reviews", c.protected(h.ListHeroReviews, user.RoleHero))
	router.Handle(http.MethodPut, "/hero/reviews/{reviewId}/reply", c.protected(h.Reply, user.RoleHero))
	router.HandleFunc(http.MethodGet, "/heroes/{heroId}/reviews", h.PublicReviews)
	router.Handle(http.MethodGet, "/admin/hero-reviews", c.protected(h.ListModerationReviews, user.RoleModerator))
	router.Handle(http.MethodPatch, "/admin/hero-reviews/{reviewId}", c.protected(h.ModerateReview, user.RoleModerator))
	router.Handle(http.MethodGet, "/admin/hero-review-replies", c.protected(h.ListModerationReplies, user.RoleModerator))
	router.Handle(http.MethodPatch, "/admin/hero-review-replies/{reviewId}", c.protected(h.ModerateReply, user.RoleModerator))
}
