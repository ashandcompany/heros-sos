package intervention

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

const entityType = "intervention"

const (
	ActionStarted            = "intervention.start"
	ActionJoined             = "intervention.join"
	ActionInvited            = "intervention.invite"
	ActionInvitationAccepted = "intervention.invitation_confirm"
	ActionInvitationDeclined = "intervention.invitation_decline"
	ActionOutcomeProposed    = "intervention.outcome_propose"
	ActionOutcomeValidated   = "intervention.outcome_validate"
	ActionOutcomeDisputed    = "intervention.outcome_dispute"
	ActionOutcomeArbitrated  = "intervention.outcome_arbitrate"
	ActionOutcomeClosed      = "intervention.outcome_close"
)

// HeroFinder lit le profil d'un héros.
type HeroFinder interface {
	Find(ctx context.Context, userID uuid.UUID) (*hero.Profile, error)
}

// Notifier envoie les notifications liées au cycle d'une intervention.
type Notifier interface {
	SendInterventionInvitation(ctx context.Context, to string, interventionID uuid.UUID) error
	SendOutcomeValidation(ctx context.Context, to string, interventionID uuid.UUID) error
	SendOutcomeDisputed(ctx context.Context, to string, interventionID uuid.UUID) error
	SendReviewRequest(ctx context.Context, to, link string) error
}

// ServiceDeps regroupe les dépendances du service d'intervention.
type ServiceDeps struct {
	Repo     Repository
	Users    user.Repository
	Heroes   HeroFinder
	Audit    audit.Recorder
	Events   events.Recorder
	Tx       database.Transactor
	Clock    clock.Clock
	App      config.App
	Notifier Notifier
	Logger   *slog.Logger
}

// Service porte les règles d'engagement et de clôture des interventions.
type Service struct {
	ServiceDeps
}

// NewService crée le service.
func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

// Start démarre une intervention : le héros devient lead confirmé.
func (s *Service) Start(ctx context.Context, heroID, incidentID uuid.UUID, ip netip.Addr) (*Intervention, error) {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return nil, err
	}
	var created *Intervention
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		var err error
		created, err = s.Repo.Start(ctx, incidentID, heroID, s.Clock.Now())
		if err != nil {
			return err
		}
		appended, err := s.emit(ctx, heroID, created.ID, events.InterventionStarted, map[string]any{"heroId": heroID.String()}, nil)
		if err != nil {
			return err
		}
		created.Version = appended.Version
		return s.record(ctx, heroID, created.ID, ActionStarted, ip)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Join ajoute un héros confirmé qui rejoint de lui-même une intervention.
func (s *Service) Join(ctx context.Context, heroID, interventionID uuid.UUID, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return err
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		if err := s.Repo.Join(ctx, interventionID, heroID, s.Clock.Now()); err != nil {
			return err
		}
		if _, err := s.emit(ctx, heroID, interventionID, events.InterventionJoined, map[string]any{"heroId": heroID.String()}, nil); err != nil {
			return err
		}
		return s.record(ctx, heroID, interventionID, ActionJoined, ip)
	})
}

// Invite ajoute des héros approuvés comme participants en attente de confirmation.
func (s *Service) Invite(ctx context.Context, actorID, interventionID uuid.UUID, heroIDs []uuid.UUID, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, actorID); err != nil {
		return err
	}
	unique := make(map[uuid.UUID]struct{}, len(heroIDs))
	for _, heroID := range heroIDs {
		if heroID == actorID {
			return ErrAlreadyParticipant
		}
		if _, duplicate := unique[heroID]; duplicate {
			continue
		}
		unique[heroID] = struct{}{}
		if err := s.requireApprovedHero(ctx, heroID); err != nil {
			return err
		}
	}
	if len(unique) == 0 {
		return fmt.Errorf("intervention : aucune invitation")
	}
	invited := make([]uuid.UUID, 0, len(unique))
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		for heroID := range unique {
			if err := s.Repo.Invite(ctx, interventionID, actorID, heroID, s.Clock.Now()); err != nil {
				return err
			}
			invitation := map[string]any{"heroId": heroID.String(), "invitedBy": actorID.String()}
			if _, err := s.emit(ctx, actorID, interventionID, events.InterventionInvited, invitation, nil); err != nil {
				return err
			}
			if err := s.record(ctx, actorID, interventionID, ActionInvited, ip); err != nil {
				return err
			}
			invited = append(invited, heroID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, heroID := range invited {
		account, err := s.Users.FindByID(ctx, heroID)
		if err != nil {
			s.log(ctx, "lecture du héros invité impossible", interventionID, err)
			continue
		}
		if err := s.Notifier.SendInterventionInvitation(ctx, account.Email, interventionID); err != nil {
			s.log(ctx, "notification d'invitation impossible", interventionID, err)
		}
	}
	return nil
}

// ConfirmInvitation accepte l'invitation et active la participation du héros.
func (s *Service) ConfirmInvitation(ctx context.Context, heroID, interventionID uuid.UUID, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return err
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		if err := s.Repo.ConfirmInvitation(ctx, interventionID, heroID, s.Clock.Now()); err != nil {
			return err
		}
		if _, err := s.emit(ctx, heroID, interventionID, events.InterventionInvitationConfirmed, map[string]any{"heroId": heroID.String()}, nil); err != nil {
			return err
		}
		return s.record(ctx, heroID, interventionID, ActionInvitationAccepted, ip)
	})
}

