package account

import (
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Les types ci-dessous sont le contrat JSON de l'API du profil, décrit dans openapi/openapi.yaml (un
// test vérifie que les réponses le respectent). Les entités (user.User, avec son empreinte de mot de
// passe) ne sont jamais sérialisées directement.

// Response est le profil complet de l'utilisateur connecté.
type Response struct {
	ID                    uuid.UUID       `json:"id"`
	Role                  string          `json:"role"`
	Civility              string          `json:"civility"`
	FirstName             string          `json:"firstName"`
	LastName              string          `json:"lastName"`
	BirthDate             string          `json:"birthDate"`
	Email                 string          `json:"email"`
	PendingEmail          string          `json:"pendingEmail,omitempty"`
	PendingEmailConfirmed bool            `json:"pendingEmailConfirmed,omitempty"`
	MobilePhone           string          `json:"mobilePhone"`
	Address               addressResponse `json:"address"`
	AvatarURL             string          `json:"avatarUrl,omitempty"`
	NewsletterSubscribed  bool            `json:"newsletterSubscribed"`
	CreatedAt             time.Time       `json:"createdAt"`
	Hero                  *heroResponse   `json:"hero,omitempty"`
}

type addressResponse struct {
	Street     string `json:"street"`
	Complement string `json:"complement,omitempty"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

type heroResponse struct {
	Alias string `json:"alias"`
	Bio   string `json:"bio"`
	// LocationConsentAt est absente tant que le héros n'a pas consenti au partage de sa position.
	LocationConsentAt *time.Time `json:"locationConsentAt,omitempty"`
}

func (h *Handler) newResponse(a *Account) Response {
	u := a.User
	response := Response{
		ID: u.ID, Role: string(u.Role), Civility: u.Civility, FirstName: u.FirstName, LastName: u.LastName,
		BirthDate: validation.FormatBirthDate(u.BirthDate), Email: u.Email, PendingEmail: a.PendingEmail, PendingEmailConfirmed: a.PendingEmailConfirmed,
		MobilePhone: u.Phone, NewsletterSubscribed: a.NewsletterSubscribed, CreatedAt: u.CreatedAt,
		Address: addressResponse{
			Street: u.Address.Street, Complement: u.Address.Complement, PostalCode: u.Address.PostalCode,
			City: u.Address.City, Country: u.Address.Country,
		},
	}
	if u.AvatarPath != "" {
		response.AvatarURL = h.AvatarURLPrefix + u.AvatarPath
	}
	if a.Hero != nil {
		response.Hero = &heroResponse{Alias: a.Hero.Alias, Bio: a.Hero.Bio, LocationConsentAt: a.Hero.LocationConsentAt}
	}
	return response
}

type contactRequest struct {
	MobilePhone *string         `json:"mobilePhone"`
	Address     *addressRequest `json:"address"`
}

type addressRequest struct {
	Street     string  `json:"street"`
	Complement *string `json:"complement"`
	PostalCode string  `json:"postalCode"`
	City       string  `json:"city"`
	Country    string  `json:"country"`
}

type avatarResponse struct {
	AvatarURL string `json:"avatarUrl"`
}

type emailChangeRequest struct {
	Email string `json:"email"`
}

// EmailChangeStatusResponse est la demande de changement d'adresse que l'utilisateur vient de déposer.
type EmailChangeStatusResponse struct {
	NewEmail string `json:"newEmail"`
	Status   string `json:"status"`
}

type confirmRequest struct {
	Token string `json:"token"`
}

type newsletterRequest struct {
	Subscribed *bool `json:"subscribed"`
}

// EmailChangeResponse est une demande de changement d'adresse vue par un modérateur.
type EmailChangeResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	CurrentEmail string    `json:"currentEmail"`
	NewEmail     string    `json:"newEmail"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	// ConfirmedAt est la date à laquelle le titulaire de la nouvelle adresse l'a confirmée (absente tant
	// qu'elle ne l'est pas) : un modérateur ne peut accepter qu'une demande confirmée.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	ReviewedAt  *time.Time `json:"reviewedAt,omitempty"`
}
