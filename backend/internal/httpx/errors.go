// Package httpx regroupe ce qui est commun à tous les handlers HTTP : format des réponses et
// des erreurs, lecture stricte du JSON, adresse IP du client et middlewares transverses.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Codes d'erreur communs, stables et destinés au frontend.
const (
	CodeBadRequest           = "bad_request"
	CodeValidationFailed     = "validation_failed"
	CodeUnauthenticated      = "unauthenticated"
	CodeForbidden            = "forbidden"
	CodeNotFound             = "not_found"
	CodeConflict             = "conflict"
	CodeRateLimited          = "rate_limited"
	CodePayloadTooLarge      = "payload_too_large"
	CodeUnsupportedMediaType = "unsupported_media_type"
	CodeInternal             = "internal_error"
	CodeUnavailable          = "service_unavailable"
)

// Error est une erreur destinée à être renvoyée telle quelle au client.
type Error struct {
	Status  int
	Code    string
	Message string
	// Fields détaille les champs invalides (champ -> code), pour les erreurs de validation.
	Fields map[string]string
	// RetryAfter, s'il est renseigné, est annoncé dans l'en-tête Retry-After.
	RetryAfter time.Duration
}

func (e *Error) Error() string { return e.Code + " : " + e.Message }

// NewError construit une erreur HTTP.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Erreurs prédéfinies les plus courantes.
func BadRequest(message string) *Error {
	return NewError(http.StatusBadRequest, CodeBadRequest, message)
}
func Unauthenticated() *Error {
	return NewError(http.StatusUnauthorized, CodeUnauthenticated, "Authentification requise.")
}
func Forbidden(message string) *Error { return NewError(http.StatusForbidden, CodeForbidden, message) }
func NotFound(message string) *Error  { return NewError(http.StatusNotFound, CodeNotFound, message) }
func Conflict(code, message string) *Error {
	return NewError(http.StatusConflict, code, message)
}

// RateLimited répond 429 en indiquant au client quand réessayer.
func RateLimited(retryAfter time.Duration) *Error {
	err := NewError(http.StatusTooManyRequests, CodeRateLimited, "Trop de tentatives, réessayez plus tard.")
	err.RetryAfter = retryAfter
	return err
}

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteError écrit une *Error au format JSON de l'API.
func WriteError(w http.ResponseWriter, apiErr *Error) {
	if apiErr.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(apiErr.RetryAfter.Round(time.Second).Seconds())))
	}
	WriteJSON(w, apiErr.Status, errorBody{Error: errorPayload{
		Code: apiErr.Code, Message: apiErr.Message, Fields: apiErr.Fields,
	}})
}

// Fail écrit la réponse correspondant à err : une *Error ou une erreur de validation sont
// renvoyées au client ; toute autre erreur est journalisée puis masquée derrière un 500
// générique, pour ne jamais exposer de détail interne.
func Fail(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var apiErr *Error
	var validationErr *validation.Error
	ctxErr := r.Context().Err()
	switch {
	case errors.As(err, &apiErr):
		WriteError(w, apiErr)
	case errors.As(err, &validationErr):
		WriteError(w, &Error{
			Status:  http.StatusUnprocessableEntity,
			Code:    CodeValidationFailed,
			Message: "Certaines informations sont invalides.",
			Fields:  validationErr.Fields,
		})
	case ctxErr != nil && errors.Is(err, ctxErr):
		// Le client a abandonné la requête : rien à lui répondre.
		logger.DebugContext(r.Context(), "requête abandonnée par le client", "path", r.URL.Path)
	default:
		logger.ErrorContext(r.Context(), "erreur interne", "error", err, "method", r.Method, "path", r.URL.Path,
			"request_id", RequestIDFrom(r.Context()))
		WriteError(w, NewError(http.StatusInternalServerError, CodeInternal, "Une erreur interne est survenue."))
	}
}
