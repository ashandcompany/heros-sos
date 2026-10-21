// Package account regroupe ce qu'un titulaire de compte fait de son propre profil : consulter ses
// informations, modifier ses coordonnées, changer sa photo, demander un changement d'adresse email
// (validé par un modérateur) et choisir de recevoir la newsletter. Les citoyens et les super-héros
// partagent les mêmes routes, puisqu'un héros est un compte comme un autre avec un profil en plus.
package account

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

var (
	// ErrEmailUnchanged indique que l'adresse demandée est déjà celle du compte.
	ErrEmailUnchanged = errors.New("account : c'est déjà l'adresse du compte")
	// ErrChangeNotFound indique qu'aucune demande de changement d'adresse ne porte cet identifiant.
	ErrChangeNotFound = errors.New("account : demande de changement d'adresse introuvable")
	// ErrChangeNotPending indique qu'une demande a déjà été traitée.
	ErrChangeNotPending = errors.New("account : demande de changement d'adresse déjà traitée")
)

// ChangeStatus est l'état d'une demande de changement d'adresse.
type ChangeStatus string

const (
	ChangePending  ChangeStatus = "pending"
	ChangeApproved ChangeStatus = "approved"
	ChangeRejected ChangeStatus = "rejected"
)

// Valid indique si le statut est connu.
func (s ChangeStatus) Valid() bool {
	switch s {
	case ChangePending, ChangeApproved, ChangeRejected:
		return true
	}
	return false
}

// EmailChange est une demande de changement d'adresse email.
type EmailChange struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	NewEmail   string
	Status     ChangeStatus
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	CreatedAt  time.Time
}

// EmailChangeView est une demande vue par un modérateur, avec l'identité du demandeur.
type EmailChangeView struct {
	EmailChange
	CurrentEmail string
	FirstName    string
	LastName     string
}

// Account est le profil complet de l'utilisateur connecté.
type Account struct {
	User *user.User
	// PendingEmail est l'adresse demandée et pas encore validée par un modérateur (vide sinon).
	PendingEmail         string
	NewsletterSubscribed bool
	// Hero est le profil de super-héros, pour un compte qui l'est devenu (nil sinon).
	Hero *hero.Profile
}

// ContactInput est le changement de coordonnées demandé : un champ nil est conservé.
type ContactInput struct {
	Phone   *string
	Address *AddressInput
}

// AddressInput est la nouvelle adresse postale. Complement nil conserve le complément actuel, ce qui
// permet à un formulaire sans ce champ (profil d'un héros) de ne pas l'effacer.
type AddressInput struct {
	Street     string
	Complement *string
	PostalCode string
	City       string
	Country    string
}
