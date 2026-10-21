// Package blog gère les articles du blog : les modérateurs les rédigent et choisissent de les
// publier, tout le monde lit ceux qui sont publiés.
package blog

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status est l'état de publication d'un article.
type Status string

const (
	StatusDraft     Status = "brouillon"
	StatusPublished Status = "publie"
)

// Valid indique si le statut est connu.
func (s Status) Valid() bool { return s == StatusDraft || s == StatusPublished }

// Limites de saisie, identiques à celles de l'éditeur du back-office
// (frontend/src/lib/validation/admin.ts).
const (
	MinTitleLength       = 5
	MaxTitleLength       = 120
	MaxExcerptLength     = 300
	MinContentTextLength = 20
	MaxContentBytes      = 200_000
)

var (
	// ErrNotFound indique qu'aucun article ne correspond (ou qu'il n'est pas public).
	ErrNotFound = errors.New("blog : article introuvable")
	// ErrSlugTaken indique qu'un autre article porte déjà ce slug.
	ErrSlugTaken = errors.New("blog : slug déjà utilisé")
)

// Article est un article du blog. Content est du HTML déjà assaini.
type Article struct {
	ID          uuid.UUID
	Title       string
	Slug        string
	Excerpt     string
	Content     string
	Status      Status
	AuthorID    *uuid.UUID
	AuthorName  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}
