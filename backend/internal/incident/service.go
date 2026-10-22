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
// nearbyRadiusKm vaut le rayon d'alerte pour un héros prévenu parce qu'il est proche, 0 sinon.
type Notifier interface {
	SendIncidentReported(ctx context.Context, to, title, interventionType string, nearbyRadiusKm int) error
}

// HeroLocator trouve les héros à proximité d'un point (implémenté par herolocation.Service).
type HeroLocator interface {
	EmailsNear(ctx context.Context, latitude, longitude float64) ([]string, error)
	// AlertRadiusKm est le rayon dans lequel EmailsNear cherche.
	AlertRadiusKm() int
}

// ServiceDeps regroupe les dépendances du service.
type ServiceDeps struct {
	Repo     Repository
	Types    TypeFinder
	Villains VillainFinder
	Users    user.Repository
	Heroes   HeroLocator
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
	TypeID            int16
	Title             string
	Description       string
	ReporterRole      *string
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
func (s *Service) Create(ctx context.Context, reporterID uuid.UUID, reporterAccountRole user.Role, in CreateInput, ip netip.Addr) (*Incident, error) {
	now := s.Clock.Now()
	created, err := s.validate(ctx, in, now)
	if err != nil {
		return nil, err
	}
	created.ID, created.ReporterID, created.Status = ids.New(), reporterID, StatusReported
	created.Priority = PriorityNormal
	created.Moderation = ModerationPending
	if reporterAccountRole == user.RoleModerator {
		created.Moderation = ModerationApproved
	}
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
	if in.ReporterRole != nil {
		role := ReporterRole(strings.TrimSpace(*in.ReporterRole))
		if role != ReporterWitness && role != ReporterVictim {
			fields.Add("reporterRole", validation.CodeInvalid)
		} else {
			created.ReporterRole = &role
		}
	}

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

// List retourne les incidents (résolus ou non) correspondant au filtre, les plus récents d'abord.
func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Incident, int64, error) {
	return s.Repo.List(ctx, filter, limit, offset)
}

// Moderate modifie la visibilité publique d'un incident et journalise la décision.
func (s *Service) Moderate(ctx context.Context, actorID, id uuid.UUID, status ModerationStatus, ip netip.Addr) error {
	if status != ModerationApproved && status != ModerationRejected {
		return &validation.Error{Fields: map[string]string{"decision": validation.CodeInvalid}}
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetModeration(ctx, id, status, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &actorID, Action: audit.ActionIncidentModerated, EntityType: entityType, EntityID: &id,
			Details: map[string]any{"moderation": status}, IP: ip,
		})
	})
}

// Update remplace les champs modifiables et synchronise l'intervention dans la même transaction.
func (s *Service) Update(ctx context.Context, actorID, id uuid.UUID, in UpdateInput, ip netip.Addr) error {
	updated, err := s.resolveUpdate(ctx, id, in)
	if err != nil {
		return err
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.Update(ctx, updated, in.HeroID, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &actorID, Action: audit.ActionIncidentUpdated, EntityType: entityType, EntityID: &id,
			Details: map[string]any{"status": updated.Status}, IP: ip,
		})
	})
}

// SetPriority modifie la priorité d'un incident encore disponible et journalise le changement.
func (s *Service) SetPriority(ctx context.Context, heroID, id uuid.UUID, priority Priority, ip netip.Addr) error {
	if priority != PriorityCritical && priority != PriorityHigh && priority != PriorityNormal && priority != PriorityLow {
		return &validation.Error{Fields: map[string]string{"priority": validation.CodeInvalid}}
	}
	public, err := s.Repo.PublicHeroExists(ctx, heroID)
	if err != nil {
		return err
	}
	if !public {
		return ErrHeroNotFound
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPriority(ctx, id, priority, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &heroID, Action: audit.ActionIncidentPrioritySet, EntityType: entityType, EntityID: &id,
			Details: map[string]any{"priority": priority}, IP: ip,
		})
	})
}

// Get retourne le détail d'un incident.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Incident, error) {
	return s.Repo.Find(ctx, id)
}

// UpdateInput porte les champs remplacés par la modération.
type UpdateInput struct {
	TypeID      int16
	Title       string
	Description string
	Status      Status
	Street      string
	City        string
	Latitude    *float64
	Longitude   *float64
	VillainID   *string
	HeroID      *uuid.UUID
}

