// Package validation contient les règles de validation des données saisies par les
// utilisateurs. Elles reprennent celles du formulaire du frontend (frontend/src/lib/validation)
// : le serveur ne fait jamais confiance au client, mais ne doit pas non plus refuser ce que le
// formulaire accepte.
package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Codes d'erreur stables, destinés au frontend (qui les traduit en messages).
const (
	CodeRequired = "required"
	CodeInvalid  = "invalid"
	CodeTooShort = "too_short"
	CodeTooLong  = "too_long"
	CodeWeak     = "weak"
)

// Longueurs maximales des champs texte.
const (
	MaxEmailLength             = 254
	MaxStreetLength            = 200
	MaxAddressComplementLength = 200
	MaxCityLength              = 100
	MaxBioLength               = 1000
	MaxRequestMessageLength    = 2000
	MaxRejectionReasonLength   = 1000
	MaxLabelLength             = 60
	MaxShortDescriptionLength  = 500
	MinIncidentTitleLength     = 5
	MaxIncidentTitleLength     = 80
	MinIncidentDescription     = 20
	MaxIncidentDescription     = 1000
	MaxIdentifierLength        = MaxEmailLength
	MaxAgeInYears              = 120
	isoDateLayout              = "2006-01-02"

	maxNameLength         = 50
	minAliasLength        = 2
	maxAliasLength        = 30
	minInternationalDigit = 8
	maxInternationalDigit = 15
	frPostalCodeLength    = 5
	minPostalCodeLength   = 2
	maxPostalCodeLength   = 10
)

// Valeurs acceptées pour la civilité.
const (
	CivilityMadame     = "madame"
	CivilityMonsieur   = "monsieur"
	CivilityUnspecific = "non-precise"
)

var (
	emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	// Les limites des motifs viennent des constantes ci-dessus : une seule source de vérité.
	namePattern        = regexp.MustCompile(fmt.Sprintf(`^\p{L}[\p{L}' -]{0,%d}$`, maxNameLength-1))
	aliasPattern       = regexp.MustCompile(fmt.Sprintf(`^[\p{L}\p{N} _-]{%d,%d}$`, minAliasLength, maxAliasLength))
	frMobilePattern    = regexp.MustCompile(`^0[67]\d{8}$`)
	intlPhonePattern   = regexp.MustCompile(fmt.Sprintf(`^\+\d{%d,%d}$`, minInternationalDigit, maxInternationalDigit))
	frPostalPattern    = regexp.MustCompile(fmt.Sprintf(`^\d{%d}$`, frPostalCodeLength))
	postalPattern      = regexp.MustCompile(fmt.Sprintf(`^[A-Za-z0-9][A-Za-z0-9 -]{%d,%d}$`, minPostalCodeLength-1, maxPostalCodeLength-1))
	countryPattern     = regexp.MustCompile(`^[A-Z]{2}$`)
	phoneSeparatorsCut = strings.NewReplacer(" ", "", ".", "", "-", "")
)

// Error regroupe les erreurs de plusieurs champs : champ -> code d'erreur.
type Error struct {
	Fields map[string]string
}

func (e *Error) Error() string {
	names := make([]string, 0, len(e.Fields))
	for field := range e.Fields {
		names = append(names, field)
	}
	sort.Strings(names)

	parts := make([]string, len(names))
	for i, field := range names {
		parts[i] = fmt.Sprintf("%s=%s", field, e.Fields[field])
	}
	return "validation : " + strings.Join(parts, ", ")
}

// Collector accumule les erreurs de validation d'une saisie. Seule la première erreur de
// chaque champ est conservée.
type Collector struct {
	fields map[string]string
}

// Add enregistre l'erreur d'un champ si code n'est pas vide.
func (c *Collector) Add(field, code string) {
	if code == "" {
		return
	}
	if c.fields == nil {
		c.fields = make(map[string]string)
	}
	if _, exists := c.fields[field]; !exists {
		c.fields[field] = code
	}
}

// Err retourne nil si tout est valide, sinon une *Error.
func (c *Collector) Err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &Error{Fields: c.fields}
}

// Email normalise (minuscules, sans espaces) et valide une adresse email.
func Email(raw string) (normalized, code string) {
	normalized = strings.ToLower(strings.TrimSpace(raw))
	switch {
	case normalized == "":
		return normalized, CodeRequired
	case len(normalized) > MaxEmailLength:
		return normalized, CodeTooLong
	}
	if !emailPattern.MatchString(normalized) {
		return normalized, CodeInvalid
	}
	// ParseAddress refuse les formes exotiques ; on exige aussi que l'adresse soit « nue »
	// (pas de nom d'affichage ni de chevrons).
	if address, err := mail.ParseAddress(normalized); err != nil || address.Address != normalized {
		return normalized, CodeInvalid
	}
	return normalized, ""
}

// Name valide un nom ou un prénom.
func Name(raw string) (trimmed, code string) {
	trimmed = strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return trimmed, CodeRequired
	case !namePattern.MatchString(trimmed):
		return trimmed, CodeInvalid
	}
	return trimmed, ""
}

// Civility valide la civilité.
func Civility(raw string) string {
	switch raw {
	case CivilityMadame, CivilityMonsieur, CivilityUnspecific:
		return ""
	case "":
		return CodeRequired
	}
	return CodeInvalid
}

