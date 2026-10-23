package auth

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// MFAStatus traite GET /auth/mfa : où en est la 2FA de la session, et combien de codes de secours restent.
func (h *Handler) MFAStatus(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	remaining := 0
	if principal.MFAEnrolled {
		var err error
		if remaining, err = h.MFA.RecoveryCodesRemaining(r.Context(), principal.UserID); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusOK, mfaStatusResponse{
		Status: NewMFAStatus(principal.MFAEnrolled, principal.MFAVerified), RecoveryCodesRemaining: remaining,
	})
}

// MFASetup traite POST /auth/mfa/setup : un secret à configurer dans l'application d'authentification.
func (h *Handler) MFASetup(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	if err := h.allow(h.mfaPerUser, principal.UserID.String()); err != nil {
		h.fail(w, r, err)
		return
	}
	setup, err := h.MFA.Setup(r.Context(), principal.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, mfaSetupResponse{Secret: setup.Secret, OtpauthURL: setup.URL})
}

// MFAEnable traite POST /auth/mfa/enable : le premier code active la 2FA, rend les codes de secours et
// ouvre une session complète (nouveau cookie, nouveau jeton CSRF).
func (h *Handler) MFAEnable(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	var body mfaCodeRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		h.fail(w, r, &validation.Error{Fields: map[string]string{"code": validation.CodeRequired}})
		return
	}
	if err := h.allow(h.mfaPerUser, principal.UserID.String()); err != nil {
		h.fail(w, r, err)
		return
	}
	sessionToken, _ := h.Cookies.Read(r)
	enabled, err := h.MFA.Enable(r.Context(), principal.UserID, sessionToken, body.Code, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	response, ok := h.completedSession(w, r, principal.UserID, enabled.Session)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, mfaEnableResponse{loginResponse: response, RecoveryCodes: enabled.RecoveryCodes})
}

// MFAVerify traite POST /auth/mfa/verify : un code TOTP, ou un code de secours, valide la session.
func (h *Handler) MFAVerify(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	var body mfaVerifyRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if strings.TrimSpace(body.Code) == "" && strings.TrimSpace(body.RecoveryCode) == "" {
		h.fail(w, r, &validation.Error{Fields: map[string]string{"code": validation.CodeRequired}})
		return
	}
	if err := h.allow(h.mfaPerUser, principal.UserID.String()); err != nil {
		h.fail(w, r, err)
		return
	}
	sessionToken, _ := h.Cookies.Read(r)
	session, err := h.MFA.Verify(r.Context(), principal.UserID, sessionToken,
		MFAVerifyInput(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	response, ok := h.completedSession(w, r, principal.UserID, session)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// MFARecoveryCodes traite POST /auth/mfa/recovery-codes : de nouveaux codes de secours, contre le code TOTP courant.
func (h *Handler) MFARecoveryCodes(w http.ResponseWriter, r *http.Request) {
	principal, _ := PrincipalFrom(r.Context())
	var body mfaCodeRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		h.fail(w, r, &validation.Error{Fields: map[string]string{"code": validation.CodeRequired}})
		return
	}
	if err := h.allow(h.mfaPerUser, principal.UserID.String()); err != nil {
		h.fail(w, r, err)
		return
	}
	codes, err := h.MFA.RegenerateRecoveryCodes(r.Context(), principal.UserID, body.Code, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

// completedSession pose le cookie de la session complète et construit la réponse (utilisateur, nouveau jeton
// CSRF). Elle écrit elle-même l'erreur et retourne false en cas d'échec.
func (h *Handler) completedSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID, session IssuedSession) (loginResponse, bool) {
	current, err := h.Service.Me(r.Context(), userID)
	if err != nil {
		h.fail(w, r, err)
		return loginResponse{}, false
	}
	h.Cookies.Set(w, session.Token, session.ExpiresAt, h.Clock.Now())
	return loginResponse{
		User: NewUserResponse(current), CSRFToken: h.CSRF.Token(session.Token), ExpiresAt: session.ExpiresAt,
		MFA: MFAStatusVerified,
	}, true
}
