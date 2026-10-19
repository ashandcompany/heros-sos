package interventiontype

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

const entityType = "intervention_type"

// Service porte les règles du catalogue des types d'intervention.
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

// Input est la saisie d'un type d'intervention.
type Input struct {
	Label       string
	Description string
}

func validate(in Input) (Input, error) {
	var fields validation.Collector
	var code string
	in.Label, code = validation.RequiredText(in.Label, validation.MaxLabelLength)
	fields.Add("label", code)
	in.Description, code = validation.OptionalText(in.Description, validation.MaxShortDescriptionLength)
	fields.Add("description", code)
	return in, fields.Err()
}

// List retourne une page de types ; les utilisateurs ne voient que les types actifs.
func (s *Service) List(ctx context.Context, onlyActive bool, limit, offset int) ([]InterventionType, int64, error) {
	return s.repo.List(ctx, onlyActive, limit, offset)
}

// Create ajoute un type, actif dès sa création.
func (s *Service) Create(ctx context.Context, actorID uuid.UUID, in Input, ip netip.Addr) (*InterventionType, error) {
	in, err := validate(in)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	created := &InterventionType{
		ID: ids.New(), Label: in.Label, Description: in.Description, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, created); err != nil {
			return err
		}
		return s.record(ctx, actorID, created.ID, audit.ActionInterventionTypeCreated, ip)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Deactivate retire un type du choix des utilisateurs. C'est idempotent.
func (s *Service) Deactivate(ctx context.Context, actorID, id uuid.UUID, ip netip.Addr) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		changed, err := s.repo.Deactivate(ctx, id, s.clock.Now())
		if err != nil || !changed {
			return err
		}
		return s.record(ctx, actorID, id, audit.ActionInterventionTypeDeactivated, ip)
	})
}

func (s *Service) record(ctx context.Context, actorID, id uuid.UUID, action string, ip netip.Addr) error {
	return s.audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: entityType, EntityID: &id, IP: ip,
	})
}
