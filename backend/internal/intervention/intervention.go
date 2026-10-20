// Package intervention gère l'engagement d'équipes de héros sur les incidents.
package intervention

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status est l'état d'une intervention.
type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
)

// ParticipantRole indique la place d'un héros dans l'équipe.
type ParticipantRole string

const (
	RoleLead   ParticipantRole = "lead"
	RoleMember ParticipantRole = "member"
)

// Participation décrit l'état d'une invitation.
type Participation string

const (
	ParticipationInvited   Participation = "invited"
	ParticipationConfirmed Participation = "confirmed"
)

// OutcomeValidation est le vote d'un héros confirmé sur le résultat proposé.
type OutcomeValidation string

const (
	OutcomePending  OutcomeValidation = "pending"
	OutcomeApproved OutcomeValidation = "approved"
	OutcomeRejected OutcomeValidation = "rejected"
)

var (
	ErrNotFound             = errors.New("intervention : introuvable")
	ErrIncidentNotAvailable = errors.New("intervention : incident non disponible")
	ErrHeroNotApproved      = errors.New("intervention : héros non approuvé")
	ErrAlreadyActive        = errors.New("intervention : intervention déjà active")
	ErrHeroAlreadyActive    = errors.New("intervention : héros déjà actif ailleurs")
	ErrAlreadyParticipant   = errors.New("intervention : héros déjà lié")
	ErrNotParticipant       = errors.New("intervention : héros non participant")
	ErrInvitationPending    = errors.New("intervention : invitation en attente requise")
	ErrOutcomePending       = errors.New("intervention : aucun résultat en attente")
	ErrOutcomeAlreadySet    = errors.New("intervention : résultat déjà proposé")
	ErrOutcomeAlreadyVoted  = errors.New("intervention : validation déjà enregistrée")
	ErrNoDispute            = errors.New("intervention : aucun désaccord à arbitrer")
	ErrNotAllowed           = errors.New("intervention : action non autorisée")
	ErrOutcomeInvalid       = errors.New("intervention : résultat invalide")
)

// Participant représente la ligne intervention_heroes.
type Participant struct {
	HeroID            uuid.UUID
	Alias             string
	Role              ParticipantRole
	Participation     Participation
	InvitedBy         *uuid.UUID
	OutcomeValidation OutcomeValidation
	IsActive          bool
	InvitedAt         *time.Time
	JoinedAt          *time.Time
}

// Intervention est l'engagement d'une équipe sur un incident.
type Intervention struct {
	ID             uuid.UUID
	IncidentID     uuid.UUID
	ReporterID     *uuid.UUID
	Status         Status
	ProposedStatus *Status
	ProposedBy     *uuid.UUID
	DecidedBy      *uuid.UUID
	Notes          string
	StartedAt      time.Time
	EndedAt        *time.Time
	Participants   []Participant
}
