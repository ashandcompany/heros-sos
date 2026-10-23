package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// registerPublicMailRoutes branche les formulaires publics qui passent par le Mailer : le contact et la
// newsletter. Aucune session n'est exigée ; la limitation de débit par IP (et par adresse pour la
// newsletter) tient lieu de protection contre l'envoi massif de mails.
func registerPublicMailRoutes(router *Router, c *components) {
	router.HandleFunc(http.MethodPost, "/contact/messages", c.contactHandler.Send)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions", c.newsletterHandler.Subscribe)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions/confirm", c.newsletterHandler.Confirm)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions/unsubscribe", c.newsletterHandler.Unsubscribe)

	h := c.newsletterHandler
	router.Handle(http.MethodGet, "/admin/newsletters", c.protected(h.ListIssues, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/newsletters", c.protected(h.CreateIssue, user.RoleModerator))
	router.Handle(http.MethodGet, "/admin/newsletters/{id}", c.protected(h.GetIssue, user.RoleModerator))
	router.Handle(http.MethodPut, "/admin/newsletters/{id}", c.protected(h.UpdateIssue, user.RoleModerator))
	router.Handle(http.MethodDelete, "/admin/newsletters/{id}", c.protected(h.DeleteIssue, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/newsletters/{id}/send", c.protected(h.SendIssue, user.RoleModerator))
}
