package intervention

import (
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

const (
	CodeIncidentUnavailable = "incident_not_available"
	CodeHeroNotApproved     = "hero_not_approved"
	CodeInterventionActive  = "intervention_already_active"
	CodeHeroActive          = "hero_already_in_intervention"
	CodeAlreadyParticipant  = "intervention_participant_already_exists"
	CodeNotParticipant      = "intervention_participant_required"
	CodeInvitationPending   = "intervention_invitation_not_pending"
	CodeOutcomePending      = "intervention_outcome_not_pending"
	CodeOutcomeProposed     = "intervention_outcome_already_proposed"
	CodeOutcomeVoted        = "intervention_outcome_already_validated"
	CodeNoDispute           = "intervention_dispute_not_found"
	CodeVersionConflict     = "intervention_version_conflict"
)

// Handler expose les routes du cycle d'intervention.
type Handler struct {
	service *Service
	logger  *slog.Logger
	http    config.HTTP
}

// NewHandler crée le handler.
func NewHandler(service *Service, logger *slog.Logger, httpConfig config.HTTP) *Handler {
	return &Handler{service: service, logger: logger, http: httpConfig}
}

type invitePayload struct {
	HeroIDs []uuid.UUID `json:"heroIds"`
}

type outcomePayload struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
	// ExpectedVersion, facultative, est la version lue par le client : si l'intervention a changé depuis, 409.
	ExpectedVersion *int `json:"expectedVersion"`
}

type outcomeValidationPayload struct {
	Approved        *bool `json:"approved"`
	ExpectedVersion *int  `json:"expectedVersion"`
}

