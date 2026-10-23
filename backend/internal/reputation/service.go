package reputation

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const entityType = "hero_review"

const (
	actionCreated         = "hero_review.create"
	actionReviewUpdated   = "hero_review.update"
	actionReplySaved      = "hero_review.reply"
	actionReviewModerated = "hero_review.moderate"
	actionReplyModerated  = "hero_review.reply_moderate"
)

type ServiceDeps struct {
	Repo  Repository
	Audit audit.Recorder
	Tx    database.Transactor
	Clock clock.Clock
}

type Service struct {
	ServiceDeps
}

func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

func (s *Service) ListCitizenInterventions(ctx context.Context, citizenID uuid.UUID, page Page) ([]RateableIntervention, int64, error) {
	return s.Repo.ListCitizenInterventions(ctx, citizenID, page)
}

func (s *Service) Rate(ctx context.Context, citizenID, interventionID, heroID uuid.UUID, rating int16, comment string, ip netip.Addr) error {
	fields := make(map[string]string)
	if rating < 1 || rating > 5 {
		fields["rating"] = "invalid"
	}
	comment = strings.TrimSpace(comment)
	if len(comment) > MaxCommentLength {
		fields["comment"] = "too_long"
	}
	if len(fields) > 0 {
		return &validation.Error{Fields: fields}
	}
	var commentValue *string
	if comment != "" {
		commentValue = &comment
	}
	status := ReviewPublished
	if commentValue != nil {
		status = ReviewPending
	}
	review := Review{
		ID: ids.New(), InterventionID: interventionID, HeroID: heroID,
		CitizenID: &citizenID, Rating: rating, Comment: commentValue,
		Status: status, CreatedAt: s.Clock.Now(),
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.CreateReview(ctx, review); err != nil {
			return err
		}
		return s.record(ctx, citizenID, review.ID, actionCreated, ip, map[string]any{
			"interventionId": interventionID.String(), "heroId": heroID.String(), "status": status,
		})
	})
}

func (s *Service) UpdateReview(ctx context.Context, citizenID, reviewID uuid.UUID, rating int16, comment string, ip netip.Addr) error {
	fields := make(map[string]string)
	if rating < 1 || rating > 5 {
		fields["rating"] = "invalid"
	}
	comment = strings.TrimSpace(comment)
	if len(comment) > MaxCommentLength {
		fields["comment"] = "too_long"
	}
	if len(fields) > 0 {
		return &validation.Error{Fields: fields}
	}
	var commentValue *string
	status := ReviewPublished
	if comment != "" {
		commentValue = &comment
		status = ReviewPending
	}
	now := s.Clock.Now()
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.UpdateReview(ctx, reviewID, citizenID, rating, commentValue, status, now); err != nil {
			return err
		}
		return s.record(ctx, citizenID, reviewID, actionReviewUpdated, ip, map[string]any{
			"rating": rating, "hasComment": commentValue != nil, "status": status,
		})
	})
}

func (s *Service) ListHeroReviews(ctx context.Context, heroID uuid.UUID, page Page) (HeroReputation, error) {
	return s.Repo.ListHeroReviews(ctx, heroID, page)
}

func (s *Service) Reply(ctx context.Context, heroID, reviewID uuid.UUID, content string, ip netip.Addr) error {
	content = strings.TrimSpace(content)
	fields := make(map[string]string)
	if content == "" {
		fields["content"] = "required"
	} else if len(content) > MaxReplyLength {
		fields["content"] = "too_long"
	}
	if len(fields) > 0 {
		return &validation.Error{Fields: fields}
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SaveReply(ctx, reviewID, heroID, content, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, heroID, reviewID, actionReplySaved, ip, map[string]any{"status": ReplyPublished})
	})
}

func (s *Service) PublicReviews(ctx context.Context, heroID uuid.UUID, page Page) (PublicReputation, error) {
	return s.Repo.ListPublicReviews(ctx, heroID, page)
}

func (s *Service) ListModerationReviews(ctx context.Context, status string, page Page) ([]Review, int64, error) {
	if status != "" && status != ReviewPublished && status != ReviewPending && status != ReviewHidden {
		return nil, 0, &validation.Error{Fields: map[string]string{"status": "invalid"}}
	}
	return s.Repo.ListModerationReviews(ctx, status, page)
}

func (s *Service) ListModerationReplies(ctx context.Context, status string, page Page) ([]ModerationReply, int64, error) {
	if status != "" && status != ReplyPending && status != ReplyPublished && status != ReplyHidden {
		return nil, 0, &validation.Error{Fields: map[string]string{"status": "invalid"}}
	}
	return s.Repo.ListModerationReplies(ctx, status, page)
}

func (s *Service) ModerateReview(ctx context.Context, moderatorID, reviewID uuid.UUID, status string, ip netip.Addr) error {
	if status != ReviewPublished && status != ReviewHidden {
		return &validation.Error{Fields: map[string]string{"status": "invalid"}}
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetReviewStatus(ctx, reviewID, moderatorID, status, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, reviewID, actionReviewModerated, ip, map[string]any{"status": status})
	})
}

func (s *Service) ModerateReply(ctx context.Context, moderatorID, reviewID uuid.UUID, status string, ip netip.Addr) error {
	if status != ReplyPublished && status != ReplyHidden {
		return &validation.Error{Fields: map[string]string{"status": "invalid"}}
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetReplyStatus(ctx, reviewID, moderatorID, status, s.Clock.Now()); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, reviewID, actionReplyModerated, ip, map[string]any{"status": status})
	})
}

func (s *Service) record(ctx context.Context, actorID, reviewID uuid.UUID, action string, ip netip.Addr, details map[string]any) error {
	if s.Audit == nil {
		return errors.New("reputation : journal d'audit absent")
	}
	if err := s.Audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: entityType, EntityID: &reviewID, IP: ip, Details: details,
	}); err != nil {
		return fmt.Errorf("reputation : audit : %w", err)
	}
	return nil
}
