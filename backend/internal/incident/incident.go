// Package incident gère la déclaration des incidents par les utilisateurs. Le lieu (adresse et
// ville) est chiffré en base ; le type d'intervention et le vilain sont des références vers
// leurs catalogues.
package incident

import (
	"time"

	"github.com/google/uuid"
)

// Status est l'état d'un incident.
type Status string

// StatusReported est l'état d'un incident venant d'être déclaré.
const StatusReported Status = "signale"

// ReporterRole dit si le déclarant est témoin ou victime.
type ReporterRole string

// Rôles du déclarant.
const (
	ReporterWitness ReporterRole = "temoin"
	ReporterVictim  ReporterRole = "victime"
)

// Valid indique si le rôle est connu.
func (r ReporterRole) Valid() bool { return r == ReporterWitness || r == ReporterVictim }

// Incident est un incident déclaré.
type Incident struct {
	ID                 uuid.UUID
	ReporterID         uuid.UUID
	ReporterRole       ReporterRole
	InterventionTypeID uuid.UUID
	// InterventionType est le libellé du type, rempli à la déclaration.
	InterventionType string
	VillainID        *uuid.UUID
	// VillainAlias est le pseudo du vilain, rempli à la déclaration.
	VillainAlias string
	Title        string
	Description  string
	Address      string
	City         string
	OccurredAt   *time.Time
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
