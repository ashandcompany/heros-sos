package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

const (
	// CSRFHeader est l'en-tête qui porte le jeton CSRF sur les requêtes qui modifient des données.
	CSRFHeader = "X-CSRF-Token"

	csrfContext = "heros-sos/csrf/v1:"
)

// CSRF fabrique et vérifie les jetons de protection contre les requêtes intersites. Le jeton est
// un HMAC du jeton de session : il est lié à la session, ne demande aucun stockage et change à
// chaque connexion. Une page tierce ne peut ni le lire ni le deviner.
type CSRF struct {
	secret []byte
}

// NewCSRF crée un générateur avec un secret serveur.
func NewCSRF(secret []byte) *CSRF {
	return &CSRF{secret: append([]byte(nil), secret...)}
}

// Token retourne le jeton CSRF associé à un jeton de session.
func (c *CSRF) Token(sessionToken string) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(csrfContext + sessionToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Valid vérifie en temps constant qu'un jeton CSRF correspond à la session.
func (c *CSRF) Valid(sessionToken, provided string) bool {
	if sessionToken == "" || provided == "" {
		return false
	}
	return hmac.Equal([]byte(c.Token(sessionToken)), []byte(provided))
}
