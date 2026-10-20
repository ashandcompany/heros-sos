package interventiontype

import (
	"context"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
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
	Code          string
	Label         string
	EndangersLife bool
}

func validate(in Input) (Input, error) {
	var fields validation.Collector
	in.Code = strings.TrimSpace(in.Code)
	in.Label = strings.TrimSpace(in.Label)
	if in.Code == "" {
		fields.Add("code", validation.CodeRequired)
	}
	if in.Label == "" {
		fields.Add("label", validation.CodeRequired)
	}
	return in, fields.Err()
}

// List retourne une page de types d'incident.
func (s *Service) List(ctx context.Context, limit, offset int) ([]InterventionType, int64, error) {
	return s.repo.List(ctx, limit, offset)
}

// Create ajoute un type, actif dès sa création.
func (s *Service) Create(ctx context.Context, actorID uuid.UUID, in Input, ip netip.Addr) (*InterventionType, error) {
	in, err := validate(in)
	if err != nil {
		return nil, err
	}
	created := &InterventionType{
		Code: in.Code, Label: in.Label, EndangersLife: in.EndangersLife,
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, created); err != nil {
			return err
		}
		return s.record(ctx, actorID, created, ip)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) record(ctx context.Context, actorID uuid.UUID, t *InterventionType, ip netip.Addr) error {
	return s.audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: audit.ActionInterventionTypeCreated, EntityType: entityType, IP: ip,
		Details: map[string]any{"typeId": t.ID, "code": t.Code},
	})
}
