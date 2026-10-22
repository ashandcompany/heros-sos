package intervention

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

const (
	tableInterventions      = "interventions"
	tableInterventionHeroes = "intervention_heroes"
)

const (
	oneActiveInterventionConstraint = "interventions_one_active_per_incident_idx"
	heroActiveInterventionIndex     = "intervention_heroes_one_active_intervention_idx"
)

type interventionRecord struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	IncidentID     uuid.UUID
	Status         string
	ProposedStatus *string
	ProposedBy     *uuid.UUID
	DecidedBy      *uuid.UUID
	Notes          *string
	StartedAt      *time.Time
	EndedAt        *time.Time
}

func (interventionRecord) TableName() string { return tableInterventions }

type interventionHeroRecord struct {
	InterventionID    uuid.UUID `gorm:"primaryKey"`
	HeroID            uuid.UUID `gorm:"primaryKey"`
	Role              string
	Participation     string
	InvitedBy         *uuid.UUID
	OutcomeValidation string
	IsActive          *bool
	InvitedAt         *time.Time
	JoinedAt          *time.Time
}

func (interventionHeroRecord) TableName() string { return tableInterventionHeroes }

type participantRow struct {
	Record interventionHeroRecord `gorm:"embedded"`
	Alias  *string
}

type joinedInterventionRecord struct {
	ID             uuid.UUID
	IncidentID     uuid.UUID
	Status         string
	ProposedStatus *string
	ProposedBy     *uuid.UUID
	DecidedBy      *uuid.UUID
	Notes          *string
	StartedAt      *time.Time
	EndedAt        *time.Time
	ReporterID     *uuid.UUID
}

