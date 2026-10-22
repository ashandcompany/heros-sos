// Package incident gère la déclaration des incidents par les utilisateurs. Le lieu (adresse et
// ville) est chiffré en base ; le type d'intervention et le vilain sont des références vers
// leurs catalogues.
package incident

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("incident : incident introuvable")
	ErrNotAvailable = errors.New("incident : incident non disponible")
)

// ErrAlreadyExists indique qu'un incident porte déjà l'identifiant fourni par le client. Rejouer une déclaration
// faite hors ligne (même identifiant) tombe ici quand la clé d'idempotence ne l'a pas déjà absorbée.
var ErrAlreadyExists = errors.New("incident : un incident porte déjà cet identifiant")

// Status est l'état d'un incident.
type Status string

const (
	// StatusReported est l'état d'un incident venant d'être déclaré.
	StatusReported   Status = "reported"
	StatusInProgress Status = "in_progress"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
)

// ParseStatus valide un statut reçu en paramètre.
func ParseStatus(raw string) (Status, bool) {
	switch status := Status(raw); status {
	case StatusReported, StatusInProgress, StatusSucceeded, StatusFailed:
		return status, true
	}
	return "", false
}

// Priority est le niveau de priorité d'un incident.
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityNormal   Priority = "normal"
	PriorityLow      Priority = "low"
)

// ModerationStatus est l'état de validation d'une déclaration.
type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

// ReporterRole indique si le déclarant était témoin ou victime.
type ReporterRole string

const (
	ReporterWitness ReporterRole = "witness"
	ReporterVictim  ReporterRole = "victim"
)

// Incident est un incident déclaré.
type Incident struct {
	ID                uuid.UUID
	ReporterID        uuid.UUID
	TypeID            int16
	TypeCode          string
	TypeLabel         string
	Title             string
	Description       string
	Status            Status
	Priority          Priority
	Moderation        ModerationStatus
	ReporterRole      *ReporterRole
	CitizensRescued   int16
	Street            string
	PostalCode        string
	City              string
	Country           string
	AddressComplement string
	Latitude          *float64
	Longitude         *float64
	OccurredAt        *time.Time
	VillainIDs        []uuid.UUID
	Villains          []VillainReference
	InterventionID    *uuid.UUID
	// ResolvedAt est la fin de l'intervention réussie (nil si l'incident n'est pas résolu).
	ResolvedAt *time.Time
	// Lead mène la dernière intervention, nil sans intervention ou si son profil n'est plus public.
	Lead      *HeroReference
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VillainReference est un vilain associé à un incident.
type VillainReference struct {
	ID    uuid.UUID
	Alias string
}
