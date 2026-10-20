// Package reputation gère les avis des citoyens sur les héros et les réponses associées.
package reputation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	MaxCommentLength = 500
	MaxReplyLength   = 500

	ReviewPublished = "published"
	ReviewHidden    = "hidden"
	ReplyPending    = "pending"
	ReplyPublished  = "published"
	ReplyHidden     = "hidden"
)

var (
	ErrNotFound         = errors.New("reputation : introuvable")
	ErrNotEligible      = errors.New("reputation : avis non autorisé")
	ErrAlreadyReviewed  = errors.New("reputation : héros déjà noté pour cette intervention")
	ErrReplyUnavailable = errors.New("reputation : commentaire indisponible pour une réponse")
	ErrInvalidRating    = errors.New("reputation : note invalide")
	ErrInvalidStatus    = errors.New("reputation : statut invalide")
)

type Review struct {
	ID             uuid.UUID  `json:"id"`
	InterventionID uuid.UUID  `json:"interventionId"`
	HeroID         uuid.UUID  `json:"heroId"`
	HeroAlias      string     `json:"heroAlias,omitempty"`
	CitizenID      *uuid.UUID `json:"-"`
	Rating         int16      `json:"rating"`
	Comment        *string    `json:"comment"`
	Status         string     `json:"status"`
	ModeratedBy    *uuid.UUID `json:"-"`
	ModeratedAt    *time.Time `json:"moderatedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Reply          *Reply     `json:"reply,omitempty"`
}

type Reply struct {
	ReviewID    uuid.UUID  `json:"reviewId"`
	HeroID      uuid.UUID  `json:"-"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	ModeratedBy *uuid.UUID `json:"-"`
	ModeratedAt *time.Time `json:"moderatedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	EditedAt    *time.Time `json:"editedAt,omitempty"`
}

type RateableIntervention struct {
	InterventionID uuid.UUID `json:"interventionId"`
	IncidentID     uuid.UUID `json:"incidentId"`
	Outcome        string    `json:"outcome"`
	EndedAt        time.Time `json:"endedAt"`
	HeroID         uuid.UUID `json:"heroId"`
	HeroAlias      string    `json:"heroAlias"`
	Review         *Review   `json:"review"`
}

type PublicReputation struct {
	AverageRating float64  `json:"averageRating"`
	RatingsCount  int64    `json:"ratingsCount"`
	Reviews       []Review `json:"reviews"`
}

type HeroReputation struct {
	AverageRating float64  `json:"averageRating"`
	RatingsCount  int64    `json:"ratingsCount"`
	Ratings       []Review `json:"ratings"`
	Total         int64    `json:"total"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

type ModerationReply struct {
	Reply        Reply   `json:"reply"`
	HeroAlias    string  `json:"heroAlias"`
	Rating       int16   `json:"rating"`
	Comment      *string `json:"comment"`
	ReviewStatus string  `json:"reviewStatus"`
}

type Page struct {
	Limit  int
	Offset int
}

type Repository interface {
	ListCitizenInterventions(ctx context.Context, citizenID uuid.UUID, page Page) ([]RateableIntervention, int64, error)
	CreateReview(ctx context.Context, review Review) error
	ListHeroReviews(ctx context.Context, heroID uuid.UUID, page Page) (HeroReputation, error)
	ListPublicReviews(ctx context.Context, heroID uuid.UUID, page Page) (PublicReputation, error)
	ListModerationReviews(ctx context.Context, status string, page Page) ([]Review, int64, error)
	ListModerationReplies(ctx context.Context, status string, page Page) ([]ModerationReply, int64, error)
	SaveReply(ctx context.Context, reviewID, heroID uuid.UUID, content string, now time.Time) error
	SetReviewStatus(ctx context.Context, reviewID, moderatorID uuid.UUID, status string, now time.Time) error
	SetReplyStatus(ctx context.Context, reviewID, moderatorID uuid.UUID, status string, now time.Time) error
}