// Repository est le stockage et les transitions atomiques d'une intervention.
type Repository interface {
	Start(ctx context.Context, incidentID, leadID uuid.UUID, now time.Time) (*Intervention, error)
	Find(ctx context.Context, id uuid.UUID) (*Intervention, error)
	Join(ctx context.Context, id, heroID uuid.UUID, now time.Time) error
	Invite(ctx context.Context, id, actorID, heroID uuid.UUID, now time.Time) error
	ConfirmInvitation(ctx context.Context, id, heroID uuid.UUID, now time.Time) error
	DeclineInvitation(ctx context.Context, id, heroID uuid.UUID) error
	ProposeOutcome(ctx context.Context, id, heroID uuid.UUID, status Status, notes string, now time.Time) (closed bool, participants []uuid.UUID, err error)
	ValidateOutcome(ctx context.Context, id, heroID uuid.UUID, approved bool, now time.Time) (closed, disputed bool, participants []uuid.UUID, err error)
	Arbitrate(ctx context.Context, id, moderatorID uuid.UUID, status Status, notes string, now time.Time) ([]uuid.UUID, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Start(ctx context.Context, incidentID, leadID uuid.UUID, now time.Time) (*Intervention, error) {
	var incident struct {
		ID         uuid.UUID
		Status     string
		Moderation string
	}
	if err := r.conn(ctx).Table("incidents").Select("id, status, moderation").Where("id = ?", incidentID).Take(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("intervention : lecture de l'incident : %w", err)
	}
	if incident.Moderation != "approved" || incident.Status != "reported" && incident.Status != "failed" {
		return nil, ErrIncidentNotAvailable
	}

	created := interventionRecord{ID: ids.New(), IncidentID: incidentID, Status: string(StatusInProgress), StartedAt: &now}
	if err := r.conn(ctx).Create(&created).Error; err != nil {
		if database.IsUniqueViolation(err, oneActiveInterventionConstraint) {
			return nil, ErrAlreadyActive
		}
		return nil, fmt.Errorf("intervention : démarrage : %w", err)
	}
	lead := interventionHeroRecord{
		InterventionID: created.ID, HeroID: leadID, Role: string(RoleLead),
		Participation: string(ParticipationConfirmed), OutcomeValidation: string(OutcomePending),
		IsActive: boolPointer(true), JoinedAt: &now,
	}
	if err := r.conn(ctx).Create(&lead).Error; err != nil {
		if database.IsUniqueViolation(err, heroActiveInterventionIndex) {
			return nil, ErrHeroAlreadyActive
		}
		return nil, fmt.Errorf("intervention : ajout du lead : %w", err)
	}
	if err := r.conn(ctx).Table("incidents").Where("id = ?", incidentID).
		Updates(map[string]any{"status": string(StatusInProgress), "updated_at": now}).Error; err != nil {
		return nil, fmt.Errorf("intervention : mise à jour de l'incident : %w", err)
	}
	return r.Find(ctx, created.ID)
}

func (r *gormRepository) Find(ctx context.Context, id uuid.UUID) (*Intervention, error) {
	var joined joinedInterventionRecord
	if err := r.conn(ctx).Table(tableInterventions).
		Select("interventions.*, incidents.reporter_id").
		Joins("JOIN incidents ON incidents.id = interventions.incident_id").
		Where("interventions.id = ?", id).Take(&joined).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("intervention : lecture : %w", err)
	}
	rec := interventionRecord{
		ID: joined.ID, IncidentID: joined.IncidentID, Status: joined.Status,
		ProposedStatus: joined.ProposedStatus, ProposedBy: joined.ProposedBy,
		DecidedBy: joined.DecidedBy, Notes: joined.Notes,
		StartedAt: joined.StartedAt, EndedAt: joined.EndedAt,
	}
	var rows []participantRow
	if err := r.conn(ctx).Table(tableInterventionHeroes).
		Select("intervention_heroes.*, hero_profiles.alias AS alias").
		Joins("LEFT JOIN hero_profiles ON hero_profiles.user_id = intervention_heroes.hero_id").
		Where("intervention_heroes.intervention_id = ?", id).
		Order("intervention_heroes.role, intervention_heroes.joined_at, intervention_heroes.hero_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("intervention : lecture des héros : %w", err)
	}
	result := &Intervention{
		ID: rec.ID, IncidentID: rec.IncidentID, ReporterID: joined.ReporterID,
		Status: Status(rec.Status), ProposedBy: rec.ProposedBy,
		DecidedBy: rec.DecidedBy, StartedAt: timeValue(rec.StartedAt), EndedAt: rec.EndedAt,
	}
	if rec.ProposedStatus != nil {
		status := Status(*rec.ProposedStatus)
		result.ProposedStatus = &status
	}
	if rec.Notes != nil {
		result.Notes = *rec.Notes
	}
	result.Participants = make([]Participant, len(rows))
	for i, row := range rows {
		participantRow := row.Record
		participant := Participant{
			HeroID: participantRow.HeroID, Role: ParticipantRole(participantRow.Role),
			Participation: Participation(participantRow.Participation), InvitedBy: participantRow.InvitedBy,
			OutcomeValidation: OutcomeValidation(participantRow.OutcomeValidation),
			IsActive:          participantRow.IsActive != nil && *participantRow.IsActive,
			InvitedAt:         participantRow.InvitedAt, JoinedAt: participantRow.JoinedAt,
		}
		if row.Alias != nil {
			participant.Alias = *row.Alias
		}
		result.Participants[i] = participant
	}
	return result, nil
}

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func (r *gormRepository) lock(ctx context.Context, id uuid.UUID) (*interventionRecord, error) {
	var rec interventionRecord
	err := r.conn(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("intervention : verrouillage : %w", err)
	}
	if rec.Status != string(StatusInProgress) {
		return nil, ErrIncidentNotAvailable
	}
	return &rec, nil
}

func (r *gormRepository) confirmed(ctx context.Context, id, heroID uuid.UUID) (bool, error) {
	var count int64
	err := r.conn(ctx).Model(&interventionHeroRecord{}).Where(
		"intervention_id = ? AND hero_id = ? AND participation = ? AND is_active = TRUE",
		id, heroID, string(ParticipationConfirmed),
	).Count(&count).Error
	return count > 0, err
}

func (r *gormRepository) participantIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.conn(ctx).Model(&interventionHeroRecord{}).Where(
		"intervention_id = ? AND participation = ?", id, string(ParticipationConfirmed),
	).Pluck("hero_id", &ids).Error
	return ids, err
}

