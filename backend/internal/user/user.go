// Package user gère les comptes : modèle, stockage (avec chiffrement des données
// personnelles) et règles d'état (activation, désactivation, verrouillage).
package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role détermine les droits d'un compte.
type Role string

// Rôles d'un compte.
const (
	RoleCitizen   Role = "citizen"
	RoleHero      Role = "hero"
	RoleModerator Role = "moderator"
)

// Valid indique si le rôle est connu.
func (r Role) Valid() bool {
	switch r {
	case RoleCitizen, RoleHero, RoleModerator:
		return true
	}
	return false
}

// Status décrit le cycle de vie d'un compte.
type Status string

// États d'un compte : à activer par mail, actif, désactivé par un modérateur.
const (
	StatusPendingActivation Status = "pending_activation"
	StatusActive            Status = "active"
	StatusDeactivated       Status = "deactivated"
)

var (
	// ErrNotFound indique qu'aucun compte ne correspond.
	ErrNotFound = errors.New("user : compte introuvable")
	// ErrEmailTaken indique qu'un compte utilise déjà cette adresse email.
	ErrEmailTaken = errors.New("user : adresse email déjà utilisée")
)

// Address est l'adresse postale d'un utilisateur.
type Address struct {
	Street     string
	Complement string
	PostalCode string
	City       string
	Country    string
}

// ContactUpdate liste les coordonnées à remplacer. Un champ nil est conservé tel quel.
type ContactUpdate struct {
	Phone   *string
	Address *AddressUpdate
}

// AddressUpdate remplace l'adresse postale d'un compte.
type AddressUpdate struct {
	Street     string
	PostalCode string
	City       string
	Country    string
	// Complement nil conserve le complément actuel ; une chaîne vide l'efface.
	Complement *string
}

// User est un compte, avec ses données personnelles en clair (le chiffrement n'existe que
// dans la couche de stockage).
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	Status       Status
	Civility     string
	FirstName    string
	LastName     string
	BirthDate    time.Time
	Phone        string
	Address      Address
	AvatarPath   string
	// Alias est le pseudo public d'un super-héros (vide pour les autres comptes).
	Alias string

	ActivatedAt      *time.Time
	VerifiedAt       *time.Time
	VerifiedBy       *uuid.UUID
	DeactivatedAt    *time.Time
	FailedLoginCount int
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DisplayName est le nom affiché dans l'interface : le pseudo pour un super-héros, le prénom
// sinon (la vraie identité d'un héros n'est jamais montrée publiquement).
func (u User) DisplayName() string {
	if u.Role == RoleHero && u.Alias != "" {
		return u.Alias
	}
	return u.FirstName
}

// IsLocked indique si le compte est verrouillé à l'instant donné.
func (u User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}
