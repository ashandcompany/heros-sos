package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Codes d'erreur propres à la protection des requêtes.
const (
	CodeOriginNotAllowed = "origin_not_allowed"
	CodeCSRFInvalid      = "csrf_invalid"
	// CodeMFARequired : la session n'est pas complète, la double authentification doit d'abord être validée
	// (ou activée, si elle est obligatoire).
	CodeMFARequired = "mfa_required"
)

// Middleware regroupe les protections HTTP liées à la session. Les autres domaines (incidents,
// modération…) y accrochent leurs routes avec Protected.
type Middleware struct {
	sessions *SessionService
	cookies  CookieManager
	csrf     *CSRF
	origins  map[string]struct{}
	logger   *slog.Logger
}

// NewMiddleware crée les middlewares. allowedOrigins est la liste d'origines autorisées à
// émettre des requêtes qui modifient des données.
func NewMiddleware(sessions *SessionService, cookies CookieManager, csrf *CSRF, allowedOrigins []string, logger *slog.Logger) *Middleware {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if normalized, ok := config.NormalizeOrigin(origin); ok {
			origins[normalized] = struct{}{}
		}
	}
	return &Middleware{sessions: sessions, cookies: cookies, csrf: csrf, origins: origins, logger: logger}
}

// Authenticate lit le cookie de session et, s'il est valide, place l'utilisateur dans le
// contexte. Il ne refuse jamais la requête : c'est Protected qui exige une session. Un cookie
// périmé est effacé pour que le navigateur cesse de l'envoyer.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, present := m.cookies.Read(r)
		if !present {
			next.ServeHTTP(w, r)
			return
		}

		principal, err := m.sessions.Authenticate(r.Context(), token)
		switch {
		case err == nil:
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		case errors.Is(err, ErrSessionInvalid):
			m.cookies.Clear(w)
			next.ServeHTTP(w, r)
		default:
			httpx.Fail(w, r, m.logger, err)
		}
	})
}

// OriginCheck refuse les requêtes qui modifient des données et ne viennent pas d'une origine
// autorisée. Les navigateurs envoient l'en-tête Origin sur toute requête POST, PUT, PATCH ou
// DELETE : une page tierce ne peut donc pas déclencher d'action avec le cookie de l'utilisateur.
// Les clients qui n'envoient pas Origin (serveur Next.js, outils) sont acceptés, sauf si le
// navigateur déclare la requête intersite via Sec-Fetch-Site.
func (m *Middleware) OriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isUnsafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		switch {
		case origin != "":
			normalized, valid := config.NormalizeOrigin(origin)
			if _, allowed := m.origins[normalized]; !valid || !allowed {
				httpx.WriteError(w, errOriginNotAllowed())
				return
			}
		case r.Header.Get("Sec-Fetch-Site") == "cross-site":
			httpx.WriteError(w, errOriginNotAllowed())
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Protected exige une session valide et COMPLÈTE (double authentification validée, ou ni activée ni
// obligatoire) et, si roles n'est pas vide, l'un de ces rôles. Sur les méthodes qui modifient des données,
// il exige aussi le jeton CSRF de la session.
// C'est le point d'entrée des autres domaines : router.Handle("POST /…", m.Protected(role)(h)). Par défaut,
// une session limitée n'ouvre donc rien : seules les routes qui la déclarent (ProtectedPendingMFA) lui sont
// accessibles.
func (m *Middleware) Protected(roles ...user.Role) httpx.Middleware {
	return m.protect(false, roles)
}

// ProtectedPendingMFA est Protected pour les routes qu'une session limitée doit pouvoir appeler : la
// double authentification elle-même (saisie du code, activation), l'état de la session et la déconnexion.
func (m *Middleware) ProtectedPendingMFA(roles ...user.Role) httpx.Middleware {
	return m.protect(true, roles)
}

func (m *Middleware) protect(allowPendingMFA bool, roles []user.Role) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, authenticated := PrincipalFrom(r.Context())
			if !authenticated {
				httpx.WriteError(w, httpx.Unauthenticated())
				return
			}
			if len(roles) > 0 && !slices.Contains(roles, principal.Role) {
				httpx.WriteError(w, httpx.Forbidden("Vous n'avez pas les droits nécessaires."))
				return
			}
			if !allowPendingMFA && !principal.MFAVerified {
				httpx.WriteError(w, httpx.NewError(http.StatusForbidden, CodeMFARequired, "La double authentification doit d'abord être validée."))
				return
			}
			if isUnsafeMethod(r.Method) {
				sessionToken, _ := m.cookies.Read(r)
				if !m.csrf.Valid(sessionToken, r.Header.Get(CSRFHeader)) {
					httpx.WriteError(w, httpx.NewError(http.StatusForbidden, CodeCSRFInvalid, "Jeton CSRF absent ou invalide."))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func errOriginNotAllowed() *httpx.Error {
	return httpx.NewError(http.StatusForbidden, CodeOriginNotAllowed, "Origine non autorisée.")
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