func (r *gormRepository) Join(ctx context.Context, id, heroID uuid.UUID, now time.Time) error {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return err
	}
	if rec.ProposedStatus != nil {
		return ErrOutcomeAlreadySet
	}
	var existing interventionHeroRecord
	err = r.conn(ctx).Where("intervention_id = ? AND hero_id = ?", id, heroID).Take(&existing).Error
	if err == nil {
		return ErrAlreadyParticipant
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("intervention : contrôle du participant : %w", err)
	}
	row := interventionHeroRecord{
		InterventionID: id, HeroID: heroID, Role: string(RoleMember), Participation: string(ParticipationConfirmed),
		OutcomeValidation: string(OutcomePending), IsActive: boolPointer(true), JoinedAt: &now,
	}
	if err := r.conn(ctx).Create(&row).Error; err != nil {
		if database.IsUniqueViolation(err, heroActiveInterventionIndex) {
			return ErrHeroAlreadyActive
		}
		return fmt.Errorf("intervention : adhésion : %w", err)
	}
	return nil
}

func (r *gormRepository) Invite(ctx context.Context, id, actorID, heroID uuid.UUID, now time.Time) error {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return err
	}
	if rec.ProposedStatus != nil {
		return ErrOutcomeAlreadySet
	}
	if member, err := r.confirmed(ctx, id, actorID); err != nil {
		return err
	} else if !member {
		return ErrNotParticipant
	}
	row := interventionHeroRecord{
		InterventionID: id, HeroID: heroID, Role: string(RoleMember), Participation: string(ParticipationInvited),
		InvitedBy: &actorID, OutcomeValidation: string(OutcomePending), IsActive: boolPointer(false), InvitedAt: &now,
	}
	if err := r.conn(ctx).Create(&row).Error; err != nil {
		if database.IsUniqueViolation(err, "intervention_heroes_pk") {
			return ErrAlreadyParticipant
		}
		return fmt.Errorf("intervention : invitation : %w", err)
	}
	return nil
}

func (r *gormRepository) ConfirmInvitation(ctx context.Context, id, heroID uuid.UUID, now time.Time) error {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return err
	}
	if rec.ProposedStatus != nil {
		return ErrOutcomeAlreadySet
	}
	result := r.conn(ctx).Table(tableInterventionHeroes).
		Where("intervention_id = ? AND hero_id = ? AND participation = ?", id, heroID, string(ParticipationInvited)).
		Updates(map[string]any{"participation": string(ParticipationConfirmed), "is_active": true, "joined_at": now})
	if result.Error != nil {
		if database.IsUniqueViolation(result.Error, heroActiveInterventionIndex) {
			return ErrHeroAlreadyActive
		}
		return fmt.Errorf("intervention : confirmation : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvitationPending
	}
	return nil
}

func (r *gormRepository) DeclineInvitation(ctx context.Context, id, heroID uuid.UUID) error {
	if _, err := r.lock(ctx, id); err != nil {
		return err
	}
	result := r.conn(ctx).Where("intervention_id = ? AND hero_id = ? AND participation = ?", id, heroID, string(ParticipationInvited)).
		Delete(&interventionHeroRecord{})
	if result.Error != nil {
		return fmt.Errorf("intervention : refus de l'invitation : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvitationPending
	}
	return nil
}

func (r *gormRepository) ProposeOutcome(ctx context.Context, id, heroID uuid.UUID, status Status, notes string, now time.Time) (bool, []uuid.UUID, error) {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return false, nil, err
	}
	if rec.ProposedStatus != nil {
		return false, nil, ErrOutcomeAlreadySet
	}
	if member, err := r.confirmed(ctx, id, heroID); err != nil {
		return false, nil, err
	} else if !member {
		return false, nil, ErrNotParticipant
	}
	participants, err := r.participantIDs(ctx, id)
	if err != nil {
		return false, nil, err
	}
	if len(participants) == 1 {
		if err := r.close(ctx, rec.IncidentID, id, status, nil, notes, now); err != nil {
			return false, nil, err
		}
		return true, participants, nil
	}
	statusValue := string(status)
	if err := r.conn(ctx).Model(&interventionRecord{}).Where("id = ?", id).Updates(map[string]any{
		"proposed_status": statusValue, "proposed_by": heroID, "notes": nullableString(notes),
	}).Error; err != nil {
		return false, nil, fmt.Errorf("intervention : proposition du résultat : %w", err)
	}
	if err := r.conn(ctx).Model(&interventionHeroRecord{}).
		Where("intervention_id = ? AND participation = ?", id, string(ParticipationConfirmed)).
		Update("outcome_validation", string(OutcomePending)).Error; err != nil {
		return false, nil, fmt.Errorf("intervention : réinitialisation des validations : %w", err)
	}
	return false, participants, nil
}

