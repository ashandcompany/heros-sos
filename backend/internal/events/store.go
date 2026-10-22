package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// writeLockKey sérialise les transactions qui écrivent des événements (testutil prend 7 000 001).
const writeLockKey int64 = 7_000_002

// Store écrit et relit le journal dans PostgreSQL. Il participe à la transaction du contexte : l'événement et
// l'état qu'il raconte réussissent ou échouent ensemble.
//
// Les écritures sont sérialisées par un verrou consultatif de transaction : l'ordre des numéros (seq) est donc
// celui des validations, et un lecteur qui a vu le numéro N a vu tous les précédents. Sans lui, une transaction
// lente pourrait valider le numéro 10 après qu'un lecteur a déjà passé le 11, et celui-ci ne le verrait jamais.
// Le prix : deux transactions qui écrivent des événements ne s'exécutent pas en parallèle (quelques
// millisecondes chacune), ce qui suffit largement au rythme des déclarations et des interventions.
type Store struct {
	db    *gorm.DB
	clock clock.Clock
}

// NewStore crée le journal.
func NewStore(db *gorm.DB, clk clock.Clock) *Store {
	return &Store{db: db, clock: clk}
}

// Lock implémente Recorder.
func (s *Store) Lock(ctx context.Context) error {
	if !database.InTx(ctx) {
		return ErrNoTransaction
	}
	if err := database.Conn(ctx, s.db).Exec(`SELECT pg_advisory_xact_lock(?)`, writeLockKey).Error; err != nil {
		return fmt.Errorf("events : verrou d'écriture : %w", err)
	}
	return nil
}

// Append implémente Recorder.
func (s *Store) Append(ctx context.Context, event Event) (Appended, error) {
	if err := s.Lock(ctx); err != nil {
		return Appended{}, err
	}
	conn := database.Conn(ctx, s.db)

	var current int
	if err := conn.Raw(`SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_id = ?`, event.AggregateID).Scan(&current).Error; err != nil {
		return Appended{}, fmt.Errorf("events : lecture de la version : %w", err)
	}
	if event.ExpectedVersion != nil && *event.ExpectedVersion != current {
		return Appended{}, fmt.Errorf("%w (attendue %d, courante %d)", ErrVersionConflict, *event.ExpectedVersion, current)
	}
	payload, err := encodePayload(event.Payload)
	if err != nil {
		return Appended{}, err
	}

	var commandID *uuid.UUID
	var occurredAt *time.Time
	if command, ok := CommandFrom(ctx); ok {
		id := command.ID
		commandID, occurredAt = &id, command.ClientCreatedAt
	}
	appended := Appended{Version: current + 1}
	err = conn.Raw(`INSERT INTO events
			(event_id, aggregate_type, aggregate_id, version, incident_id, type, payload, actor_id, command_id, occurred_at, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?, ?)
		RETURNING seq`,
		ids.New(), event.AggregateType, event.AggregateID, appended.Version, event.IncidentID, event.Type,
		string(payload), event.ActorID, commandID, occurredAt, s.clock.Now()).Scan(&appended.Seq).Error
	if err != nil {
		return Appended{}, fmt.Errorf("events : écriture de %s : %w", event.Type, err)
	}
	return appended, nil
}

// Version implémente Recorder.
func (s *Store) Version(ctx context.Context, aggregateID uuid.UUID) (int, error) {
	var version int
	err := database.Conn(ctx, s.db).Raw(`SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_id = ?`, aggregateID).Scan(&version).Error
	if err != nil {
		return 0, fmt.Errorf("events : version de l'agrégat : %w", err)
	}
	return version, nil
}

type recordedRow struct {
	Seq           int64
	EventID       uuid.UUID
	Type          string
	AggregateType string
	AggregateID   uuid.UUID
	Version       int
	IncidentID    uuid.UUID
	ActorID       *uuid.UUID
	CommandID     *uuid.UUID
	Payload       string
	OccurredAt    *time.Time
	RecordedAt    time.Time
}

const recordedColumns = `seq, event_id, type, aggregate_type, aggregate_id, version, incident_id, actor_id, command_id,
	payload::text AS payload, occurred_at, recorded_at`

// Since retourne au plus limit événements de numéro supérieur à afterSeq, dans l'ordre du journal.
func (s *Store) Since(ctx context.Context, afterSeq int64, limit int) ([]Recorded, error) {
	var rows []recordedRow
	err := database.Conn(ctx, s.db).Raw(
		`SELECT `+recordedColumns+` FROM events WHERE seq > ? ORDER BY seq LIMIT ?`, afterSeq, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("events : lecture : %w", err)
	}
	return decodeRows(rows)
}

// OfIncident retourne l'histoire d'un incident (ses événements et ceux de ses interventions), dans l'ordre.
func (s *Store) OfIncident(ctx context.Context, incidentID uuid.UUID) ([]Recorded, error) {
	var rows []recordedRow
	err := database.Conn(ctx, s.db).Raw(
		`SELECT `+recordedColumns+` FROM events WHERE incident_id = ? ORDER BY seq`, incidentID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("events : histoire de l'incident : %w", err)
	}
	return decodeRows(rows)
}

// LatestSeq retourne le numéro du dernier événement (0 si le journal est vide).
func (s *Store) LatestSeq(ctx context.Context) (int64, error) {
	var seq int64
	if err := database.Conn(ctx, s.db).Raw(`SELECT COALESCE(MAX(seq), 0) FROM events`).Scan(&seq).Error; err != nil {
		return 0, fmt.Errorf("events : dernier numéro : %w", err)
	}
	return seq, nil
}

func decodeRows(rows []recordedRow) ([]Recorded, error) {
	events := make([]Recorded, len(rows))
	for i, row := range rows {
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
			return nil, fmt.Errorf("events : contenu illisible (événement %d) : %w", row.Seq, err)
		}
		events[i] = Recorded{
			Seq: row.Seq, ID: row.EventID, Type: row.Type, AggregateType: row.AggregateType, AggregateID: row.AggregateID,
			Version: row.Version, IncidentID: row.IncidentID, ActorID: row.ActorID, CommandID: row.CommandID,
			Payload: payload, OccurredAt: row.OccurredAt, RecordedAt: row.RecordedAt,
		}
	}
	return events, nil
}
