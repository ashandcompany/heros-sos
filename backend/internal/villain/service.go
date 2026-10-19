package villain

import (
	"context"
	"net/netip"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const entityType = "villain"

// Service porte les règles des vilains.
type Service struct {
	repo  Repository
	audit audit.Recorder
	tx    database.Transactor
	clock clock.Clock
}

// NewService crée le service.
func NewService(repo Repository, recorder audit.Recorder, tx database.Transactor, clk clock.Clock) *Service {
	return &Service{repo: repo, audit: recorder, tx: tx, clock: clk}
}

// CreateInput est la saisie d'un nouveau vilain.
type CreateInput struct {
	Alias       string
	Description string
}

// Create enregistre un vilain. Un pseudo déjà pris (sans tenir compte de la casse) retourne
// ErrAliasTaken.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput, ip netip.Addr) (*Villain, error) {
	var fields validation.Collector
	alias, code := validation.Alias(in.Alias)
	fields.Add("alias", code)
	description, code := validation.OptionalText(in.Description, validation.MaxShortDescriptionLength)
	fields.Add("description", code)
	if err := fields.Err(); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	created := &Villain{
		ID: ids.New(), Alias: alias, Description: description, CreatedBy: &userID, CreatedAt: now,
	}
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, created); err != nil {
			return err
		}
		return s.record(ctx, userID, created.ID, audit.ActionVillainCreated, ip)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// List retourne une page de vilains, éventuellement filtrée par début de pseudo.
func (s *Service) List(ctx context.Context, aliasPrefix string, limit, offset int) ([]Villain, int64, error) {
	return s.repo.List(ctx, aliasPrefix, limit, offset)
}

func (s *Service) record(ctx context.Context, actorID, id uuid.UUID, action string, ip netip.Addr) error {
	return s.audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: entityType, EntityID: &id, IP: ip,
	})
}
