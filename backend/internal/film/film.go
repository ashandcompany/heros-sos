package film

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type Film struct {
	ID          uuid.UUID
	IMDBID      string
	VideoURL    string
	Title       string
	ReleaseYear *int16
	PosterURL   string
	Plot        string
	Raw         string
	FetchedAt   *time.Time
	CreatedAt   time.Time
}

type Title struct {
	ID          string
	Title       string
	ReleaseYear *int16
	PosterURL   string
	Plot        string
}

type HeroFilm struct {
	Film
	HeroID     uuid.UUID
	HeroAlias  string
	Status     string
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	AddedAt    time.Time
}

type VillainFilm struct {
	Film
	VillainID    uuid.UUID
	VillainAlias string
	CreatedBy    *uuid.UUID
	AddedAt      time.Time
}

// HeroFilmFilter restreint une liste de rattachements (héros, film). Status vide : tous les
// statuts pour un héros donné, seulement les rattachements approuvés sinon ; "all" : tous.
type HeroFilmFilter struct {
	HeroID *uuid.UUID
	Status string
	// Title : extrait du titre, sans tenir compte de la casse.
	Title string
}

// PublicFilmFilter restreint la vidéothèque publique.
type PublicFilmFilter struct {
	// HeroID : films rattachés (et approuvés) à ce héros, sans masquer leurs autres héros.
	HeroID *uuid.UUID
	Title  string
}

type HeroRef struct {
	ID    uuid.UUID `json:"id"`
	Alias string    `json:"alias"`
}

// PublicFilm est un film de la vidéothèque publique, une seule fois quel que soit le nombre de
// héros auxquels il est rattaché. AddedAt est la date de son rattachement le plus récent.
type PublicFilm struct {
	Film
	Heroes  []HeroRef
	AddedAt time.Time
}

// FilmHero est un super-héros ayant au moins un film publié.
type FilmHero struct {
	HeroRef
	FilmCount int64
}
