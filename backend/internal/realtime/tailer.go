package realtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
)

// Tailer lit les nouveaux événements du journal et les remet au hub. Son avancement n'est qu'en mémoire : à
// chaque démarrage il repart du dernier événement, car les flux ne diffusent que ce qui arrive ensuite (un
// lecteur qui revient rattrape son retard lui-même, depuis le journal). Chaque instance du serveur a le sien.
type Tailer struct {
	store    *events.Store
	db       *gorm.DB
	hub      *Hub
	logger   *slog.Logger
	interval time.Duration
	batch    int
	ready    chan struct{}
}

// NewTailer crée la lecture continue du journal.
func NewTailer(store *events.Store, db *gorm.DB, hub *Hub, logger *slog.Logger, cfg config.Events) *Tailer {
	return &Tailer{
		store: store, db: db, hub: hub, logger: logger, interval: cfg.PollInterval, batch: cfg.BatchSize,
		ready: make(chan struct{}),
	}
}

// Ready est fermé quand la lecture continue a noté où en est le journal : les événements écrits ensuite sont
// diffusés, ceux d'avant non. Un test qui écrit des événements doit l'attendre, sans quoi un événement écrit
// avant ce point serait perdu pour les flux en direct (par conception : un lecteur rattrape depuis le journal).
func (t *Tailer) Ready() <-chan struct{} { return t.ready }

// Run diffuse les nouveaux événements jusqu'à l'annulation du contexte.
func (t *Tailer) Run(ctx context.Context) {
	cursor, err := t.store.LatestSeq(ctx)
	close(t.ready)
	if err != nil {
		t.logger.ErrorContext(ctx, "flux temps réel : lecture du dernier événement impossible", "error", err)
		return
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next, err := t.publishFrom(ctx, cursor)
			if err != nil {
				t.logger.ErrorContext(ctx, "flux temps réel : lecture des événements en échec", "error", err)
				continue
			}
			cursor = next
		}
	}
}

// publishFrom diffuse tous les événements après afterSeq, lot après lot, et retourne le numéro du dernier.
func (t *Tailer) publishFrom(ctx context.Context, afterSeq int64) (int64, error) {
	for ctx.Err() == nil {
		batch, err := t.store.Since(ctx, afterSeq, t.batch)
		if err != nil {
			return afterSeq, err
		}
		enriched, err := Enrich(ctx, t.db, batch)
		if err != nil {
			return afterSeq, err
		}
		for _, event := range enriched {
			t.hub.Publish(event)
			afterSeq = event.Seq
		}
		if len(batch) < t.batch {
			break
		}
	}
	return afterSeq, nil
}

// Enrich ajoute à chaque événement le déclarant de son incident et son état de modération courant, en une requête
// pour tout le lot.
func Enrich(ctx context.Context, db *gorm.DB, batch []events.Recorded) ([]StreamEvent, error) {
	if len(batch) == 0 {
		return nil, nil
	}
	incidentIDs := make([]uuid.UUID, 0, len(batch))
	seen := make(map[uuid.UUID]struct{}, len(batch))
	for _, event := range batch {
		if _, known := seen[event.IncidentID]; !known {
			seen[event.IncidentID] = struct{}{}
			incidentIDs = append(incidentIDs, event.IncidentID)
		}
	}
	var rows []struct {
		ID         uuid.UUID
		ReporterID *uuid.UUID
		Approved   bool
	}
	if err := database.Conn(ctx, db).Table("incidents").Select("id, reporter_id, moderation = 'approved' AS approved").
		Where("id IN ?", incidentIDs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("realtime : lecture des déclarants : %w", err)
	}
	type incidentInfo struct {
		reporter *uuid.UUID
		approved bool
	}
	infos := make(map[uuid.UUID]incidentInfo, len(rows))
	for _, row := range rows {
		infos[row.ID] = incidentInfo{reporter: row.ReporterID, approved: row.Approved}
	}
	enriched := make([]StreamEvent, len(batch))
	for i, event := range batch {
		info := infos[event.IncidentID]
		enriched[i] = StreamEvent{Recorded: event, ReporterID: info.reporter, Approved: info.approved}
	}
	return enriched, nil
}