func (s *Service) resolveUpdate(ctx context.Context, id uuid.UUID, in UpdateInput) (*Incident, error) {
	var fields validation.Collector
	current, err := s.Repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *current
	var titleCode string
	if updated.Title, titleCode = validation.RequiredText(in.Title, titleMaxLength); titleCode != "" {
		fields.Add("title", titleCode)
	}
	updated.Description = strings.TrimSpace(in.Description)
	if updated.Description == "" {
		fields.Add("description", validation.CodeRequired)
	}
	updated.Status = in.Status
	if _, ok := ParseStatus(string(in.Status)); !ok {
		fields.Add("status", validation.CodeInvalid)
	}
	updated.Street = strings.TrimSpace(in.Street)
	if updated.Street == "" {
		fields.Add("location.address", validation.CodeRequired)
	}
	updated.City = strings.TrimSpace(in.City)
	if updated.City == "" {
		fields.Add("location.city", validation.CodeRequired)
	}
	updated.Latitude, updated.Longitude = in.Latitude, in.Longitude
	if (in.Latitude == nil) != (in.Longitude == nil) {
		fields.Add("location", validation.CodeInvalid)
	}
	if in.Latitude != nil && (*in.Latitude < -90 || *in.Latitude > 90 || *in.Longitude < -180 || *in.Longitude > 180) {
		fields.Add("location", validation.CodeInvalid)
	}

	typeInfo, err := s.Types.Find(ctx, in.TypeID)
	if errors.Is(err, interventiontype.ErrNotFound) || in.TypeID <= 0 {
		fields.Add("typeId", validation.CodeInvalid)
	} else if err != nil {
		return nil, err
	} else {
		updated.TypeID, updated.TypeCode, updated.TypeLabel = typeInfo.ID, typeInfo.Code, typeInfo.Label
	}

	updated.VillainIDs = nil
	updated.Villains = nil
	if in.VillainID != nil && strings.TrimSpace(*in.VillainID) != "" {
		villainID, parseErr := uuid.Parse(strings.TrimSpace(*in.VillainID))
		if parseErr != nil {
			fields.Add("villainId", validation.CodeInvalid)
		} else if villainInfo, findErr := s.Villains.Find(ctx, villainID); errors.Is(findErr, villain.ErrNotFound) {
			fields.Add("villainId", validation.CodeInvalid)
		} else if findErr != nil {
			return nil, findErr
		} else {
			updated.VillainIDs = []uuid.UUID{villainID}
			updated.Villains = []VillainReference{{ID: villainID, Alias: villainInfo.Alias}}
		}
	}

	if in.HeroID != nil {
		public, findErr := s.Repo.PublicHeroExists(ctx, *in.HeroID)
		if findErr != nil {
			return nil, findErr
		}
		if !public {
			return nil, ErrHeroNotFound
		}
	}
	if updated.Status == StatusReported && in.HeroID != nil {
		fields.Add("heroId", validation.CodeInvalid)
	}
	if updated.Status != StatusReported && in.HeroID == nil {
		fields.Add("heroId", validation.CodeRequired)
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	return &updated, nil
}

// notify prévient les modérateurs et les héros, sauf le déclarant. Avec des coordonnées, seuls les héros à
// proximité sont prévenus, et leur mail le dit ; s'il n'y en a aucun, ou sans coordonnées, tous les héros actifs
// le sont. Le lieu n'est jamais envoyé par mail.
func (s *Service) notify(ctx context.Context, created *Incident) {
	reporter, err := s.Users.FindByID(ctx, created.ReporterID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "notification d'incident : déclarant introuvable", "incident_id", created.ID, "error", err)
		return
	}
	moderators, err := s.Users.ActiveEmailsByRole(ctx, user.RoleModerator)
	if err != nil {
		s.Logger.ErrorContext(ctx, "notification d'incident : destinataires illisibles", "incident_id", created.ID, "error", err)
		return
	}
	heroes, nearby, err := s.heroRecipients(ctx, created)
	if err != nil {
		s.Logger.ErrorContext(ctx, "notification d'incident : héros illisibles", "incident_id", created.ID, "error", err)
	}
	heroRadiusKm := 0
	if nearby {
		heroRadiusKm = s.Heroes.AlertRadiusKm()
	}

	seen := make(map[string]bool)
	send := func(recipients []string, nearbyRadiusKm int) {
		for _, to := range recipients {
			if to == reporter.Email || seen[to] {
				continue
			}
			seen[to] = true
			if err := s.Notifier.SendIncidentReported(ctx, to, created.Title, created.TypeLabel, nearbyRadiusKm); err != nil {
				s.Logger.ErrorContext(ctx, "notification d'incident impossible", "incident_id", created.ID, "error", err)
			}
		}
	}
	send(moderators, 0)
	send(heroes, heroRadiusKm)
}

// heroRecipients retourne les héros à prévenir, et vrai s'ils l'ont été pour leur proximité.
func (s *Service) heroRecipients(ctx context.Context, created *Incident) ([]string, bool, error) {
	if created.Latitude != nil && created.Longitude != nil {
		nearby, err := s.Heroes.EmailsNear(ctx, *created.Latitude, *created.Longitude)
		if err != nil {
			return nil, false, err
		}
		if len(nearby) > 0 {
			return nearby, true, nil
		}
	}
	all, err := s.Users.ActiveEmailsByRole(ctx, user.RoleHero)
	return all, false, err
}
