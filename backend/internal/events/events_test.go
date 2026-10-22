package events_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

var startAt = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

type fixture struct {
	db    *gorm.DB
	store *events.Store
	tx    *database.GormTransactor
	clock *clock.Fake
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	clk := clock.NewFake(startAt)
	return fixture{db: db, store: events.NewStore(db, clk), tx: database.NewTransactor(db), clock: clk}
}

func declared(incidentID uuid.UUID) events.Event {
	return events.Event{
		Type: events.IncidentDeclared, AggregateType: events.AggregateIncident, AggregateID: incidentID, IncidentID: incidentID,
		Payload: map[string]any{"typeId": 3},
	}
}

func (f fixture) append(t *testing.T, ctx context.Context, event events.Event) events.Appended {
	t.Helper()
	var appended events.Appended
	err := f.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		appended, err = f.store.Append(ctx, event)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return appended
}

func TestAppendNumbersEventsPerAggregate(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, second := ids.New(), ids.New()

	a := f.append(t, ctx, declared(first))
	b := f.append(t, ctx, events.Event{Type: events.InterventionStarted, AggregateType: events.AggregateIntervention, AggregateID: ids.New(), IncidentID: first})
	c := f.append(t, ctx, declared(second))
	d := f.append(t, ctx, events.Event{Type: events.InterventionJoined, AggregateType: events.AggregateIncident, AggregateID: first, IncidentID: first})

	if a.Version != 1 || b.Version != 1 || c.Version != 1 || d.Version != 2 {
		t.Errorf("versions = %d %d %d %d, attendu 1 1 1 2 : une version par agrégat", a.Version, b.Version, c.Version, d.Version)
	}
	if a.Seq >= b.Seq || b.Seq >= c.Seq || c.Seq >= d.Seq {
		t.Errorf("numéros = %d %d %d %d : ils doivent croître", a.Seq, b.Seq, c.Seq, d.Seq)
	}
	history, err := f.store.OfIncident(ctx, first)
	if err != nil || len(history) != 3 || history[0].Type != events.IncidentDeclared || history[2].Version != 2 {
		t.Errorf("histoire de l'incident = %+v, %v : trois événements dans l'ordre", history, err)
	}
	if got := history[0].Payload["typeId"]; got != float64(3) {
		t.Errorf("contenu = %v", history[0].Payload)
	}
}

func TestAppendRefusesToWriteOutsideATransaction(t *testing.T) {
	f := newFixture(t)

	_, err := f.store.Append(context.Background(), declared(ids.New()))
	if !errors.Is(err, events.ErrNoTransaction) {
		t.Fatalf("erreur = %v, attendu ErrNoTransaction", err)
	}
	if err := f.store.Lock(context.Background()); !errors.Is(err, events.ErrNoTransaction) {
		t.Errorf("Lock hors transaction : %v", err)
	}
}

func TestAppendFollowsTheTransactionOfTheState(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	incidentID := ids.New()
	boom := errors.New("l'état n'a pas pu être écrit")

	err := f.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := f.store.Append(ctx, declared(incidentID)); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("erreur = %v", err)
	}
	if latest, err := f.store.LatestSeq(ctx); err != nil || latest != 0 {
		t.Errorf("une transaction annulée a laissé un événement (dernier numéro %d, %v)", latest, err)
	}
}

func TestExpectedVersionDetectsAConcurrentChange(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	incidentID := ids.New()
	f.append(t, ctx, declared(incidentID))

	one, zero := 1, 0
	stale := declared(incidentID)
	stale.ExpectedVersion = &zero
	err := f.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, err := f.store.Append(ctx, stale)
		return err
	})
	if !errors.Is(err, events.ErrVersionConflict) {
		t.Fatalf("version périmée : erreur = %v, attendu ErrVersionConflict", err)
	}

	current := declared(incidentID)
	current.ExpectedVersion = &one
	if appended := f.append(t, ctx, current); appended.Version != 2 {
		t.Errorf("version attendue à jour : version %d, attendu 2", appended.Version)
	}
}

func TestAppendKeepsTheCommandAndTheClientClock(t *testing.T) {
	f := newFixture(t)
	command := events.Command{ID: ids.New()}
	created := startAt.Add(-5 * time.Minute)
	command.ClientCreatedAt = &created
	ctx := events.WithCommand(context.Background(), command)

	f.append(t, ctx, declared(ids.New()))

	recorded, err := f.store.Since(context.Background(), 0, 10)
	if err != nil || len(recorded) != 1 {
		t.Fatalf("événements = %v, %v", recorded, err)
	}
	got := recorded[0]
	if got.CommandID == nil || *got.CommandID != command.ID || got.OccurredAt == nil || !got.OccurredAt.Equal(created) {
		t.Errorf("commande = %v, horloge client = %v", got.CommandID, got.OccurredAt)
	}
	if delay, ok := got.SyncDelay(); !ok || delay != 5*time.Minute {
		t.Errorf("délai de synchronisation = %v (%v), attendu 5 minutes", delay, ok)
	}

	// Un événement sans commande ni horloge client n'a pas de délai à mesurer.
	f.append(t, context.Background(), declared(ids.New()))
	recorded, _ = f.store.Since(context.Background(), got.Seq, 10)
	if _, ok := recorded[0].SyncDelay(); ok || recorded[0].CommandID != nil {
		t.Errorf("sans commande : %+v", recorded[0])
	}
}

func TestJournalIsAppendOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.append(t, ctx, declared(ids.New()))

	for _, statement := range []string{`UPDATE events SET type = 'x'`, `DELETE FROM events`} {
		err := f.db.Exec(statement).Error
		if err == nil || !strings.Contains(err.Error(), "ajout seul") {
			t.Errorf("%s : erreur = %v, attendu un refus (journal en ajout seul)", statement, err)
		}
	}
	if latest, _ := f.store.LatestSeq(ctx); latest != 1 {
		t.Errorf("le journal a été modifié (dernier numéro %d)", latest)
	}
}