// DeclineInvitation refuse une invitation non encore confirmée.
func (s *Service) DeclineInvitation(ctx context.Context, heroID, interventionID uuid.UUID, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return err
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		if err := s.Repo.DeclineInvitation(ctx, interventionID, heroID); err != nil {
			return err
		}
		if _, err := s.emit(ctx, heroID, interventionID, events.InterventionInvitationDeclined, map[string]any{"heroId": heroID.String()}, nil); err != nil {
			return err
		}
		return s.record(ctx, heroID, interventionID, ActionInvitationDeclined, ip)
	})
}

// ProposeOutcome clôt immédiatement une intervention solo ou soumet le résultat au vote de l'équipe. Si
// expectedVersion est renseignée, l'action n'est appliquée que si l'intervention est toujours à cette version
// (voir Intervention.Version) : sinon l'écriture de l'événement échoue (ErrVersionConflict) et tout est annulé.
func (s *Service) ProposeOutcome(ctx context.Context, heroID, interventionID uuid.UUID, status Status, notes string, expectedVersion *int, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return err
	}
	if status != StatusSucceeded && status != StatusFailed {
		return ErrOutcomeInvalid
	}
	notes = strings.TrimSpace(notes)
	var participants []uuid.UUID
	closed := false
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		var err error
		closed, participants, err = s.Repo.ProposeOutcome(ctx, interventionID, heroID, status, notes, s.Clock.Now())
		if err != nil {
			return err
		}
		action, eventType := ActionOutcomeProposed, events.InterventionOutcomeProposed
		if closed {
			action, eventType = ActionOutcomeClosed, events.InterventionClosed
		}
		details := map[string]any{"heroId": heroID.String(), "proposedStatus": string(status)}
		if _, err := s.emit(ctx, heroID, interventionID, eventType, details, expectedVersion); err != nil {
			return err
		}
		return s.record(ctx, heroID, interventionID, action, ip)
	})
	if err != nil {
		return err
	}
	if !closed {
		s.notifyHeroes(ctx, participants, interventionID, s.Notifier.SendOutcomeValidation)
	} else {
		s.notifyReviewers(ctx, interventionID, participants)
	}
	return nil
}

// ValidateOutcome approuve ou refuse le résultat proposé par l'équipe. expectedVersion a le même sens que pour
// ProposeOutcome.
func (s *Service) ValidateOutcome(ctx context.Context, heroID, interventionID uuid.UUID, approved bool, expectedVersion *int, ip netip.Addr) error {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return err
	}
	closed, disputed := false, false
	var participants []uuid.UUID
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		var err error
		closed, disputed, participants, err = s.Repo.ValidateOutcome(ctx, interventionID, heroID, approved, s.Clock.Now())
		if err != nil {
			return err
		}
		action, eventType := ActionOutcomeValidated, events.InterventionOutcomeValidated
		switch {
		case disputed:
			action, eventType = ActionOutcomeDisputed, events.InterventionOutcomeDisputed
		case closed:
			eventType = events.InterventionClosed
		}
		if _, err := s.emit(ctx, heroID, interventionID, eventType, map[string]any{"heroId": heroID.String(), "approved": approved}, expectedVersion); err != nil {
			return err
		}
		return s.record(ctx, heroID, interventionID, action, ip)
	})
	if err != nil {
		return err
	}
	if disputed {
		s.notifyModerators(ctx, interventionID)
	} else if closed {
		s.notifyReviewers(ctx, interventionID, participants)
	}
	return nil
}

// Arbitrate tranche un désaccord et clôt l'intervention.
func (s *Service) Arbitrate(ctx context.Context, moderatorID, interventionID uuid.UUID, status Status, notes string, ip netip.Addr) error {
	moderator, err := s.Users.FindByID(ctx, moderatorID)
	if err != nil {
		return err
	}
	if moderator.Role != user.RoleModerator || moderator.Status != user.StatusActive {
		return ErrNotAllowed
	}
	if status != StatusSucceeded && status != StatusFailed {
		return ErrOutcomeInvalid
	}
	var participants []uuid.UUID
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Events.Lock(ctx); err != nil {
			return err
		}
		var err error
		participants, err = s.Repo.Arbitrate(ctx, interventionID, moderatorID, status, strings.TrimSpace(notes), s.Clock.Now())
		if err != nil {
			return err
		}
		if _, err := s.emit(ctx, moderatorID, interventionID, events.InterventionArbitrated, map[string]any{"decidedBy": moderatorID.String()}, nil); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, interventionID, ActionOutcomeArbitrated, ip)
	})
	if err != nil {
		return err
	}
	s.notifyReviewers(ctx, interventionID, participants)
	return nil
}