func (r *gormRepository) ValidateOutcome(ctx context.Context, id, heroID uuid.UUID, approved bool, now time.Time) (bool, bool, []uuid.UUID, error) {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return false, false, nil, err
	}
	if rec.ProposedStatus == nil {
		return false, false, nil, ErrOutcomePending
	}
	if member, err := r.confirmed(ctx, id, heroID); err != nil {
		return false, false, nil, err
	} else if !member {
		return false, false, nil, ErrNotParticipant
	}
	participants, err := r.participantIDs(ctx, id)
	if err != nil {
		return false, false, nil, err
	}
	if !approved {
		if err := r.conn(ctx).Model(&interventionRecord{}).Where("id = ?", id).
			Updates(map[string]any{"proposed_status": nil, "proposed_by": nil}).Error; err != nil {
			return false, false, nil, fmt.Errorf("intervention : annulation du résultat : %w", err)
		}
		if err := r.conn(ctx).Model(&interventionHeroRecord{}).
			Where("intervention_id = ? AND participation = ?", id, string(ParticipationConfirmed)).
			Update("outcome_validation", string(OutcomePending)).Error; err != nil {
			return false, false, nil, fmt.Errorf("intervention : réinitialisation des validations : %w", err)
		}
		return false, true, participants, nil
	}
	result := r.conn(ctx).Table(tableInterventionHeroes).
		Where("intervention_id = ? AND hero_id = ? AND participation = ? AND is_active = TRUE AND outcome_validation = ?", id, heroID, string(ParticipationConfirmed), string(OutcomePending)).
		Update("outcome_validation", string(OutcomeApproved))
	if result.Error != nil {
		return false, false, nil, fmt.Errorf("intervention : validation du résultat : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return false, false, nil, ErrOutcomeAlreadyVoted
	}
	var pending int64
	if err := r.conn(ctx).Model(&interventionHeroRecord{}).
		Where("intervention_id = ? AND participation = ? AND outcome_validation <> ?", id, string(ParticipationConfirmed), string(OutcomeApproved)).
		Count(&pending).Error; err != nil {
		return false, false, nil, fmt.Errorf("intervention : comptage des validations : %w", err)
	}
	if pending > 0 {
		return false, false, participants, nil
	}
	status := Status(*rec.ProposedStatus)
	if err := r.close(ctx, rec.IncidentID, id, status, nil, valueString(rec.Notes), now); err != nil {
		return false, false, nil, err
	}
	return true, false, participants, nil
}

func (r *gormRepository) Arbitrate(ctx context.Context, id, moderatorID uuid.UUID, status Status, notes string, now time.Time) ([]uuid.UUID, error) {
	rec, err := r.lock(ctx, id)
	if err != nil {
		return nil, err
	}
	var disputes int64
	if err := r.conn(ctx).Table("audit_logs").Where("entity_id = ? AND action = ?", id, ActionOutcomeDisputed).Count(&disputes).Error; err != nil {
		return nil, fmt.Errorf("intervention : recherche du désaccord : %w", err)
	}
	if disputes == 0 {
		return nil, ErrNoDispute
	}
	participants, err := r.participantIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := r.close(ctx, rec.IncidentID, id, status, &moderatorID, notes, now); err != nil {
		return nil, err
	}
	return participants, nil
}

func (r *gormRepository) close(ctx context.Context, incidentID, id uuid.UUID, status Status, moderatorID *uuid.UUID, notes string, now time.Time) error {
	values := map[string]any{
		"status": string(status), "proposed_status": nil, "ended_at": now,
		"notes": nullableString(notes), "decided_by": moderatorID,
	}
	if err := r.conn(ctx).Model(&interventionRecord{}).Where("id = ?", id).Updates(values).Error; err != nil {
		return fmt.Errorf("intervention : clôture : %w", err)
	}
	if err := r.conn(ctx).Model(&interventionHeroRecord{}).Where("intervention_id = ?", id).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("intervention : fin des participations actives : %w", err)
	}
	if err := r.conn(ctx).Table("incidents").Where("id = ?", incidentID).
		Updates(map[string]any{"status": string(status), "updated_at": now}).Error; err != nil {
		return fmt.Errorf("intervention : mise à jour de l'incident : %w", err)
	}
	return nil
}

func boolPointer(value bool) *bool { return &value }

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