// Deux transactions qui écrivent des événements ne se chevauchent pas : la seconde attend la validation de la
// première. C'est ce qui garantit qu'un lecteur ayant vu le numéro N a vu tous les précédents.
func TestWritersAreSerializedSoNumbersFollowCommitOrder(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	firstHasLock := make(chan struct{})
	releaseFirst := make(chan struct{})
	var order []string
	var mu sync.Mutex
	note := func(step string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, step)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := f.tx.WithinTx(ctx, func(ctx context.Context) error {
			if _, err := f.store.Append(ctx, declared(ids.New())); err != nil {
				return err
			}
			close(firstHasLock)
			<-releaseFirst
			note("première validée")
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		<-firstHasLock
		err := f.tx.WithinTx(ctx, func(ctx context.Context) error {
			_, err := f.store.Append(ctx, declared(ids.New()))
			note("seconde écrite")
			return err
		})
		if err != nil {
			t.Error(err)
		}
	}()

	<-firstHasLock
	time.Sleep(150 * time.Millisecond) // laisse à la seconde le temps de se bloquer, si elle le doit
	mu.Lock()
	early := len(order)
	mu.Unlock()
	close(releaseFirst)
	wg.Wait()

	if early != 0 {
		t.Errorf("la seconde transaction a écrit pendant que la première tenait le verrou : %v", order)
	}
	if len(order) != 2 || order[0] != "première validée" {
		t.Errorf("ordre = %v, attendu la première validée avant la seconde écrite", order)
	}
}

// syncBuffer est un tampon utilisable par la goroutine du Shipper et par le test en même temps.
type syncBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Len()
}

func (b *syncBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer.Reset()
}

func shipperConfig() config.Events {
	return config.Events{PollInterval: 10 * time.Millisecond, BatchSize: 2, Heartbeat: time.Second}
}

func TestShipperLogsEachEventOnceAndRemembersWhereItStopped(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	var logs syncBuffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	shipper := events.NewShipper(f.store, f.db, logger, shipperConfig())

	command := events.Command{ID: ids.New()}
	created := startAt.Add(-2 * time.Second)
	command.ClientCreatedAt = &created
	f.append(t, events.WithCommand(ctx, command), declared(ids.New()))
	f.append(t, ctx, declared(ids.New()))
	f.append(t, ctx, declared(ids.New()))

	// Lots de deux : trois événements demandent deux passages.
	for pass, want := range []int{2, 1, 0} {
		got, err := shipper.ShipOnce(ctx)
		if err != nil || got != want {
			t.Fatalf("passage %d : %d événements journalisés (%v), attendu %d", pass+1, got, err, want)
		}
	}

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d lignes de log, attendu 3 : %s", len(lines), logs.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first["msg"] != "événement" || first["event_type"] != events.IncidentDeclared || first["event_seq"] != float64(1) ||
		first["command_id"] != command.ID.String() || first["sync_delay_ms"] != float64(2000) {
		t.Errorf("première ligne = %v", first)
	}
	if payload, ok := first["payload"].(map[string]any); !ok || payload["typeId"] != float64(3) {
		t.Errorf("le contenu doit être un objet JSON imbriqué : %v", first["payload"])
	}

	// Un événement écrit plus tard est repris, et seulement lui.
	f.append(t, ctx, declared(ids.New()))
	logs.Reset()
	if got, err := shipper.ShipOnce(ctx); err != nil || got != 1 || strings.Count(logs.String(), "\n") != 1 {
		t.Errorf("nouvel événement : %d journalisé(s) (%v), %q", got, err, logs.String())
	}
}

func TestOnlyOneShipperWritesAtATime(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.append(t, ctx, declared(ids.New()))
	var logs syncBuffer
	shipper := events.NewShipper(f.store, f.db, slog.New(slog.NewJSONHandler(&logs, nil)), shipperConfig())

	// Une autre instance journalise : elle tient le verrou de journalisation le temps de sa transaction.
	holding, release := make(chan struct{}), make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = f.tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := database.Conn(ctx, f.db).Exec(`SELECT pg_advisory_xact_lock(?)`, int64(7_000_003)).Error; err != nil {
				t.Error(err)
			}
			close(holding)
			<-release
			return nil
		})
	}()
	<-holding
	got, err := shipper.ShipOnce(ctx)
	close(release)
	<-done
	if err != nil || got != 0 || logs.Len() != 0 {
		t.Errorf("pendant qu'une autre instance journalise : %d événement(s), %v, %q : celle-ci doit passer son tour", got, err, logs.String())
	}

	if got, err := shipper.ShipOnce(ctx); err != nil || got != 1 {
		t.Errorf("une fois le verrou libre : %d événement(s), %v", got, err)
	}
}

func TestShipperRunStopsWhenTheContextIsCancelled(t *testing.T) {
	f := newFixture(t)
	f.append(t, context.Background(), declared(ids.New()))
	var logs syncBuffer
	shipper := events.NewShipper(f.store, f.db, slog.New(slog.NewJSONHandler(&logs, nil)), shipperConfig())
	ctx, cancel := context.WithCancel(context.Background())

	stopped := make(chan struct{})
	go func() {
		shipper.Run(ctx)
		close(stopped)
	}()
	deadline := time.After(3 * time.Second)
	for !strings.Contains(logs.String(), "événement") {
		select {
		case <-deadline:
			t.Fatal("l'événement n'a pas été journalisé")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Run ne s'arrête pas à l'annulation du contexte")
	}
}
