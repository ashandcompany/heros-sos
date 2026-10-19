// Package hero gère la demande du statut de super-héros : un citoyen dépose une demande, un
// modérateur l'accepte ou la refuse (cahier §3.1.2 et §3.1.3).
package hero

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// VerificationStatus est l'état d'une demande.
type VerificationStatus string

const (
	StatusPending  VerificationStatus = "pending"
	StatusApproved VerificationStatus = "approved"
	StatusRejected VerificationStatus = "rejected"
)

// Valid indique si le statut est connu.
func (s VerificationStatus) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	}
	return false
}

var (
	// ErrNotFound indique que l'utilisateur n'a aucune demande.
	ErrNotFound = errors.New("hero : demande introuvable")
	// ErrAlreadyPending indique qu'une demande est déjà en cours d'examen.
	ErrAlreadyPending = errors.New("hero : demande déjà en attente")
	// ErrAlreadyHero indique que l'utilisateur est déjà un super-héros validé.
	ErrAlreadyHero = errors.New("hero : déjà super-héros")
	// ErrAliasTaken indique que le pseudo est déjà pris.
	ErrAliasTaken = errors.New("hero : pseudo déjà utilisé")
	// ErrNotPending indique qu'une demande n'est plus à l'état « en attente » (déjà traitée).
	ErrNotPending = errors.New("hero : la demande n'est pas en attente")
	// ErrNotEligible indique que le compte ne peut pas déposer de demande (seul un citoyen
	// actif le peut).
	ErrNotEligible = errors.New("hero : compte non éligible")
)

// Profile est le profil de super-héros d'un utilisateur.
type Profile struct {
	UserID          uuid.UUID
	Alias           string
	Bio             string
	RequestMessage  string
	Status          VerificationStatus
	ReviewedBy      *uuid.UUID
	ReviewedAt      *time.Time
	RejectionReason string
	CreatedAt       time.Time
}
