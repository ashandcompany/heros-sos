package incident

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/interventiontype"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

const (
	entityType = "incident"
	// futureTolerance absorbe le décalage d'horloge entre le navigateur et le serveur.
	futureTolerance = time.Minute
)

// TypeFinder lit le catalogue des types d'intervention.
type TypeFinder interface {
	Find(ctx context.Context, id uuid.UUID) (*interventiontype.InterventionType, error)
}

// VillainFinder lit les vilains.
type VillainFinder interface {
	Find(ctx context.Context, id uuid.UUID) (*villain.Villain, error)
}

// Notifier prévient les modérateurs et les héros d'un nouvel incident (implémenté par mail.Service).
type Notifier interface {
	SendIncidentReported(ctx context.Context, to, title, interventionType string) error
}

// ServiceDeps regroupe les dépendances du service.
type ServiceDeps struct {
	Repo     Repository
	Types    TypeFinder
	Villains VillainFinder
	Users    user.Repository
	Notifier Notifier
	Audit    audit.Recorder
	Tx       database.Transactor
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Service porte les règles de déclaration des incidents.
type Service struct {
	ServiceDeps
}

// NewService crée le service.
func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

// CreateInput est la saisie d'une déclaration (valeurs brutes, non validées).
type CreateInput struct {
	ReporterRole       string
	InterventionTypeID string
	// VillainID est vide quand le déclarant ne sait pas quel vilain est impliqué.
	VillainID   string
	Title       string
	Description string
	Address     string
	City        string
	// OccurredAt est une date RFC 3339, vide si l'incident est en cours.
	OccurredAt string
}

// Create enregistre l'incident d'un utilisateur, puis prévient les modérateurs et les héros. Toutes
// les erreurs de saisie sont retournées ensemble (*validation.Error). Un échec d'envoi des mails
// n'annule pas la déclaration : il est journalisé.
func (s *Service) Create(ctx context.Context, reporterID uuid.UUID, in CreateInput, ip netip.Addr) (*Incident, error) {
	now := s.Clock.Now()
	created, err := s.validate(ctx, in, now)
	if err != nil {
		return nil, err
	}
	created.ID, created.ReporterID, created.Status = ids.New(), reporterID, StatusReported
	created.CreatedAt, created.UpdatedAt = now, now

	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.Create(ctx, created); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &reporterID, Action: audit.ActionIncidentDeclared, EntityType: entityType,
			EntityID: &created.ID, IP: ip,
		})
	})
	if err != nil {
		return nil, err
	}
	s.notify(ctx, created)
	return created, nil
}

func (s *Service) validate(ctx context.Context, in CreateInput, now time.Time) (*Incident, error) {
	var fields validation.Collector
	created := &Incident{}
	var code string

	created.ReporterRole = ReporterRole(strings.TrimSpace(in.ReporterRole))
	if !created.ReporterRole.Valid() {
		fields.Add("reporterRole", validation.CodeInvalid)
	}
	created.Title, code = validation.BoundedText(in.Title, validation.MinIncidentTitleLength, validation.MaxIncidentTitleLength)
	fields.Add("title", code)
	created.Description, code = validation.BoundedText(in.Description, validation.MinIncidentDescription, validation.MaxIncidentDescription)
	fields.Add("description", code)
	created.Address, code = validation.RequiredText(in.Address, validation.MaxStreetLength)
	fields.Add("location.address", code)
	created.City, code = validation.RequiredText(in.City, validation.MaxCityLength)
	fields.Add("location.city", code)

	if raw := strings.TrimSpace(in.OccurredAt); raw != "" {
		occurredAt, err := time.Parse(time.RFC3339, raw)
		if err != nil || occurredAt.After(now.Add(futureTolerance)) {
			fields.Add("occurredAt", validation.CodeInvalid)
		} else {
			created.OccurredAt = &occurredAt
		}
	}

	if err := s.resolveType(ctx, in.InterventionTypeID, created, &fields); err != nil {
		return nil, err
	}
	if err := s.resolveVillain(ctx, in.VillainID, created, &fields); err != nil {
		return nil, err
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	return created, nil
}

// resolveType exige un type existant et actif.
func (s *Service) resolveType(ctx context.Context, raw string, created *Incident, fields *validation.Collector) error {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		fields.Add("interventionTypeId", validation.CodeInvalid)
		return nil
	}
	found, err := s.Types.Find(ctx, id)
	switch {
	case errors.Is(err, interventiontype.ErrNotFound), err == nil && !found.Active:
		fields.Add("interventionTypeId", validation.CodeInvalid)
		return nil
	case err != nil:
		return err
	}
	created.InterventionTypeID, created.InterventionType = found.ID, found.Label
	return nil
}

func (s *Service) resolveVillain(ctx context.Context, raw string, created *Incident, fields *validation.Collector) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		fields.Add("villainId", validation.CodeInvalid)
		return nil
	}
	found, err := s.Villains.Find(ctx, id)
	if err != nil {
		if errors.Is(err, villain.ErrNotFound) {
			fields.Add("villainId", validation.CodeInvalid)
			return nil
		}
		return err
	}
	created.VillainID, created.VillainAlias = &found.ID, found.Alias
	return nil
}

// notify prévient les modérateurs et les héros actifs, sauf le déclarant. Le lieu n'est jamais
// envoyé par mail.
func (s *Service) notify(ctx context.Context, created *Incident) {
	reporter, err := s.Users.FindByID(ctx, created.ReporterID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "notification d'incident : déclarant introuvable", "incident_id", created.ID, "error", err)
		return
	}
	recipients, err := s.Users.ActiveEmailsByRole(ctx, user.RoleModerator, user.RoleHero)
	if err != nil {
		s.Logger.ErrorContext(ctx, "notification d'incident : destinataires illisibles", "incident_id", created.ID, "error", err)
		return
	}
	for _, to := range recipients {
		if to == reporter.Email {
			continue
		}
		if err := s.Notifier.SendIncidentReported(ctx, to, created.Title, created.InterventionType); err != nil {
			s.Logger.ErrorContext(ctx, "notification d'incident impossible", "incident_id", created.ID, "error", err)
		}
	}
}
