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
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/interventiontype"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

const (
	entityType = "incident"
	// titleMaxLength borne le titre affiché dans les listes (même limite que le frontend).
	titleMaxLength = 80
	// futureTolerance absorbe le décalage d'horloge entre le navigateur et le serveur.
	futureTolerance = time.Minute
)

// TypeFinder lit le catalogue des types d'intervention.
type TypeFinder interface {
	Find(ctx context.Context, id int16) (*interventiontype.InterventionType, error)
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
	Events   events.Recorder
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
	// ID est l'identifiant choisi par le client (uuid.Nil : le serveur le choisit). Un appareil hors ligne le
	// fixe à la déclaration pour reconnaître son incident une fois synchronisé.
	ID                uuid.UUID
	TypeID            int16
	Title             string
	Description       string
	CitizensRescued   int16
	Street            string
	PostalCode        string
	City              string
	Country           string
	AddressComplement string
	Latitude          *float64
	Longitude         *float64
	OccurredAt        string
	VillainIDs        []string
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
	created.ID, created.ReporterID, created.Status = in.ID, reporterID, StatusReported
	if created.ID == uuid.Nil {
		created.ID = ids.New()
	}
	created.Priority = PriorityNormal
	created.CreatedAt, created.UpdatedAt = now, now

	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		// En premier : voir events.Recorder.Lock.
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		if err := s.Repo.Create(ctx, created); err != nil {
			return err
		}
		if _, err := s.Events.Append(ctx, declaredEvent(created, reporterID)); err != nil {
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

// declaredEvent est l'événement d'une déclaration : des identifiants et des états, ni description ni adresse.
func declaredEvent(created *Incident, reporterID uuid.UUID) events.Event {
	villainIDs := make([]string, len(created.VillainIDs))
	for i, id := range created.VillainIDs {
		villainIDs[i] = id.String()
	}
	return events.Event{
		Type: events.IncidentDeclared, AggregateType: events.AggregateIncident, AggregateID: created.ID, IncidentID: created.ID,
		ActorID: &reporterID,
		Payload: map[string]any{
			"typeId": created.TypeID, "priority": string(created.Priority), "city": created.City, "villainIds": villainIDs,
		},
	}
}

func (s *Service) validate(ctx context.Context, in CreateInput, now time.Time) (*Incident, error) {
	var fields validation.Collector
	created := &Incident{}

	var code string
	if created.Title, code = validation.RequiredText(in.Title, titleMaxLength); code != "" {
		fields.Add("title", code)
	}

	created.Description = strings.TrimSpace(in.Description)
	if created.Description == "" {
		fields.Add("description", validation.CodeRequired)
	}
	created.Street = strings.TrimSpace(in.Street)
	if created.Street == "" {
		fields.Add("location.address", validation.CodeRequired)
	}
	created.City = strings.TrimSpace(in.City)
	if created.City == "" {
		fields.Add("location.city", validation.CodeRequired)
	}
	created.PostalCode = strings.TrimSpace(in.PostalCode)
	created.Country = strings.TrimSpace(in.Country)
	created.AddressComplement = strings.TrimSpace(in.AddressComplement)
	created.CitizensRescued = in.CitizensRescued
	if in.CitizensRescued < 0 {
		fields.Add("citizensRescued", validation.CodeInvalid)
	}
	created.Latitude, created.Longitude = in.Latitude, in.Longitude
	created.Priority = PriorityNormal

	if raw := strings.TrimSpace(in.OccurredAt); raw != "" {
		occurredAt, err := time.Parse(time.RFC3339, raw)
		if err != nil || occurredAt.After(now.Add(futureTolerance)) {
			fields.Add("occurredAt", validation.CodeInvalid)
		} else {
			created.OccurredAt = &occurredAt
		}
	}

	if err := s.resolveType(ctx, in.TypeID, created, &fields); err != nil {
		return nil, err
	}
	if err := s.resolveVillains(ctx, in.VillainIDs, created, &fields); err != nil {
		return nil, err
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	return created, nil
}

// resolveType exige un type d'incident existant.
func (s *Service) resolveType(ctx context.Context, id int16, created *Incident, fields *validation.Collector) error {
	if id <= 0 {
		fields.Add("typeId", validation.CodeInvalid)
		return nil
	}
	found, err := s.Types.Find(ctx, id)
	if errors.Is(err, interventiontype.ErrNotFound) {
		fields.Add("typeId", validation.CodeInvalid)
		return nil
	}
	if err != nil {
		return err
	}
	created.TypeID, created.TypeCode, created.TypeLabel = found.ID, found.Code, found.Label
	return nil
}

func (s *Service) resolveVillains(ctx context.Context, rawIDs []string, created *Incident, fields *validation.Collector) error {
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			fields.Add("villainIds", validation.CodeInvalid)
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		found, err := s.Villains.Find(ctx, id)
		if err != nil {
			if errors.Is(err, villain.ErrNotFound) {
				fields.Add("villainIds", validation.CodeInvalid)
				continue
			}
			return err
		}
		created.VillainIDs = append(created.VillainIDs, id)
		created.Villains = append(created.Villains, VillainReference{ID: id, Alias: found.Alias})
	}
	return nil
}

// List retourne une page des incidents (résolus ou non) correspondant au filtre, les plus récents d'abord.
func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) (ListResult, error) {
	return s.Repo.List(ctx, filter, limit, offset)
}

// Get retourne le détail d'un incident.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Incident, error) {
	return s.Repo.Find(ctx, id)
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
		if err := s.Notifier.SendIncidentReported(ctx, to, created.Title, created.TypeLabel); err != nil {
			s.Logger.ErrorContext(ctx, "notification d'incident impossible", "incident_id", created.ID, "error", err)
		}
	}
}