type Response struct {
	ID             uuid.UUID         `json:"id"`
	IncidentID     uuid.UUID         `json:"incidentId"`
	Version        int               `json:"version"`
	Status         string            `json:"status"`
	ProposedStatus *string           `json:"proposedStatus,omitempty"`
	ProposedBy     *uuid.UUID        `json:"proposedBy,omitempty"`
	DecidedBy      *uuid.UUID        `json:"decidedBy,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	StartedAt      time.Time         `json:"startedAt"`
	EndedAt        *time.Time        `json:"endedAt,omitempty"`
	Participants   []participantView `json:"participants"`
}

type participantView struct {
	HeroID            uuid.UUID  `json:"heroId"`
	Alias             string     `json:"alias,omitempty"`
	Role              string     `json:"role"`
	Participation     string     `json:"participation"`
	InvitedBy         *uuid.UUID `json:"invitedBy,omitempty"`
	OutcomeValidation string     `json:"outcomeValidation"`
	IsActive          bool       `json:"isActive"`
	InvitedAt         *time.Time `json:"invitedAt,omitempty"`
	JoinedAt          *time.Time `json:"joinedAt,omitempty"`
}

func responseFrom(in *Intervention) Response {
	participants := make([]participantView, len(in.Participants))
	for i, participant := range in.Participants {
		participants[i] = participantView{
			HeroID: participant.HeroID, Alias: participant.Alias, Role: string(participant.Role),
			Participation: string(participant.Participation), InvitedBy: participant.InvitedBy,
			OutcomeValidation: string(participant.OutcomeValidation), IsActive: participant.IsActive,
			InvitedAt: participant.InvitedAt, JoinedAt: participant.JoinedAt,
		}
	}
	var proposed *string
	if in.ProposedStatus != nil {
		value := string(*in.ProposedStatus)
		proposed = &value
	}
	return Response{
		ID: in.ID, IncidentID: in.IncidentID, Version: in.Version, Status: string(in.Status), ProposedStatus: proposed,
		ProposedBy: in.ProposedBy, DecidedBy: in.DecidedBy, Notes: in.Notes,
		StartedAt: in.StartedAt, EndedAt: in.EndedAt, Participants: participants,
	}
}

// Start traite POST /incidents/{incidentId}/interventions.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	incidentID, ok := parseID(w, r, "incidentId", h)
	if !ok {
		return
	}
	created, err := h.service.Start(r.Context(), principal.UserID, incidentID, clientIP(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, responseFrom(created))
}

// Get traite GET /interventions/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	found, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, responseFrom(found))
}

// ListByIncident traite GET /incidents/{incidentId}/interventions.
func (h *Handler) ListByIncident(w http.ResponseWriter, r *http.Request) {
	incidentID, ok := parseID(w, r, "incidentId", h)
	if !ok {
		return
	}
	found, err := h.service.ListByIncident(r.Context(), incidentID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	items := make([]Response, len(found))
	for i, in := range found {
		items[i] = responseFrom(in)
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Items []Response `json:"items"`
	}{items})
}

// Join traite POST /interventions/{id}/join.
func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	if err := h.service.Join(r.Context(), principal.UserID, id, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Invite traite POST /interventions/{id}/invitations.
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	var body invitePayload
	if err := httpx.DecodeJSON(w, r, h.serviceBodyLimit(), &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if len(body.HeroIDs) == 0 {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains champs sont invalides.", Fields: map[string]string{"heroIds": "required"},
		})
		return
	}
	if err := h.service.Invite(r.Context(), principal.UserID, id, body.HeroIDs, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ConfirmInvitation traite POST /interventions/{id}/invitations/me/confirm.
func (h *Handler) ConfirmInvitation(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	if err := h.service.ConfirmInvitation(r.Context(), principal.UserID, id, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// DeclineInvitation traite POST /interventions/{id}/invitations/me/decline.
func (h *Handler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	if err := h.service.DeclineInvitation(r.Context(), principal.UserID, id, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ProposeOutcome traite POST /interventions/{id}/outcome.
func (h *Handler) ProposeOutcome(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	var body outcomePayload
	if err := httpx.DecodeJSON(w, r, h.serviceBodyLimit(), &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.ProposeOutcome(r.Context(), principal.UserID, id, Status(body.Status), body.Notes, body.ExpectedVersion, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// ValidateOutcome traite POST /interventions/{id}/outcome/validate.
func (h *Handler) ValidateOutcome(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	var body outcomeValidationPayload
	if err := httpx.DecodeJSON(w, r, h.serviceBodyLimit(), &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if body.Approved == nil {
		h.fail(w, r, &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains champs sont invalides.", Fields: map[string]string{"approved": "required"},
		})
		return
	}
	if err := h.service.ValidateOutcome(r.Context(), principal.UserID, id, *body.Approved, body.ExpectedVersion, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

// Arbitrate traite POST /admin/interventions/{id}/decision.
func (h *Handler) Arbitrate(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	id, ok := parseID(w, r, "id", h)
	if !ok {
		return
	}
	var body outcomePayload
	if err := httpx.DecodeJSON(w, r, h.serviceBodyLimit(), &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.Arbitrate(r.Context(), principal.UserID, id, Status(body.Status), body.Notes, clientIP(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func parseID(w http.ResponseWriter, r *http.Request, name string, h *Handler) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		h.fail(w, r, ErrNotFound)
		return uuid.Nil, false
	}
	return id, true
}

func clientIP(r *http.Request) netip.Addr { return httpx.ClientIPFrom(r.Context()) }

func (h *Handler) serviceBodyLimit() int64 { return h.http.MaxBodyBytes }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Fail(w, r, h.logger, toHTTPError(err))
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.NotFound("Intervention introuvable.")
	case errors.Is(err, events.ErrVersionConflict):
		return httpx.Conflict(CodeVersionConflict, "L'intervention a changé depuis la version que vous avez lue.")
	case errors.Is(err, ErrIncidentNotAvailable):
		return httpx.Conflict(CodeIncidentUnavailable, "Cet incident ne peut pas recevoir d'intervention.")
	case errors.Is(err, ErrHeroNotApproved):
		return httpx.NewError(http.StatusForbidden, CodeHeroNotApproved, "Un profil héros approuvé est requis.")
	case errors.Is(err, ErrAlreadyActive):
		return httpx.Conflict(CodeInterventionActive, "Une intervention est déjà en cours pour cet incident.")
	case errors.Is(err, ErrHeroAlreadyActive):
		return httpx.Conflict(CodeHeroActive, "Ce héros participe déjà à une intervention en cours.")
	case errors.Is(err, ErrAlreadyParticipant):
		return httpx.Conflict(CodeAlreadyParticipant, "Ce héros est déjà lié à l'intervention.")
	case errors.Is(err, ErrNotParticipant):
		return httpx.Forbidden("Vous devez participer à cette intervention.")
	case errors.Is(err, ErrInvitationPending):
		return httpx.Conflict(CodeInvitationPending, "Aucune invitation en attente ne correspond.")
	case errors.Is(err, ErrOutcomePending):
		return httpx.Conflict(CodeOutcomePending, "Aucun résultat n'attend de validation.")
	case errors.Is(err, ErrOutcomeAlreadySet):
		return httpx.Conflict(CodeOutcomeProposed, "Un résultat est déjà en attente de validation.")
	case errors.Is(err, ErrOutcomeAlreadyVoted):
		return httpx.Conflict(CodeOutcomeVoted, "Votre validation est déjà enregistrée.")
	case errors.Is(err, ErrNoDispute):
		return httpx.Conflict(CodeNoDispute, "Aucun désaccord ne nécessite d'arbitrage.")
	case errors.Is(err, ErrNotAllowed):
		return httpx.Forbidden("Vous ne pouvez pas arbitrer cette intervention.")
	case errors.Is(err, ErrOutcomeInvalid):
		return &httpx.Error{
			Status: http.StatusUnprocessableEntity, Code: httpx.CodeValidationFailed,
			Message: "Certains champs sont invalides.", Fields: map[string]string{"status": "invalid"},
		}
	}
	return err
}
