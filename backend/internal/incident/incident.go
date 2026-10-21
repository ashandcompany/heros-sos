// Package incident gère la déclaration des incidents par les utilisateurs. Le lieu (adresse et
// ville) est chiffré en base ; le type d'intervention et le vilain sont des références vers
// leurs catalogues.
package incident

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("incident : incident introuvable")

// Status est l'état d'un incident.
type Status string

// StatusReported est l'état d'un incident venant d'être déclaré.
const StatusReported Status = "reported"

// Priority est le niveau de priorité d'un incident.
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// Incident est un incident déclaré.
type Incident struct {
	ID                uuid.UUID
	ReporterID        uuid.UUID
	TypeID            int16
	TypeCode          string
	TypeLabel         string
	Description       string
	Status            Status
	Priority          Priority
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
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// VillainReference est un vilain associé à un incident.
type VillainReference struct {
	ID    uuid.UUID
	Alias string
}
