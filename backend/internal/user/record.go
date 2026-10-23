package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const tableUsers = "users"

// Colonnes chiffrées : leur nom entre dans le contexte authentifié (AAD) de chaque valeur.
const (
	columnLastName          = "last_name"
	columnFirstName         = "first_name"
	columnBirthDate         = "birth_date"
	columnPhone             = "phone"
	columnStreet            = "street"
	columnAddressComplement = "address_complement"
	columnPostalCode        = "postal_code"
	columnCity              = "city"
	columnCountry           = "country"
)

// Sealer chiffre et déchiffre une valeur pour un contexte donné (*crypto.Sealer).
type Sealer interface {
	Seal(plaintext []byte, aad crypto.AAD) ([]byte, error)
	Open(sealed []byte, aad crypto.AAD) ([]byte, error)
}

// record est la ligne de la table users : les données personnelles y sont chiffrées.
type record struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	Email             string
	PasswordHash      string
	Role              string
	Status            string
	Civility          string
	LastName          []byte
	FirstName         []byte
	BirthDate         []byte
	Phone             []byte
	Street            []byte
	AddressComplement []byte
	PostalCode        []byte
	City              []byte
	Country           []byte
	AvatarPath        *string
	ActivatedAt       *time.Time
	VerifiedAt        *time.Time
	VerifiedBy        *uuid.UUID
	DeactivatedAt     *time.Time
	DeletedAt         *time.Time
	FailedLoginCount  int
	LockedUntil       *time.Time
	LastLoginAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (record) TableName() string { return tableUsers }

// row ajoute à une ligne users le pseudo issu de hero_profiles (jointure à gauche).
// Le champ Record est exporté : GORM ignore les champs non exportés, donc un champ embarqué
// anonyme de type non exporté ne serait jamais renseigné.
type row struct {
	Record record `gorm:"embedded"`
	Alias  *string
}

// codec chiffre et déchiffre les colonnes d'une ligne précise.
type codec struct {
	sealer Sealer
	id     uuid.UUID
}

func (c codec) aad(column string) crypto.AAD {
	return crypto.AAD{Table: tableUsers, Column: column, RecordID: c.id}
}

func (c codec) seal(column, value string) ([]byte, error) {
	sealed, err := c.sealer.Seal([]byte(value), c.aad(column))
	if err != nil {
		return nil, fmt.Errorf("user : chiffrement de %s : %w", column, err)
	}
	return sealed, nil
}

// sealOptional stocke NULL pour une valeur vide, pour ne pas chiffrer du vide.
func (c codec) sealOptional(column, value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	return c.seal(column, value)
}

func (c codec) open(column string, sealed []byte) (string, error) {
	plaintext, err := c.sealer.Open(sealed, c.aad(column))
	if err != nil {
		return "", fmt.Errorf("user : déchiffrement de %s : %w", column, err)
	}
	return string(plaintext), nil
}

func (c codec) openOptional(column string, sealed []byte) (string, error) {
	if sealed == nil {
		return "", nil
	}
	return c.open(column, sealed)
}

// toRecord chiffre les données personnelles d'un utilisateur pour son stockage.
func toRecord(u *User, sealer Sealer) (record, error) {
	c := codec{sealer: sealer, id: u.ID}
	rec := record{
		ID:               u.ID,
		Email:            u.Email,
		PasswordHash:     u.PasswordHash,
		Role:             string(u.Role),
		Status:           string(u.Status),
		Civility:         u.Civility,
		ActivatedAt:      u.ActivatedAt,
		VerifiedAt:       u.VerifiedAt,
		VerifiedBy:       u.VerifiedBy,
		DeactivatedAt:    u.DeactivatedAt,
		FailedLoginCount: u.FailedLoginCount,
		LockedUntil:      u.LockedUntil,
		LastLoginAt:      u.LastLoginAt,
	}
	if u.AvatarPath != "" {
		rec.AvatarPath = &u.AvatarPath
	}

	var err error
	required := []struct {
		target *[]byte
		column string
		value  string
	}{
		{&rec.LastName, columnLastName, u.LastName},
		{&rec.FirstName, columnFirstName, u.FirstName},
		{&rec.BirthDate, columnBirthDate, validation.FormatBirthDate(u.BirthDate)},
	}
	for _, field := range required {
		if *field.target, err = c.seal(field.column, field.value); err != nil {
			return record{}, err
		}
	}

	optional := []struct {
		target *[]byte
		column string
		value  string
	}{
		{&rec.Phone, columnPhone, u.Phone},
		{&rec.Street, columnStreet, u.Address.Street},
		{&rec.AddressComplement, columnAddressComplement, u.Address.Complement},
		{&rec.PostalCode, columnPostalCode, u.Address.PostalCode},
		{&rec.City, columnCity, u.Address.City},
		{&rec.Country, columnCountry, u.Address.Country},
	}
	for _, field := range optional {
		if *field.target, err = c.sealOptional(field.column, field.value); err != nil {
			return record{}, err
		}
	}
	return rec, nil
}

// toUser déchiffre une ligne en utilisateur.
func (r row) toUser(sealer Sealer) (*User, error) {
	rec := r.Record
	c := codec{sealer: sealer, id: rec.ID}
	u := &User{
		ID:               rec.ID,
		Email:            rec.Email,
		PasswordHash:     rec.PasswordHash,
		Role:             Role(rec.Role),
		Status:           Status(rec.Status),
		Civility:         rec.Civility,
		ActivatedAt:      rec.ActivatedAt,
		VerifiedAt:       rec.VerifiedAt,
		VerifiedBy:       rec.VerifiedBy,
		DeactivatedAt:    rec.DeactivatedAt,
		DeletedAt:        rec.DeletedAt,
		FailedLoginCount: rec.FailedLoginCount,
		LockedUntil:      rec.LockedUntil,
		LastLoginAt:      rec.LastLoginAt,
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
	}
	if rec.AvatarPath != nil {
		u.AvatarPath = *rec.AvatarPath
	}
	if r.Alias != nil {
		u.Alias = *r.Alias
	}

	var err error
	if u.LastName, err = c.open(columnLastName, rec.LastName); err != nil {
		return nil, err
	}
	if u.FirstName, err = c.open(columnFirstName, rec.FirstName); err != nil {
		return nil, err
	}
	birthDate, err := c.open(columnBirthDate, rec.BirthDate)
	if err != nil {
		return nil, err
	}
	if u.BirthDate, err = validation.ParseBirthDate(birthDate); err != nil {
		return nil, fmt.Errorf("user : date de naissance stockée invalide : %w", err)
	}

	optional := []struct {
		target *string
		column string
		sealed []byte
	}{
		{&u.Phone, columnPhone, rec.Phone},
		{&u.Address.Street, columnStreet, rec.Street},
		{&u.Address.Complement, columnAddressComplement, rec.AddressComplement},
		{&u.Address.PostalCode, columnPostalCode, rec.PostalCode},
		{&u.Address.City, columnCity, rec.City},
		{&u.Address.Country, columnCountry, rec.Country},
	}
	for _, field := range optional {
		if *field.target, err = c.openOptional(field.column, field.sealed); err != nil {
			return nil, err
		}
	}
	return u, nil
}
