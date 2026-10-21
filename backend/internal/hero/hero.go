// Package hero gère la demande du statut de super-héros : un citoyen dépose une demande, un
// modérateur l'accepte ou la refuse (cahier §3.1.2 et §3.1.3).
package hero

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// VerificationStatus est l'état d'une demande.
type VerificationStatus string

// États d'une demande : en attente de décision, acceptée, refusée.
const (
	StatusPending  VerificationStatus = "pending"
	StatusApproved VerificationStatus = "approved"
	StatusRejected VerificationStatus = "rejected"
)

// PublicSort est l'ordre de tri des profils publics de super-héros.
type PublicSort string

// Ordres de tri acceptés par la liste publique (valeur du paramètre `sort`).
const (
	PublicSortAlias         PublicSort = "alias"          // alias, croissant
	PublicSortAverageRating PublicSort = "-averageRating" // note moyenne, décroissante
)

// Valid indique si l'ordre de tri est pris en charge.
func (s PublicSort) Valid() bool {
	return s == PublicSortAlias || s == PublicSortAverageRating
}

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
	// ErrProfileNotFound indique qu'aucun profil de héros public n'existe pour cet identifiant.
	ErrProfileNotFound = errors.New("hero : profil public introuvable")
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

// PublicProfile contient les données publiques d'un super-héros approuvé.
type PublicProfile struct {
	UserID uuid.UUID
	Alias  string
	// AvatarPath est la valeur stockée dans users.avatar_path, sans interprétation : c'est le
	// handler qui en fait une adresse publique (voir resolveAvatarURL).
	AvatarPath         *string
	Bio                string
	AverageRating      float64
	InterventionsCount int64
}

// DirectoryEntry est un super-héros dans l'annuaire des modérateurs : la fiche publique, plus ce que
// seule la modération voit (état du compte, adresse email, demande de changement d'adresse).
type DirectoryEntry struct {
	PublicProfile
	Email         string
	AccountStatus user.Status
	// PendingEmail est l'adresse demandée et pas encore décidée par un modérateur (vide sinon).
	PendingEmail string
	// RegisteredAt est la date de création du compte.
	RegisteredAt time.Time
	// LastInterventionAt est le début de la dernière intervention à laquelle le héros a participé
	// (nil s'il n'a jamais participé).
	LastInterventionAt *time.Time
}

// DirectoryView est une entrée de l'annuaire avec l'identité civile du héros. Celle-ci est chiffrée
// en base : elle est lue fiche par fiche, jamais en SQL.
type DirectoryView struct {
	DirectoryEntry
	Civility  string
	FirstName string
	LastName  string
	BirthDate time.Time
	Phone     string
	Address   user.Address
}
