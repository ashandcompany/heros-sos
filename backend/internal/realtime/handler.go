package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Handler sert le flux GET /events/stream.
type Handler struct {
	hub    *Hub
	store  *events.Store
	db     *gorm.DB
	logger *slog.Logger
	cfg    config.Events
	// writeTimeout est le délai d'écriture d'un message : le délai d'écriture global du serveur couperait un flux
	// qui dure, il est donc repoussé à chaque message.
	writeTimeout time.Duration
}

// NewHandler crée le handler du flux.
func NewHandler(hub *Hub, store *events.Store, db *gorm.DB, logger *slog.Logger, events config.Events, http config.HTTP) *Handler {
	return &Handler{hub: hub, store: store, db: db, logger: logger, cfg: events, writeTimeout: http.WriteTimeout}
}

// eventView est ce que le navigateur reçoit : des identifiants et des états, rien de plus que le journal.
type eventView struct {
	Seq           int64          `json:"seq"`
	Type          string         `json:"type"`
	AggregateType string         `json:"aggregateType"`
	AggregateID   uuid.UUID      `json:"aggregateId"`
	IncidentID    uuid.UUID      `json:"incidentId"`
	Version       int            `json:"version"`
	ActorID       *uuid.UUID     `json:"actorId,omitempty"`
	Payload       map[string]any `json:"payload"`
	RecordedAt    time.Time      `json:"recordedAt"`
}

// visibleTo dit si le lecteur reçoit l'événement, et sous quelle forme. Les modérateurs et les héros voient tous
// les événements (les incidents sont publics, ainsi que les héros). Un citoyen ne voit que ceux des incidents
// qu'il a déclarés. L'auteur de la déclaration n'est jamais révélé à un autre que lui et les modérateurs :
// l'identité du déclarant n'est pas publique.
func visibleTo(principal auth.Principal, event StreamEvent) (eventView, bool) {
	isReporter := event.ReporterID != nil && *event.ReporterID == principal.UserID
	if principal.Role == user.RoleCitizen && !isReporter {
		return eventView{}, false
	}
	view := eventView{
		Seq: event.Seq, Type: event.Type, AggregateType: event.AggregateType, AggregateID: event.AggregateID,
		IncidentID: event.IncidentID, Version: event.Version, Payload: event.Payload, RecordedAt: event.RecordedAt,
	}
	hidesActor := event.Type == events.IncidentDeclared && principal.Role != user.RoleModerator && !isReporter
	if !hidesActor {
		view.ActorID = event.ActorID
	}
	return view, true
}

// lastEventID lit le numéro du dernier événement reçu : l'en-tête que le navigateur renvoie à chaque reconnexion,
// ou le paramètre lastEventId (pour un client qui n'a pas la main sur les en-têtes).
func lastEventID(r *http.Request) (int64, bool) {
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("lastEventId")
	}
	if raw == "" {
		return 0, false
	}
	seq, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seq < 0 {
		return 0, false
	}
	return seq, true
}

// Stream traite GET /events/stream.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	subscription, err := h.hub.Subscribe(principal.UserID)
	if err != nil {
		if errors.Is(err, ErrTooManyStreams) {
			httpx.WriteError(w, httpx.NewError(http.StatusTooManyRequests, httpx.CodeRateLimited, "Trop de flux ouverts pour ce compte."))
			return
		}
		httpx.WriteError(w, httpx.NewError(http.StatusServiceUnavailable, httpx.CodeInternal, "Le service s'arrête."))
		return
	}
	defer subscription.Close()

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache, no-transform")
	// Dit à un reverse proxy qui mettrait les réponses en mémoire de ne pas retenir celle-ci.
	header.Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	stream := &writer{w: w, controller: controller, timeout: h.writeTimeout}
	// retry : délai avant la reconnexion automatique du navigateur.
	if !stream.raw("retry: 3000\n: connecté\n\n") {
		return
	}

	// Rattrapage depuis le journal, APRÈS l'abonnement : ce qui arrive pendant le rattrapage attend dans la file du
	// flux, et les numéros déjà envoyés sont écartés ensuite.
	sent, resuming := lastEventID(r)
	if resuming {
		var ok bool
		if sent, ok = h.catchUp(r.Context(), principal, stream, sent); !ok {
			return
		}
	} else {
		// Première connexion : seuls comptent les événements qui suivent. Ceux déjà enregistrés mais pas encore
		// remis par la lecture continue sont donc écartés, par leur numéro.
		var err error
		if sent, err = h.store.LatestSeq(r.Context()); err != nil {
			h.logger.ErrorContext(r.Context(), "flux temps réel : dernier événement illisible", "error", err)
			return
		}
	}

	expiry := time.NewTimer(h.cfg.StreamMaxAge)
	defer expiry.Stop()
	heartbeat := time.NewTicker(h.cfg.Heartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-expiry.C:
			return // le navigateur se reconnecte : la session est revérifiée
		case <-heartbeat.C:
			if !stream.raw(": signal de vie\n\n") {
				return
			}
		case event, open := <-subscription.C:
			if !open {
				return
			}
			if event.Seq <= sent {
				continue
			}
			sent = event.Seq
			if view, visible := visibleTo(principal, event); visible && !stream.event(view) {
				return
			}
		}
	}
}

// catchUp rejoue au lecteur les événements qu'il a manqués, et retourne le numéro du dernier lu.
func (h *Handler) catchUp(ctx context.Context, principal auth.Principal, stream *writer, after int64) (int64, bool) {
	for ctx.Err() == nil {
		batch, err := h.store.Since(ctx, after, h.cfg.BatchSize)
		if err != nil {
			h.logger.ErrorContext(ctx, "flux temps réel : rattrapage impossible", "error", err)
			return after, false
		}
		enriched, err := Enrich(ctx, h.db, batch)
		if err != nil {
			h.logger.ErrorContext(ctx, "flux temps réel : rattrapage impossible", "error", err)
			return after, false
		}
		for _, event := range enriched {
			after = event.Seq
			if view, visible := visibleTo(principal, event); visible && !stream.event(view) {
				return after, false
			}
		}
		if len(batch) < h.cfg.BatchSize {
			return after, true
		}
	}
	return after, false
}

// writer écrit des messages SSE et les envoie aussitôt, en repoussant le délai d'écriture à chaque fois.
type writer struct {
	w          http.ResponseWriter
	controller *http.ResponseController
	timeout    time.Duration
}

func (s *writer) raw(text string) bool {
	_ = s.controller.SetWriteDeadline(time.Now().Add(s.timeout))
	if _, err := fmt.Fprint(s.w, text); err != nil {
		return false
	}
	return s.controller.Flush() == nil
}

// event écrit un événement : id (numéro, repris dans Last-Event-ID), event (type) et data (JSON sur une ligne).
func (s *writer) event(view eventView) bool {
	data, err := json.Marshal(view)
	if err != nil {
		return true // un événement illisible est écarté, le flux continue
	}
	return s.raw(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", view.Seq, view.Type, data))
}