// Get retourne une intervention à ses participants ou aux modérateurs.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Intervention, error) {
	found, err := s.Repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if found.Version, err = s.Events.Version(ctx, id); err != nil {
		return nil, err
	}
	return found, nil
}

func (s *Service) requireApprovedHero(ctx context.Context, userID uuid.UUID) error {
	account, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if account.Role != user.RoleHero || account.Status != user.StatusActive {
		return ErrHeroNotApproved
	}
	profile, err := s.Heroes.Find(ctx, userID)
	if errors.Is(err, hero.ErrNotFound) || err == nil && profile.Status != hero.StatusApproved {
		return ErrHeroNotApproved
	}
	return err
}

// emit écrit l'événement d'une action dans la transaction en cours. L'intervention est relue après sa mise à
// jour : l'incident concerné et l'état courant (statut de l'intervention) viennent d'elle, ce qui évite à chaque
// appelant de les reconstituer. Le contenu ne comporte que des identifiants et des états.
func (s *Service) emit(ctx context.Context, actorID, interventionID uuid.UUID, eventType string, details map[string]any, expectedVersion *int) (events.Appended, error) {
	current, err := s.Repo.Find(ctx, interventionID)
	if err != nil {
		return events.Appended{}, err
	}
	payload := map[string]any{
		"interventionId": interventionID.String(), "incidentId": current.IncidentID.String(), "status": string(current.Status),
	}
	for key, value := range details {
		payload[key] = value
	}
	return s.Events.Append(ctx, events.Event{
		Type: eventType, AggregateType: events.AggregateIntervention, AggregateID: interventionID,
		IncidentID: current.IncidentID, ActorID: &actorID, Payload: payload, ExpectedVersion: expectedVersion,
	})
}

func (s *Service) record(ctx context.Context, actorID, id uuid.UUID, action string, ip netip.Addr) error {
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: entityType, EntityID: &id, IP: ip,
	})
}

func (s *Service) notifyHeroes(ctx context.Context, heroIDs []uuid.UUID, interventionID uuid.UUID, send func(context.Context, string, uuid.UUID) error) {
	for _, heroID := range heroIDs {
		account, err := s.Users.FindByID(ctx, heroID)
		if err != nil {
			s.log(ctx, "lecture du héros participant impossible", interventionID, err)
			continue
		}
		if err := send(ctx, account.Email, interventionID); err != nil {
			s.log(ctx, "notification au héros impossible", interventionID, err)
		}
	}
}

func (s *Service) notifyModerators(ctx context.Context, interventionID uuid.UUID) {
	emails, err := s.Users.ActiveEmailsByRole(ctx, user.RoleModerator)
	if err != nil {
		s.log(ctx, "lecture des modérateurs impossible", interventionID, err)
		return
	}
	for _, email := range emails {
		if err := s.Notifier.SendOutcomeDisputed(ctx, email, interventionID); err != nil {
			s.log(ctx, "notification de désaccord impossible", interventionID, err)
		}
	}
}

func (s *Service) notifyReviewers(ctx context.Context, interventionID uuid.UUID, heroIDs []uuid.UUID) {
	if s.Notifier == nil {
		return
	}
	intervention, err := s.Repo.Find(ctx, interventionID)
	if err != nil || intervention.ReporterID == nil {
		s.log(ctx, "lecture du déclarant pour la notation impossible", interventionID, err)
		return
	}
	reporterID := *intervention.ReporterID
	reporter, err := s.Users.FindByID(ctx, reporterID)
	if err != nil {
		s.log(ctx, "lecture du déclarant pour la notation impossible", interventionID, err)
		return
	}
	for _, heroID := range heroIDs {
		if heroID == reporterID {
			continue
		}
		link := s.App.HeroReviewLink(interventionID.String(), heroID.String())
		if err := s.Notifier.SendReviewRequest(ctx, reporter.Email, link); err != nil {
			s.log(ctx, "envoi du lien de notation impossible", interventionID, err)
		}
	}
}

func (s *Service) log(ctx context.Context, message string, interventionID uuid.UUID, err error) {
	s.Logger.ErrorContext(ctx, message, "intervention_id", interventionID, "error", err)
}
