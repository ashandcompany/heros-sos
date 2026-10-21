package server

import "net/http"

// registerPublicMailRoutes branche les formulaires publics qui passent par le Mailer : le contact et la
// newsletter. Aucune session n'est exigée ; la limitation de débit par IP (et par adresse pour la
// newsletter) tient lieu de protection contre l'envoi massif de mails.
func registerPublicMailRoutes(router *Router, c *components) {
	router.HandleFunc(http.MethodPost, "/contact/messages", c.contactHandler.Send)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions", c.newsletterHandler.Subscribe)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions/confirm", c.newsletterHandler.Confirm)
	router.HandleFunc(http.MethodPost, "/newsletter/subscriptions/unsubscribe", c.newsletterHandler.Unsubscribe)
}
