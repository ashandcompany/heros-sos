package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerInterventionRoutes(router *Router, c *components) {
	h := c.interventionHandler
	router.Handle(http.MethodGet, "/incidents/{incidentId}/interventions", c.protected(h.ListByIncident, user.RoleHero, user.RoleModerator))
	router.Handle(http.MethodPost, "/incidents/{incidentId}/interventions", c.protected(c.idempotent.Wrap(h.Start), user.RoleHero))
	router.Handle(http.MethodGet, "/interventions/{id}", c.protected(h.Get, user.RoleHero, user.RoleModerator))
	router.Handle(http.MethodPost, "/interventions/{id}/join", c.protected(c.idempotent.Wrap(h.Join), user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{id}/invitations", c.protected(c.idempotent.Wrap(h.Invite), user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{id}/invitations/me/confirm", c.protected(c.idempotent.Wrap(h.ConfirmInvitation), user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{id}/invitations/me/decline", c.protected(c.idempotent.Wrap(h.DeclineInvitation), user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{id}/outcome", c.protected(c.idempotent.Wrap(h.ProposeOutcome), user.RoleHero))
	router.Handle(http.MethodPost, "/interventions/{id}/outcome/validate", c.protected(c.idempotent.Wrap(h.ValidateOutcome), user.RoleHero))
	router.Handle(http.MethodPost, "/admin/interventions/{id}/decision", c.protected(c.idempotent.Wrap(h.Arbitrate), user.RoleModerator))
}
