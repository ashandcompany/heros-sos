package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Les types ci-dessous sont le contrat JSON de l'API d'authentification, décrit dans
// openapi/auth.yaml (un test vérifie que les réponses respectent ce fichier). Les entités
// (user.User, avec son empreinte de mot de passe et ses données personnelles) ne sont jamais
// sérialisées directement.

// UserResponse est la vue publique d'un compte connecté.
type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
}

// NewUserResponse construit la vue d'un compte.
func NewUserResponse(u *user.User) UserResponse {
	return UserResponse{ID: u.ID, DisplayName: u.DisplayName(), Role: string(u.Role), Email: u.Email}
}

type addressRequest struct {
	Street     string `json:"street"`
	Complement string `json:"complement"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

type registerRequest struct {
	Civility    string         `json:"civility"`
	FirstName   string         `json:"firstName"`
	LastName    string         `json:"lastName"`
	BirthDate   string         `json:"birthDate"`
	Email       string         `json:"email"`
	MobilePhone string         `json:"mobilePhone"`
	Address     addressRequest `json:"address"`
	Password    string         `json:"password"`
}

type registerResponse struct {
	User               UserResponse `json:"user"`
	ActivationRequired bool         `json:"activationRequired"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type loginResponse struct {
	User      UserResponse `json:"user"`
	CSRFToken string       `json:"csrfToken"`
	ExpiresAt time.Time    `json:"expiresAt"`
	MFA       MFAStatus    `json:"mfa"`
}

type meResponse struct {
	User      UserResponse `json:"user"`
	CSRFToken string       `json:"csrfToken"`
	MFA       MFAStatus    `json:"mfa"`
}

type mfaStatusResponse struct {
	Status                 MFAStatus `json:"status"`
	RecoveryCodesRemaining int       `json:"recoveryCodesRemaining"`
}

type mfaSetupResponse struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
}

type mfaCodeRequest struct {
	Code string `json:"code"`
}

type mfaVerifyRequest struct {
	Code         string `json:"code"`
	RecoveryCode string `json:"recoveryCode"`
}

// mfaEnableResponse est la session complète d'une activation réussie, avec les codes de secours (montrés
// cette seule fois).
type mfaEnableResponse struct {
	loginResponse
	RecoveryCodes []string `json:"recoveryCodes"`
}

type recoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type passwordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
