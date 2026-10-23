package newsletter

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// Codes d'erreur métier de l'envoi des numéros.
const (
	CodeIssueSent   = "newsletter_already_sent"
	CodeNoRecipient = "no_subscribed_recipient"
)

type issuePayload struct {
	Subject         string     `json:"subject"`
	Body            string     `json:"body"`
	RecipientUserID *uuid.UUID `json:"recipientUserId"`
}

// IssueSummaryResponse est un numéro sans son corps (liste).
type IssueSummaryResponse struct {
	ID              uuid.UUID  `json:"id"`
	Subject         string     `json:"subject"`
	RecipientUserID *uuid.UUID `json:"recipientUserId"`
	Sent            bool       `json:"sent"`
	CreatedAt       time.Time  `json:"createdAt"`
	SentAt          *time.Time `json:"sentAt"`
}

// IssueResponse est un numéro complet.
type IssueResponse struct {
	IssueSummaryResponse
	Body string `json:"body"`
}

// SendResponse confirme la prise en charge de l'envoi.
type SendResponse struct {
	Recipients int `json:"recipients"`
}

func newIssueSummary(i Issue) IssueSummaryResponse {
	return IssueSummaryResponse{
		ID: i.ID, Subject: i.Subject, RecipientUserID: i.RecipientUserID, Sent: i.Sent(),
		CreatedAt: i.CreatedAt, SentAt: i.SentAt,
	}
}

func newIssueResponse(i Issue) IssueResponse {
	return IssueResponse{IssueSummaryResponse: newIssueSummary(i), Body: i.BodyHTML}
}

func (h *Handler) issueID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, ErrIssueNotFound)
		return uuid.Nil, false
	}
	return id, true
}

// ListIssues traite GET /admin/newsletters.
func (h *Handler) ListIssues(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	issues, total, err := h.Service.ListIssues(r.Context(), page.Limit, page.Offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]IssueSummaryResponse, len(issues))
	for i, issue := range issues {
		items[i] = newIssueSummary(issue)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPageResponse(items, total, page))
}

// GetIssue traite GET /admin/newsletters/{id}.
func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := h.issueID(w, r)
	if !ok {
		return
	}
	issue, err := h.Service.GetIssue(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newIssueResponse(*issue))
}

// CreateIssue traite POST /admin/newsletters.
func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	var body issuePayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	issue, err := h.Service.CreateIssue(r.Context(), principal.UserID, IssueInput(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newIssueResponse(*issue))
}

// UpdateIssue traite PUT /admin/newsletters/{id}.
func (h *Handler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := h.issueID(w, r)
	if !ok {
		return
	}
	var body issuePayload
	if err := httpx.DecodeJSON(w, r, h.HTTP.MaxBodyBytes, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	issue, err := h.Service.UpdateIssue(r.Context(), principal.UserID, id, IssueInput(body), httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newIssueResponse(*issue))
}

// DeleteIssue traite DELETE /admin/newsletters/{id}.
func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := h.issueID(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteIssue(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// SendIssue traite POST /admin/newsletters/{id}/send : l'envoi est accepté (202), les mails partent
// ensuite en arrière-plan.
func (h *Handler) SendIssue(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := h.issueID(w, r)
	if !ok {
		return
	}
	count, err := h.Service.SendIssue(r.Context(), principal.UserID, id, httpx.ClientIPFrom(r.Context()))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, SendResponse{Recipients: count})
}