// BirthDate valide une date ISO (AAAA-MM-JJ) ni future, ni antérieure à MaxAgeInYears ans.
func BirthDate(raw string, now time.Time) (time.Time, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, CodeRequired
	}
	date, err := time.Parse(isoDateLayout, raw)
	if err != nil {
		return time.Time{}, CodeInvalid
	}
	// La comparaison se fait au jour près : une date de naissance n'a pas d'heure.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	oldest := today.AddDate(-MaxAgeInYears, 0, 0)
	if date.After(today) || date.Before(oldest) {
		return time.Time{}, CodeInvalid
	}
	return date, ""
}

// FormatBirthDate écrit une date de naissance au format ISO.
func FormatBirthDate(date time.Time) string { return date.Format(isoDateLayout) }

// ParseBirthDate relit une date de naissance stockée, sans contrôle de plage.
func ParseBirthDate(stored string) (time.Time, error) { return time.Parse(isoDateLayout, stored) }

// Phone normalise (sans espaces, points ni tirets) et valide un numéro de portable français
// ou international.
func Phone(raw string) (normalized, code string) {
	normalized = phoneSeparatorsCut.Replace(strings.TrimSpace(raw))
	switch {
	case normalized == "":
		return normalized, CodeRequired
	case !frMobilePattern.MatchString(normalized) && !intlPhonePattern.MatchString(normalized):
		return normalized, CodeInvalid
	}
	return normalized, ""
}

// RequiredText valide un texte obligatoire de longueur bornée (rue, ville…).
func RequiredText(raw string, maxLength int) (trimmed, code string) {
	trimmed = strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return trimmed, CodeRequired
	case utf8.RuneCountInString(trimmed) > maxLength:
		return trimmed, CodeTooLong
	}
	return trimmed, ""
}

// BoundedText valide un texte obligatoire dont la longueur (en caractères) est comprise entre
// minLength et maxLength.
func BoundedText(raw string, minLength, maxLength int) (trimmed, code string) {
	trimmed = strings.TrimSpace(raw)
	length := utf8.RuneCountInString(trimmed)
	switch {
	case trimmed == "":
		return trimmed, CodeRequired
	case length < minLength:
		return trimmed, CodeTooShort
	case length > maxLength:
		return trimmed, CodeTooLong
	}
	return trimmed, ""
}

// OptionalText valide un texte facultatif de longueur bornée.
func OptionalText(raw string, maxLength int) (trimmed, code string) {
	trimmed = strings.TrimSpace(raw)
	if utf8.RuneCountInString(trimmed) > maxLength {
		return trimmed, CodeTooLong
	}
	return trimmed, ""
}

// Country valide un code pays ISO 3166-1 alpha-2 en majuscules.
func Country(raw string) (normalized, code string) {
	normalized = strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case normalized == "":
		return normalized, CodeRequired
	case !countryPattern.MatchString(normalized):
		return normalized, CodeInvalid
	}
	return normalized, ""
}

// PostalCode valide un code postal : cinq chiffres pour la France, format libre ailleurs.
func PostalCode(raw, country string) (normalized, code string) {
	normalized = strings.TrimSpace(raw)
	pattern := postalPattern
	if country == "FR" {
		pattern = frPostalPattern
	}
	switch {
	case normalized == "":
		return normalized, CodeRequired
	case !pattern.MatchString(normalized):
		return normalized, CodeInvalid
	}
	return normalized, ""
}

// PasswordPolicy fixe les règles d'un mot de passe.
type PasswordPolicy struct {
	MinLength int
	MaxLength int
}

// Password valide un mot de passe : longueur bornée, avec au moins une minuscule, une
// majuscule et un chiffre. Le mot de passe n'est jamais modifié (pas de trim).
func Password(raw string, policy PasswordPolicy) string {
	length := utf8.RuneCountInString(raw)
	switch {
	case raw == "":
		return CodeRequired
	case length < policy.MinLength:
		return CodeTooShort
	case length > policy.MaxLength:
		return CodeTooLong
	}

	var hasLower, hasUpper, hasDigit bool
	for _, r := range raw {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		return CodeWeak
	}
	return ""
}

// Alias valide le pseudo public d'un super-héros.
func Alias(raw string) (trimmed, code string) {
	trimmed = strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return trimmed, CodeRequired
	case !aliasPattern.MatchString(trimmed):
		return trimmed, CodeInvalid
	}
	return trimmed, ""
}

// IsEmailLike indique si un identifiant de connexion doit être traité comme une adresse email
// (les pseudos de super-héros n'en contiennent jamais).
func IsEmailLike(identifier string) bool { return strings.Contains(identifier, "@") }

// Identifier valide l'identifiant de connexion (email ou pseudo) sans révéler sa nature.
func Identifier(raw string) (normalized, code string) {
	normalized = strings.TrimSpace(raw)
	switch {
	case normalized == "":
		return normalized, CodeRequired
	case utf8.RuneCountInString(normalized) > MaxIdentifierLength:
		return normalized, CodeTooLong
	}
	if IsEmailLike(normalized) {
		return Email(normalized)
	}
	return Alias(normalized)
}
