package auth

import (
	"net/http"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// CookieManager pose, lit et efface le cookie de session. Tous ses attributs sont définis ici,
// une seule fois : HttpOnly (illisible en JavaScript), Path=/ et aucun Domain (le cookie ne
// concerne que l'hôte exact), SameSite=Lax, Secure selon la configuration.
type CookieManager struct {
	name   string
	secure bool
}

// NewCookieManager construit le gestionnaire depuis la configuration des sessions.
func NewCookieManager(cfg config.Session) CookieManager {
	return CookieManager{name: cfg.CookieName(), secure: cfg.CookieSecure}
}

// Name retourne le nom du cookie.
func (c CookieManager) Name() string { return c.name }

// Set envoie le cookie de session, valable jusqu'à expires.
func (c CookieManager) Set(w http.ResponseWriter, token string, expires, now time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   max(int(expires.Sub(now).Seconds()), 1),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear demande au navigateur d'oublier le cookie.
func (c CookieManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Read retourne la valeur du cookie de session de la requête.
func (c CookieManager) Read(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(c.name)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}
