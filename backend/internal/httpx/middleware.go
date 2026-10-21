package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

const (
	requestIDHeader = "X-Request-Id"
	// hstsValue demande aux navigateurs de n'utiliser que HTTPS pendant un an.
	hstsValue = "max-age=31536000"

	minRequestIDLength = 8
	maxRequestIDLength = 64
)

var validRequestID = regexp.MustCompile(fmt.Sprintf(`^[A-Za-z0-9_-]{%d,%d}$`, minRequestIDLength, maxRequestIDLength))

type requestIDKey struct{}

// Middleware transforme un handler en un autre.
type Middleware func(http.Handler) http.Handler

// Chain applique les middlewares à h. Le premier de la liste est le plus externe : il voit la
// requête en premier et la réponse en dernier.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RequestID attribue un identifiant à chaque requête, renvoyé dans l'en-tête X-Request-Id
// et présent dans les logs. Un identifiant fourni par le client n'est conservé que s'il a un
// format sûr (il finit dans les logs).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = ids.New().String()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom retourne l'identifiant de la requête (vide hors RequestID).
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// SecurityHeaders ajoute les en-têtes de sécurité d'une API JSON. Les réponses ne sont jamais
// mises en cache (elles dépendent de la session). hsts n'est activé qu'en production.
func SecurityHeaders(hsts bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("Cache-Control", "no-store")
			header.Set("Referrer-Policy", "no-referrer")
			header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			if hsts {
				header.Set("Strict-Transport-Security", hstsValue)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter mémorise le code de statut et le nombre d'octets écrits.
type statusWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status, w.wroteHeader = status, true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(data)
	w.bytes += written
	return written, err
}

// Unwrap permet à http.ResponseController d'atteindre le ResponseWriter d'origine.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func wrap(w http.ResponseWriter) *statusWriter {
	if existing, ok := w.(*statusWriter); ok {
		return existing
	}
	return &statusWriter{ResponseWriter: w, status: http.StatusOK}
}

// AccessLog journalise chaque requête : méthode, chemin (jamais la query string, qui peut
// contenir un jeton), statut, durée et adresse du client.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := wrap(w)
			next.ServeHTTP(recorder, r)

			level := slog.LevelInfo
			switch {
			case recorder.status >= http.StatusInternalServerError:
				level = slog.LevelError
			case recorder.status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}
			logger.Log(r.Context(), level, "requête",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"bytes", recorder.bytes,
				"ip", ClientIPFrom(r.Context()).String(),
				"request_id", RequestIDFrom(r.Context()),
			)
		})
	}
}

// Recover transforme une panique en réponse 500 et la journalise avec sa pile d'appels.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := wrap(w)
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler { //nolint:errorlint // valeur sentinelle de la bibliothèque standard
					panic(recovered)
				}
				logger.ErrorContext(r.Context(), "panique dans un handler",
					"panic", recovered, "stack", string(debug.Stack()),
					"method", r.Method, "path", r.URL.Path, "request_id", RequestIDFrom(r.Context()))
				if !recorder.wroteHeader {
					WriteError(recorder, NewError(http.StatusInternalServerError, CodeInternal, "Une erreur interne est survenue."))
				}
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}
