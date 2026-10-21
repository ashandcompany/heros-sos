package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

func registerBlogRoutes(router *Router, c *components) {
	h := c.blogHandler
	router.HandleFunc(http.MethodGet, "/articles", h.PublicList)
	router.HandleFunc(http.MethodGet, "/articles/{slug}", h.PublicGet)

	router.Handle(http.MethodGet, "/admin/articles", c.protected(h.AdminList, user.RoleModerator))
	router.Handle(http.MethodPost, "/admin/articles", c.protected(h.Create, user.RoleModerator))
	router.Handle(http.MethodGet, "/admin/articles/{id}", c.protected(h.AdminGet, user.RoleModerator))
	router.Handle(http.MethodPut, "/admin/articles/{id}", c.protected(h.Update, user.RoleModerator))
	router.Handle(http.MethodDelete, "/admin/articles/{id}", c.protected(h.Delete, user.RoleModerator))
}
