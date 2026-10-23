package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// avatarRoutePrefix précède le nom d'une photo de profil dans son adresse publique.
const avatarRoutePrefix = "/avatars/"

func registerAccountRoutes(router *Router, c *components) {
	h := c.accountHandler
	// Tous les rôles connectés ont un profil, sauf le changement d'adresse : celui d'un modérateur se
	// gère hors ligne, comme son compte (personne ne pourrait valider sa demande).
	router.Handle(http.MethodGet, "/me", c.protected(h.Get))
	router.Handle(http.MethodPatch, "/me", c.protected(h.UpdateContact))
	router.Handle(http.MethodPut, "/me/avatar", c.protected(h.UpdateAvatar))
	router.Handle(http.MethodDelete, "/me/avatar", c.protected(h.RemoveAvatar))
	router.Handle(http.MethodGet, "/me/export", c.protected(h.Export))
	router.Handle(http.MethodDelete, "/me", c.protected(h.DeleteAccount))
	router.Handle(http.MethodPost, "/admin/users/{id}/anonymize", c.protected(h.Anonymize, user.RoleModerator))
	router.Handle(http.MethodPost, "/me/email-change-requests", c.protected(h.RequestEmailChange, user.RoleCitizen, user.RoleHero))
	router.Handle(http.MethodPut, "/me/newsletter", c.protected(h.SetNewsletter))
	// Publique : le lien de confirmation s'ouvre souvent sur un autre appareil, et se prouve par sa signature.
	router.HandleFunc(http.MethodPost, "/email-change-requests/confirm", h.ConfirmEmailChange)

	// Publique, comme les photos des héros dans l'encyclopédie (voir avatar.Handler).
	router.HandleFunc(http.MethodGet, avatarRoutePrefix+"{name}", c.avatarHandler.Serve)

	router.Handle(http.MethodGet, "/admin/email-change-requests", c.protected(h.ListEmailChanges, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/email-change-requests/{id}/approve", c.protected(h.ApproveEmailChange, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/email-change-requests/{id}/reject", c.protected(h.RejectEmailChange, user.RoleModerator))
}
