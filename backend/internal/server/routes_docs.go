package server

import (
	"github.com/ashandcompany/heros-sos/backend/internal/apidocs"
	"github.com/ashandcompany/heros-sos/backend/openapi"
)

// registerDocsRoutes sert la documentation interactive de l'API (Swagger UI), sous APIPrefix pour
// que le reverse proxy la route avec le reste de l'API. Elle n'existe pas en production : la page
// a une politique de sécurité plus souple que le reste de l'API, et elle partagerait l'origine du
// site (donc de la session).
func registerDocsRoutes(router *Router, c *components) {
	if c.cfg.IsProduction() {
		return
	}
	apidocs.New(openapi.Spec).Register(router.HandleOutsideContract)
	c.logger.Info("documentation de l'API disponible", "path", APIPrefix+apidocs.PagePath)
}
