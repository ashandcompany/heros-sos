package account

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Codes d'erreur de la suppression d'un compte.
const (
	CodeNotErasable     = "account_not_erasable"
	CodeAlreadyErased   = "account_already_erased"
	exportFilenameStamp = "2006-01-02"
)

// exportResponse est l'export des données d'un compte : le profil de GET /me et toute l'activité.
type exportResponse struct {
	ExportedAt time.Time `json:"exportedAt"`
	Account    Response  `json:"account"`
	*Activity
}

type deleteAccountRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// Export traite GET /me/export : un fichier JSON à télécharger.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	if allowed, retryAfter := h.exportPerUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	export, err := h.Service.Export(r.Context(), principal.UserID, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	now := h.Clock.Now()
	w.Header().Set("Content-Disposition", `attachment; filename="heros-sos-mes-donnees-`+now.Format(exportFilenameStamp)+`.json"`)
	httpx.WriteJSON(w, http.StatusOK, exportResponse{
		ExportedAt: now, Account: h.newResponse(export.Account), Activity: export.Activity,
	})
}

// DeleteAccount traite DELETE /me : le titulaire supprime (anonymise) son compte.
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body deleteAccountRequest
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if allowed, retryAfter := h.passwordCheckPerUser.Allow(principal.UserID.String()); !allowed {
		h.fail(w, r, httpx.RateLimited(retryAfter))
		return
	}
	if err := h.Service.DeleteOwnAccount(r.Context(), principal.UserID, body.Password, body.Code,
		httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Anonymize traite POST /admin/users/{id}/anonymize.
func (h *Handler) Anonymize(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, user.ErrNotFound)
		return
	}
	if err := h.Service.AnonymizeByModerator(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}
