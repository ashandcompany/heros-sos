package httpx

import (
	"net/http"
	"strconv"
)

// Bornes de pagination des listes.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Page est une fenêtre de résultats.
type Page struct {
	Limit  int
	Offset int
}

// ParsePage lit les paramètres limit et offset de la requête. Une valeur absente prend la valeur
// par défaut ; une valeur invalide ou hors bornes est refusée (jamais corrigée silencieusement).
func ParsePage(r *http.Request) (Page, error) {
	page := Page{Limit: DefaultPageSize}
	query := r.URL.Query()

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > MaxPageSize {
			return Page{}, &Error{
				Status: http.StatusUnprocessableEntity, Code: CodeValidationFailed,
				Message: "Certains paramètres sont invalides.", Fields: map[string]string{"limit": "invalid"},
			}
		}
		page.Limit = limit
	}
	if raw := query.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return Page{}, &Error{
				Status: http.StatusUnprocessableEntity, Code: CodeValidationFailed,
				Message: "Certains paramètres sont invalides.", Fields: map[string]string{"offset": "invalid"},
			}
		}
		page.Offset = offset
	}
	return page, nil
}

// PageResponse est l'enveloppe commune des listes paginées.
type PageResponse[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// NewPageResponse construit l'enveloppe d'une liste (jamais null : une liste vide reste []).
func NewPageResponse[T any](items []T, total int64, page Page) PageResponse[T] {
	if items == nil {
		items = []T{}
	}
	return PageResponse[T]{Items: items, Total: int(total), Limit: page.Limit, Offset: page.Offset}
}
