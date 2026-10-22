package events

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	shipperName = "log"
	// shipperLockKey désigne l'instance qui journalise : avec plusieurs serveurs, un seul écrit les événements
	// dans les logs, les autres passent leur tour.
	shipperLockKey int64 = 7_000_003
)

// Shipper écrit chaque événement du journal dans les logs applicatifs (une ligne JSON par événement), d'où le
// monitoring (Filebeat, Logstash, Elasticsearch, Kibana) les récupère. Son avancement est en base : un
// redémarrage ne perd rien ; un plantage entre l'écriture et l'enregistrement de l'avancement répète au plus un
// lot (au moins une fois).
type Shipper struct {
	store    *Store
	db       *gorm.DB
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

// NewShipper crée la tâche de journalisation.
func NewShipper(store *Store, db *gorm.DB, logger *slog.Logger, cfg config.Events) *Shipper {
	return &Shipper{store: store, db: db, logger: logger, interval: cfg.PollInterval, batch: cfg.BatchSize}
}

// Run journalise les nouveaux événements jusqu'à l'annulation du contexte.
func (s *Shipper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.drain(ctx)
		}
	}
}

// drain journalise tout le retard, lot après lot.
func (s *Shipper) drain(ctx context.Context) {
	for ctx.Err() == nil {
		shipped, err := s.ShipOnce(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "journalisation des événements en échec", "error", err)
			return
		}
		if shipped < s.batch {
			return
		}
	}
}

// ShipOnce journalise un lot d'événements et retourne leur nombre (0 si une autre instance journalise).
func (s *Shipper) ShipOnce(ctx context.Context) (int, error) {
	shipped := 0
	err := database.NewTransactor(s.db).WithinTx(ctx, func(ctx context.Context) error {
		conn := database.Conn(ctx, s.db)
		var leader bool
		if err := conn.Raw(`SELECT pg_try_advisory_xact_lock(?)`, shipperLockKey).Scan(&leader).Error; err != nil || !leader {
			return err
		}
		var last int64
		if err := conn.Raw(`SELECT COALESCE((SELECT last_seq FROM event_publication WHERE name = ?), 0)`, shipperName).Scan(&last).Error; err != nil {
			return err
		}
		batch, err := s.store.Since(ctx, last, s.batch)
		if err != nil || len(batch) == 0 {
			return err
		}
		for _, event := range batch {
			s.log(ctx, event)
		}
		shipped = len(batch)
		return conn.Exec(`INSERT INTO event_publication (name, last_seq) VALUES (?, ?)
			ON CONFLICT (name) DO UPDATE SET last_seq = EXCLUDED.last_seq`, shipperName, batch[len(batch)-1].Seq).Error
	})
	return shipped, err
}

// log écrit un événement : des champs à plat, faciles à filtrer et à agréger dans Kibana.
func (s *Shipper) log(ctx context.Context, event Recorded) {
	attrs := []slog.Attr{
		slog.String("event_type", event.Type),
		slog.Int64("event_seq", event.Seq),
		slog.String("event_id", event.ID.String()),
		slog.String("aggregate_type", event.AggregateType),
		slog.String("aggregate_id", event.AggregateID.String()),
		slog.Int("aggregate_version", event.Version),
		slog.String("incident_id", event.IncidentID.String()),
		slog.Time("recorded_at", event.RecordedAt),
		slog.Any("payload", event.Payload),
	}
	if event.ActorID != nil {
		attrs = append(attrs, slog.String("actor_id", event.ActorID.String()))
	}
	if event.CommandID != nil {
		attrs = append(attrs, slog.String("command_id", event.CommandID.String()))
	}
	if event.OccurredAt != nil {
		attrs = append(attrs, slog.Time("occurred_at", *event.OccurredAt))
	}
	if delay, ok := event.SyncDelay(); ok {
		attrs = append(attrs, slog.Int64("sync_delay_ms", delay.Milliseconds()))
	}
	s.logger.LogAttrs(ctx, slog.LevelInfo, "événement", attrs...)
}
