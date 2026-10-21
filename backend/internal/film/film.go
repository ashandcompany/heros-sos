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
