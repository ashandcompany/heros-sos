package reputation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableReviews = "hero_reviews"
	tableReplies = "hero_review_replies"
)

type reviewRecord struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	InterventionID uuid.UUID
	HeroID         uuid.UUID
	CitizenID      *uuid.UUID
	Rating         int16
	Comment        *string
	Status         string
	ModeratedBy    *uuid.UUID
	ModeratedAt    *time.Time
	CreatedAt      time.Time
}

func (reviewRecord) TableName() string { return tableReviews }

type replyRecord struct {
	ReviewID    uuid.UUID `gorm:"primaryKey"`
	HeroID      uuid.UUID
	Content     string
	Status      string
	ModeratedBy *uuid.UUID
	ModeratedAt *time.Time
	CreatedAt   time.Time
	EditedAt    *time.Time
}

func (replyRecord) TableName() string { return tableReplies }

type reviewRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	InterventionID uuid.UUID  `gorm:"column:intervention_id"`
	HeroID         uuid.UUID  `gorm:"column:hero_id"`
	Rating         int16      `gorm:"column:rating"`
	Comment        *string    `gorm:"column:comment"`
	Status         string     `gorm:"column:status"`
	ModeratedAt    *time.Time `gorm:"column:moderated_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	HeroAlias      string     `gorm:"column:hero_alias"`
	ReplyID        *uuid.UUID `gorm:"column:reply_id"`
	ReplyHeroID    *uuid.UUID `gorm:"column:reply_hero_id"`
	ReplyContent   *string    `gorm:"column:reply_content"`
	ReplyStatus    *string    `gorm:"column:reply_status"`
	ReplyCreatedAt *time.Time `gorm:"column:reply_created_at"`
	ReplyEditedAt  *time.Time `gorm:"column:reply_edited_at"`
}

type citizenInterventionRow struct {
	InterventionID    uuid.UUID  `gorm:"column:intervention_id"`
	IncidentID        uuid.UUID  `gorm:"column:incident_id"`
	Outcome           string     `gorm:"column:outcome"`
	EndedAt           time.Time  `gorm:"column:ended_at"`
	HeroID            uuid.UUID  `gorm:"column:hero_id"`
	HeroAlias         string     `gorm:"column:hero_alias"`
	ReviewID          *uuid.UUID `gorm:"column:review_id"`
	Rating            *int16     `gorm:"column:rating"`
	Comment           *string    `gorm:"column:comment"`
	ReviewStatus      *string    `gorm:"column:review_status"`
	ReviewCreatedAt   *time.Time `gorm:"column:review_created_at"`
	ReviewModeratedAt *time.Time `gorm:"column:review_moderated_at"`
	ReplyID           *uuid.UUID `gorm:"column:reply_id"`
	ReplyHeroID       *uuid.UUID `gorm:"column:reply_hero_id"`
	ReplyContent      *string    `gorm:"column:reply_content"`
	ReplyStatus       *string    `gorm:"column:reply_status"`
	ReplyCreatedAt    *time.Time `gorm:"column:reply_created_at"`
	ReplyEditedAt     *time.Time `gorm:"column:reply_edited_at"`
}

type moderationReplyRow struct {
	ReviewID     uuid.UUID
	HeroID       uuid.UUID
	Content      string
	Status       string
	CreatedAt    time.Time
	EditedAt     *time.Time
	HeroAlias    string
	Rating       int16
	Comment      *string
	ReviewStatus string
}

type interventionState struct {
	Status string
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) ListCitizenInterventions(ctx context.Context, citizenID uuid.UUID, page Page) ([]RateableIntervention, int64, error) {
	query := r.citizenInterventionsQuery(ctx, citizenID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : comptage des interventions citoyennes : %w", err)
	}
	var rows []citizenInterventionRow
	if err := r.citizenInterventionsQuery(ctx, citizenID).
		Order("interventions.ended_at DESC, interventions.id, hero_profiles.alias").
		Limit(page.Limit).Offset(page.Offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : liste des interventions citoyennes : %w", err)
	}
	items := make([]RateableIntervention, len(rows))
	for i, row := range rows {
		item := RateableIntervention{
			InterventionID: row.InterventionID, IncidentID: row.IncidentID, Outcome: row.Outcome,
			EndedAt: row.EndedAt, HeroID: row.HeroID, HeroAlias: row.HeroAlias,
		}
		if row.ReviewID != nil && row.Rating != nil && row.ReviewStatus != nil && row.ReviewCreatedAt != nil {
			item.Review = &Review{
				ID: *row.ReviewID, InterventionID: row.InterventionID, HeroID: row.HeroID,
				Rating: *row.Rating, Comment: row.Comment, Status: *row.ReviewStatus,
				ModeratedAt: row.ReviewModeratedAt, CreatedAt: *row.ReviewCreatedAt,
			}
			if row.ReplyID != nil && row.ReplyContent != nil && row.ReplyStatus != nil && row.ReplyCreatedAt != nil {
				item.Review.Reply = &Reply{
					ReviewID: *row.ReplyID, Content: *row.ReplyContent, Status: *row.ReplyStatus,
					CreatedAt: *row.ReplyCreatedAt, EditedAt: row.ReplyEditedAt,
				}
			}
		}
		items[i] = item
	}
	return items, total, nil
}

func (r *gormRepository) citizenInterventionsQuery(ctx context.Context, citizenID uuid.UUID) *gorm.DB {
	return r.conn(ctx).Table("interventions").
		Joins("JOIN incidents ON incidents.id = interventions.incident_id").
		Joins("JOIN intervention_heroes ON intervention_heroes.intervention_id = interventions.id AND intervention_heroes.participation = 'confirmed'").
		Joins("JOIN hero_profiles ON hero_profiles.user_id = intervention_heroes.hero_id").
		Joins("LEFT JOIN hero_reviews ON hero_reviews.intervention_id = interventions.id AND hero_reviews.hero_id = intervention_heroes.hero_id AND hero_reviews.citizen_id = ?", citizenID).
		Joins("LEFT JOIN hero_review_replies ON hero_review_replies.review_id = hero_reviews.id AND hero_review_replies.status = ?", ReplyPublished).
		Where("incidents.reporter_id = ? AND intervention_heroes.hero_id <> ? AND interventions.status IN ?",
			citizenID, citizenID, []string{"succeeded", "failed"}).
		Select(`interventions.id AS intervention_id, incidents.id AS incident_id,
			interventions.status AS outcome, interventions.ended_at,
			intervention_heroes.hero_id, hero_profiles.alias AS hero_alias,
			hero_reviews.id AS review_id, hero_reviews.rating, hero_reviews.comment,
			hero_reviews.status AS review_status, hero_reviews.created_at AS review_created_at,
			hero_reviews.moderated_at AS review_moderated_at,
			hero_review_replies.review_id AS reply_id, hero_review_replies.hero_id AS reply_hero_id,
			hero_review_replies.content AS reply_content, hero_review_replies.status AS reply_status,
			hero_review_replies.created_at AS reply_created_at, hero_review_replies.edited_at AS reply_edited_at`)
}

func (r *gormRepository) CreateReview(ctx context.Context, review Review) error {
	var state interventionState
	err := r.conn(ctx).Table("interventions").Select("status").Where("id = ?", review.InterventionID).Take(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("reputation : lecture de l'intervention : %w", err)
	}
	if state.Status != "succeeded" && state.Status != "failed" {
		return ErrNotEligible
	}
	if review.CitizenID == nil || *review.CitizenID == review.HeroID {
		return ErrNotEligible
	}
	var eligible int64
	if err := r.conn(ctx).Table("interventions").
		Joins("JOIN incidents ON incidents.id = interventions.incident_id").
		Joins("JOIN intervention_heroes ON intervention_heroes.intervention_id = interventions.id").
		Where("interventions.id = ? AND incidents.reporter_id = ? AND intervention_heroes.hero_id = ? AND intervention_heroes.participation = 'confirmed'",
			review.InterventionID, *review.CitizenID, review.HeroID).Count(&eligible).Error; err != nil {
		return fmt.Errorf("reputation : vérification de l'éligibilité : %w", err)
	}
	if eligible == 0 {
		return ErrNotEligible
	}
	record := reviewRecord{
		ID: review.ID, InterventionID: review.InterventionID, HeroID: review.HeroID,
		CitizenID: review.CitizenID, Rating: review.Rating, Comment: review.Comment,
		Status: review.Status, CreatedAt: review.CreatedAt,
	}
	if err := r.conn(ctx).Create(&record).Error; err != nil {
		if database.IsUniqueViolation(err, "hero_reviews_intervention_hero_key") {
			return ErrAlreadyReviewed
		}
		return fmt.Errorf("reputation : création de l'avis : %w", err)
	}
	return nil
}

func (r *gormRepository) UpdateReview(ctx context.Context, reviewID, citizenID uuid.UUID, rating int16, comment *string, status string, now time.Time) error {
	var review reviewRecord
	if err := r.conn(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", reviewID).Take(&review).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("reputation : lecture de l'avis avant modification : %w", err)
	}
	if review.CitizenID == nil || *review.CitizenID != citizenID {
		return ErrNotEligible
	}
	result := r.conn(ctx).Model(&reviewRecord{}).Where("id = ?", reviewID).Updates(map[string]any{
		"rating": rating, "comment": comment, "status": status, "moderated_by": nil, "moderated_at": nil,
	})
	return updateStatusResult(result, "reputation : modification de l'avis")
}

func (r *gormRepository) ListHeroReviews(ctx context.Context, heroID uuid.UUID, page Page) (HeroReputation, error) {
	var reputation HeroReputation
	var summary struct {
		AverageRating float64
		RatingsCount  int64
	}
	if err := r.conn(ctx).Table(tableReviews).
		Select("COUNT(*) FILTER (WHERE status = ?) AS ratings_count, COALESCE(AVG(rating) FILTER (WHERE status = ?), 0) AS average_rating", ReviewPublished, ReviewPublished).
		Where("hero_id = ?", heroID).Scan(&summary).Error; err != nil {
		return HeroReputation{}, fmt.Errorf("reputation : calcul de la note du héros : %w", err)
	}
	reputation.AverageRating, reputation.RatingsCount = summary.AverageRating, summary.RatingsCount
	if err := r.conn(ctx).Model(&reviewRecord{}).Where("hero_id = ?", heroID).Count(&reputation.Total).Error; err != nil {
		return HeroReputation{}, fmt.Errorf("reputation : comptage des avis du héros : %w", err)
	}
	var rows []reviewRow
	if err := r.reviewQuery(ctx).Where("hero_reviews.hero_id = ?", heroID).
		Order("hero_reviews.created_at DESC, hero_reviews.id DESC").Limit(page.Limit).Offset(page.Offset).
		Scan(&rows).Error; err != nil {
		return HeroReputation{}, fmt.Errorf("reputation : lecture des avis du héros : %w", err)
	}
	reputation.Ratings = reviewItems(rows, false)
	reputation.Limit, reputation.Offset = page.Limit, page.Offset
	return reputation, nil
}

func (r *gormRepository) ListPublicReviews(ctx context.Context, heroID uuid.UUID, page Page) (PublicReputation, error) {
	var approved int64
	if err := r.conn(ctx).Table("hero_profiles").Joins("JOIN users ON users.id = hero_profiles.user_id").
		Where("hero_profiles.user_id = ? AND hero_profiles.verification_status = 'approved' AND users.status = 'active'", heroID).
		Count(&approved).Error; err != nil {
		return PublicReputation{}, fmt.Errorf("reputation : vérification du profil public : %w", err)
	}
	if approved == 0 {
		return PublicReputation{}, ErrNotFound
	}
	query := r.conn(ctx).Table(tableReviews).Where("hero_id = ? AND status = ?", heroID, ReviewPublished)
	var result PublicReputation
	if err := query.Count(&result.RatingsCount).Error; err != nil {
		return PublicReputation{}, fmt.Errorf("reputation : comptage des avis publics : %w", err)
	}
	if err := r.conn(ctx).Table(tableReviews).Select("COALESCE(AVG(rating), 0)").
		Where("hero_id = ? AND status = ?", heroID, ReviewPublished).Scan(&result.AverageRating).Error; err != nil {
		return PublicReputation{}, fmt.Errorf("reputation : moyenne publique : %w", err)
	}
	var rows []reviewRow
	if err := r.reviewQuery(ctx).Where("hero_reviews.hero_id = ? AND hero_reviews.status = ?", heroID, ReviewPublished).
		Order("hero_reviews.created_at DESC, hero_reviews.id DESC").Limit(page.Limit).Offset(page.Offset).
		Scan(&rows).Error; err != nil {
		return PublicReputation{}, fmt.Errorf("reputation : lecture des avis publics : %w", err)
	}
	result.Reviews = reviewItems(rows, true)
	return result, nil
}

func (r *gormRepository) ListModerationReviews(ctx context.Context, status string, page Page) ([]Review, int64, error) {
	query := r.reviewQuery(ctx)
	if status != "" {
		query = query.Where("hero_reviews.status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : comptage des avis à modérer : %w", err)
	}
	var rows []reviewRow
	query = r.reviewQuery(ctx)
	if status != "" {
		query = query.Where("hero_reviews.status = ?", status)
	}
	if err := query.Order("hero_reviews.created_at DESC, hero_reviews.id DESC").
		Limit(page.Limit).Offset(page.Offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : lecture des avis à modérer : %w", err)
	}
	return reviewItems(rows, false), total, nil
}

func (r *gormRepository) ListModerationReplies(ctx context.Context, status string, page Page) ([]ModerationReply, int64, error) {
	query := r.moderationRepliesQuery(ctx)
	if status != "" {
		query = query.Where("hero_review_replies.status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : comptage des réponses à modérer : %w", err)
	}
	query = r.moderationRepliesQuery(ctx)
	if status != "" {
		query = query.Where("hero_review_replies.status = ?", status)
	}
	var rows []moderationReplyRow
	if err := query.Order("hero_review_replies.created_at DESC, hero_review_replies.review_id").
		Limit(page.Limit).Offset(page.Offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("reputation : lecture des réponses à modérer : %w", err)
	}
	items := make([]ModerationReply, len(rows))
	for i, row := range rows {
		items[i] = ModerationReply{
			Reply: Reply{
				ReviewID: row.ReviewID, HeroID: row.HeroID, Content: row.Content,
				Status: row.Status, CreatedAt: row.CreatedAt, EditedAt: row.EditedAt,
			},
			HeroAlias: row.HeroAlias, Rating: row.Rating, Comment: row.Comment,
			ReviewStatus: row.ReviewStatus,
		}
	}
	return items, total, nil
}

func (r *gormRepository) moderationRepliesQuery(ctx context.Context) *gorm.DB {
	return r.conn(ctx).Table(tableReplies).
		Select(`hero_review_replies.review_id, hero_review_replies.hero_id, hero_review_replies.content,
			hero_review_replies.status, hero_review_replies.created_at, hero_review_replies.edited_at,
			hero_profiles.alias AS hero_alias, hero_reviews.rating, hero_reviews.comment,
			hero_reviews.status AS review_status`).
		Joins("JOIN hero_reviews ON hero_reviews.id = hero_review_replies.review_id").
		Joins("JOIN hero_profiles ON hero_profiles.user_id = hero_reviews.hero_id")
}

func (r *gormRepository) reviewQuery(ctx context.Context) *gorm.DB {
	return r.conn(ctx).Table(tableReviews).
		Select(`hero_reviews.id, hero_reviews.intervention_id, hero_reviews.hero_id, hero_reviews.rating,
			hero_reviews.comment, hero_reviews.status, hero_reviews.moderated_at, hero_reviews.created_at,
			hero_profiles.alias AS hero_alias, hero_review_replies.review_id AS reply_id,
			hero_review_replies.hero_id AS reply_hero_id, hero_review_replies.content AS reply_content,
			hero_review_replies.status AS reply_status, hero_review_replies.created_at AS reply_created_at,
			hero_review_replies.edited_at AS reply_edited_at`).
		Joins("JOIN hero_profiles ON hero_profiles.user_id = hero_reviews.hero_id").
		Joins("LEFT JOIN hero_review_replies ON hero_review_replies.review_id = hero_reviews.id")
}

func reviewItems(rows []reviewRow, public bool) []Review {
	items := make([]Review, len(rows))
	for i, row := range rows {
		item := Review{
			ID: row.ID, InterventionID: row.InterventionID, HeroID: row.HeroID, HeroAlias: row.HeroAlias,
			Rating: row.Rating, Comment: row.Comment, Status: row.Status,
			ModeratedAt: row.ModeratedAt, CreatedAt: row.CreatedAt,
		}
		if row.Status == ReviewHidden {
			item.Comment = nil
		}
		if row.ReplyID != nil && row.ReplyContent != nil && row.ReplyStatus != nil && row.ReplyCreatedAt != nil &&
			(!public || *row.ReplyStatus == ReplyPublished) {
			item.Reply = &Reply{
				ReviewID: *row.ReplyID, Content: *row.ReplyContent, Status: *row.ReplyStatus,
				CreatedAt: *row.ReplyCreatedAt, EditedAt: row.ReplyEditedAt,
			}
		}
		items[i] = item
	}
	return items
}

func (r *gormRepository) SaveReply(ctx context.Context, reviewID, heroID uuid.UUID, content string, now time.Time) error {
	var review reviewRecord
	if err := r.conn(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", reviewID).Take(&review).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("reputation : lecture de l'avis avant réponse : %w", err)
	}
	if review.HeroID != heroID {
		return ErrNotEligible
	}
	if review.Status != ReviewPublished || review.Comment == nil || *review.Comment == "" {
		return ErrReplyUnavailable
	}
	var reply replyRecord
	err := r.conn(ctx).Where("review_id = ?", reviewID).Take(&reply).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		reply = replyRecord{ReviewID: reviewID, HeroID: heroID, Content: content, Status: ReplyPublished, CreatedAt: now}
		if err := r.conn(ctx).Create(&reply).Error; err != nil {
			return fmt.Errorf("reputation : création de la réponse : %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("reputation : lecture de la réponse : %w", err)
	}
	if reply.HeroID != heroID {
		return ErrNotEligible
	}
	if err := r.conn(ctx).Model(&replyRecord{}).Where("review_id = ?", reviewID).Updates(map[string]any{
		"content": content, "status": ReplyPending, "moderated_by": nil, "moderated_at": nil, "edited_at": now,
	}).Error; err != nil {
		return fmt.Errorf("reputation : modification de la réponse : %w", err)
	}
	return nil
}

func (r *gormRepository) SetReviewStatus(ctx context.Context, reviewID, moderatorID uuid.UUID, status string, now time.Time) error {
	// Un modérateur déclarant peut noter les héros, mais son avis est validé par un autre modérateur.
	var review reviewRecord
	if err := r.conn(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", reviewID).Take(&review).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("reputation : lecture de l'avis avant modération : %w", err)
	}
	if review.CitizenID != nil && *review.CitizenID == moderatorID {
		return ErrOwnReview
	}
	result := r.conn(ctx).Model(&reviewRecord{}).Where("id = ?", reviewID).Updates(map[string]any{
		"status": status, "moderated_by": moderatorID, "moderated_at": now,
	})
	return updateStatusResult(result, "reputation : modération de l'avis")
}

func (r *gormRepository) SetReplyStatus(ctx context.Context, reviewID, moderatorID uuid.UUID, status string, now time.Time) error {
	result := r.conn(ctx).Model(&replyRecord{}).Where("review_id = ?", reviewID).Updates(map[string]any{
		"status": status, "moderated_by": moderatorID, "moderated_at": now,
	})
	return updateStatusResult(result, "reputation : modération de la réponse")
}

func updateStatusResult(result *gorm.DB, operation string) error {
	if result.Error != nil {
		return fmt.Errorf("%s : %w", operation, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
