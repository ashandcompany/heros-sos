package villain

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/interventiontype"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const entityType = "villain"

var ErrNotAllowed = errors.New("villain : modification non autorisée")

type TypeFinder interface {
	Find(ctx context.Context, id int16) (*interventiontype.InterventionType, error)
}

// Service porte les règles des vilains.
type Service struct {
	repo  Repository
	audit audit.Recorder
	tx    database.Transactor
	clock clock.Clock
	types TypeFinder
}

// NewService crée le service.
func NewService(repo Repository, recorder audit.Recorder, tx database.Transactor, clk clock.Clock, types TypeFinder) *Service {
	return &Service{repo: repo, audit: recorder, tx: tx, clock: clk, types: types}
}

// CreateInput est la saisie d'un nouveau vilain.
type CreateInput struct {
	Alias       string
	LastName    string
	FirstName   string
	MainSector  string
	Description string
	TypeIDs     []int16
}

type UpdateInput struct {
	Alias       string
	LastName    *string
	FirstName   *string
	MainSector  *string
	Description *string
	TypeIDs     []int16
	ThreatLevel *string
	Status      *string
}

// Create enregistre un vilain. Un alias déjà pris retourne ErrAliasTaken.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput, ip netip.Addr) (*Villain, error) {
	var fields validation.Collector
	alias := strings.TrimSpace(in.Alias)
	if alias == "" {
		fields.Add("alias", validation.CodeRequired)
	}
	description := strings.TrimSpace(in.Description)
	lastName := strings.TrimSpace(in.LastName)
	firstName := strings.TrimSpace(in.FirstName)
	mainSector := strings.TrimSpace(in.MainSector)
	typeIDs, err := s.validateTypeIDs(ctx, in.TypeIDs, &fields)
	if err != nil {
		return nil, err
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	created := &Villain{
		ID: ids.New(), Alias: alias, LastName: lastName, FirstName: firstName,
		MainSector: mainSector, Description: description, CreatedBy: &userID, CreatedAt: now,
		UpdatedAt: now, TypeIDs: typeIDs, ThreatLevel: defaultThreatLevel, Status: defaultStatus,
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
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

func (s *Service) Update(ctx context.Context, actorID uuid.UUID, role user.Role, id uuid.UUID, in UpdateInput, ip netip.Addr) (*Villain, error) {
	current, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if role != user.RoleModerator && (current.CreatedBy == nil || *current.CreatedBy != actorID) {
		return nil, ErrNotAllowed
	}
	if role != user.RoleModerator && (in.ThreatLevel != nil || in.Status != nil) {
		return nil, ErrNotAllowed
	}
	var fields validation.Collector
	alias := strings.TrimSpace(in.Alias)
	if alias == "" {
		fields.Add("alias", validation.CodeRequired)
	}
	typeIDs, err := s.validateTypeIDs(ctx, in.TypeIDs, &fields)
	if err != nil {
		return nil, err
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	updated := &Villain{
		ID: id, Alias: alias, LastName: current.LastName, FirstName: current.FirstName,
		MainSector: current.MainSector, Description: current.Description,
		ThreatLevel: current.ThreatLevel, Status: current.Status,
		CreatedBy: current.CreatedBy, CreatedAt: current.CreatedAt, UpdatedAt: s.clock.Now(), TypeIDs: typeIDs,
	}
	if in.LastName != nil {
		updated.LastName = strings.TrimSpace(*in.LastName)
	}
	if in.FirstName != nil {
		updated.FirstName = strings.TrimSpace(*in.FirstName)
	}
	if in.MainSector != nil {
		updated.MainSector = strings.TrimSpace(*in.MainSector)
	}
	if in.Description != nil {
		updated.Description = strings.TrimSpace(*in.Description)
	}
	if in.ThreatLevel != nil {
		level := ThreatLevel(*in.ThreatLevel)
		if !level.Valid() {
			fields.Add("threatLevel", validation.CodeInvalid)
		}
		updated.ThreatLevel = level
	}
	if in.Status != nil {
		status := Status(*in.Status)
		if !status.Valid() {
			fields.Add("status", validation.CodeInvalid)
		}
		updated.Status = status
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Update(ctx, updated); err != nil {
			return err
		}
		return s.record(ctx, actorID, id, audit.ActionVillainUpdated, ip)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) validateTypeIDs(ctx context.Context, raw []int16, fields *validation.Collector) ([]int16, error) {
	seen := make(map[int16]struct{}, len(raw))
	valid := make([]int16, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			fields.Add("typeIds", validation.CodeInvalid)
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		if _, err := s.types.Find(ctx, id); err != nil {
			if errors.Is(err, interventiontype.ErrNotFound) {
				fields.Add("typeIds", validation.CodeInvalid)
				continue
			}
			return nil, err
		}
		valid = append(valid, id)
	}
	return valid, nil
}

// Get retourne un vilain, ou ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Villain, error) {
	return s.repo.Find(ctx, id)
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
