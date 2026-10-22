package realtime_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/realtime"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

type world struct {
	db    *gorm.DB
	store *events.Store
	tx    *database.GormTransactor
	hub   *realtime.Hub
	cfg   config.Events
	url   string

	citizen, otherCitizen, hero, moderator uuid.UUID
	incident, otherIncident                uuid.UUID
}

func exec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}

// newWorld démarre le flux sur une vraie base : quatre comptes, deux incidents (chacun déclaré par un citoyen), le
// journal, la lecture continue et un serveur HTTP qui prend l'identité du lecteur dans un en-tête de test.
func newWorld(t *testing.T, tune func(*config.Events)) *world {
	t.Helper()
	db := testutil.NewDB(t)
	w := &world{
		db: db, store: events.NewStore(db, clock.NewFake(time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC))),
		tx: database.NewTransactor(db),
		cfg: config.Events{
			PollInterval: 10 * time.Millisecond, BatchSize: 100, Heartbeat: time.Minute,
			MaxStreamsPerUser: 3, StreamMaxAge: time.Minute, StreamBuffer: 16,
		},
	}
	if tune != nil {
		tune(&w.cfg)
	}
	exec(t, db, `INSERT INTO incident_types (code, label, endangers_life) VALUES ('theft', 'Vol', false)`)
	account := func(role user.Role, email string) uuid.UUID {
		id := ids.New()
		exec(t, db, `INSERT INTO users (id, email, password_hash, role, status, civility, last_name, first_name, birth_date)
			VALUES (?, ?, 'x', ?, 'active', 'non-precise', '\x00', '\x00', '\x00')`, id, email, string(role))
		return id
	}
	w.citizen, w.otherCitizen = account(user.RoleCitizen, "citizen@example.test"), account(user.RoleCitizen, "other@example.test")
	w.hero, w.moderator = account(user.RoleHero, "hero@example.test"), account(user.RoleModerator, "mod@example.test")
	declare := func(reporter uuid.UUID) uuid.UUID {
		id := ids.New()
		exec(t, db, `INSERT INTO incidents (id, reporter_id, type_id, title, description, status) VALUES (?, ?, 1, 'Incident', 'texte', 'reported')`, id, reporter)
		return id
	}
	w.incident, w.otherIncident = declare(w.citizen), declare(w.otherCitizen)

	w.hub = realtime.NewHub(w.cfg.StreamBuffer, w.cfg.MaxStreamsPerUser)
	tailer := realtime.NewTailer(w.store, db, w.hub, discardLogger(), w.cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		tailer.Run(ctx)
		close(done)
	}()
	// Les événements écrits par le test ne sont diffusés que s'ils suivent le point noté par la lecture continue.
	select {
	case <-tailer.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("la lecture continue ne démarre pas")
	}
	t.Cleanup(func() {
		cancel()
		<-done
	})

	handler := realtime.NewHandler(w.hub, w.store, db, discardLogger(), w.cfg, config.HTTP{WriteTimeout: 5 * time.Second})
	roles := map[string]user.Role{
		w.citizen.String(): user.RoleCitizen, w.otherCitizen.String(): user.RoleCitizen,
		w.hero.String(): user.RoleHero, w.moderator.String(): user.RoleModerator,
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Test-User")
		userID, _ := uuid.Parse(id)
		principal := auth.Principal{UserID: userID, Role: roles[id], MFAVerified: true}
		handler.Stream(rw, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
	}))
	t.Cleanup(server.Close)
	w.url = server.URL
	return w
}

// append écrit un événement dans sa propre transaction et retourne son numéro.
func (w *world) append(t *testing.T, event events.Event) int64 {
	t.Helper()
	var appended events.Appended
	err := w.tx.WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		appended, err = w.store.Append(ctx, event)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return appended.Seq
}

func (w *world) declared(incident, actor uuid.UUID) events.Event {
	return events.Event{
		Type: events.IncidentDeclared, AggregateType: events.AggregateIncident, AggregateID: incident, IncidentID: incident,
		ActorID: &actor, Payload: map[string]any{"typeId": 1},
	}
}

func (w *world) started(incident, actor uuid.UUID) events.Event {
	return events.Event{
		Type: events.InterventionStarted, AggregateType: events.AggregateIntervention, AggregateID: ids.New(), IncidentID: incident,
		ActorID: &actor, Payload: map[string]any{"status": "in_progress"},
	}
}

// message est un message SSE reçu.
type message struct {
	ID, Event string
	Data      map[string]any
}

// stream est un flux ouvert par un lecteur.
type stream struct {
	response *http.Response
	messages chan message
	comments chan string
	closed   chan struct{}
	cancel   context.CancelFunc
}

func (w *world) connect(t *testing.T, viewer uuid.UUID, lastEventID string) *stream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, w.url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Test-User", viewer.String())
	if lastEventID != "" {
		request.Header.Set("Last-Event-ID", lastEventID)
	}
	response, err := http.DefaultClient.Do(request) //nolint:bodyclose // fermé par t.Cleanup ci-dessous
	if err != nil {
		t.Fatal(err)
	}
	s := &stream{response: response, messages: make(chan message, 100), comments: make(chan string, 100), closed: make(chan struct{}), cancel: cancel}
	go func() {
		defer close(s.closed)
		scanner := bufio.NewScanner(response.Body)
		var current message
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, ":"):
				s.comments <- strings.TrimSpace(strings.TrimPrefix(line, ":"))
			case strings.HasPrefix(line, "id: "):
				current.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				current.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &current.Data)
			case line == "" && current.ID != "":
				s.messages <- current
				current = message{}
			}
		}
	}()
	t.Cleanup(func() {
		cancel()
		_ = response.Body.Close()
	})
	return s
}

func (s *stream) next(t *testing.T) message {
	t.Helper()
	select {
	case m := <-s.messages:
		return m
	case <-time.After(3 * time.Second):
		t.Fatal("aucun message reçu")
		return message{}
	}
}

func (s *stream) expectNothing(t *testing.T) {
	t.Helper()
	select {
	case m := <-s.messages:
		t.Fatalf("message inattendu : %+v", m)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAHeroReceivesEventsLiveInOrderWithoutTheReportersIdentity(t *testing.T) {
	w := newWorld(t, nil)
	s := w.connect(t, w.hero, "")
	if got := s.response.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q", got)
	}

	first := w.append(t, w.declared(w.incident, w.citizen))
	second := w.append(t, w.started(w.incident, w.hero))

	declared, started := s.next(t), s.next(t)
	if declared.ID != strconv.FormatInt(first, 10) || declared.Event != events.IncidentDeclared || started.ID != strconv.FormatInt(second, 10) || started.Event != events.InterventionStarted {
		t.Fatalf("messages : %+v puis %+v", declared, started)
	}
	if _, leaked := declared.Data["actorId"]; leaked {
		t.Errorf("l'identité du déclarant ne doit pas être révélée à un héros : %v", declared.Data)
	}
	if started.Data["actorId"] != w.hero.String() || started.Data["incidentId"] != w.incident.String() || started.Data["version"] != float64(1) {
		t.Errorf("données de l'événement d'intervention : %v", started.Data)
	}
}

func TestCitizensOnlyReceiveTheEventsOfTheirOwnIncidents(t *testing.T) {
	w := newWorld(t, nil)
	owner, stranger, moderator := w.connect(t, w.citizen, ""), w.connect(t, w.otherCitizen, ""), w.connect(t, w.moderator, "")

	w.append(t, w.declared(w.incident, w.citizen))
	w.append(t, w.started(w.incident, w.hero))

	// L'auteur voit sa déclaration, avec son identité ; le modérateur aussi.
	for name, viewer := range map[string]*stream{"auteur": owner, "modérateur": moderator} {
		declared, started := viewer.next(t), viewer.next(t)
		if declared.Event != events.IncidentDeclared || declared.Data["actorId"] != w.citizen.String() || started.Event != events.InterventionStarted {
			t.Errorf("%s : %+v puis %+v", name, declared, started)
		}
	}
	// Un autre citoyen ne reçoit rien de l'incident d'autrui.
	stranger.expectNothing(t)

	w.append(t, w.declared(w.otherIncident, w.otherCitizen))
	if got := stranger.next(t); got.Data["incidentId"] != w.otherIncident.String() {
		t.Errorf("l'autre citoyen doit recevoir son propre incident : %+v", got)
	}
	owner.expectNothing(t)
}

func TestReconnectionReplaysWhatWasMissedThenGoesLive(t *testing.T) {
	w := newWorld(t, nil)
	first := w.append(t, w.declared(w.incident, w.citizen))
	second := w.append(t, w.started(w.incident, w.hero))
	third := w.append(t, w.started(w.incident, w.hero))

	// Le navigateur avait reçu le premier événement : il se reconnecte avec son numéro.
	s := w.connect(t, w.hero, strconv.FormatInt(first, 10))
	for _, want := range []int64{second, third} {
		if got := s.next(t); got.ID != strconv.FormatInt(want, 10) {
			t.Errorf("rattrapage : reçu %s, attendu %d", got.ID, want)
		}
	}
	live := w.append(t, w.started(w.incident, w.hero))
	if got := s.next(t); got.ID != strconv.FormatInt(live, 10) {
		t.Errorf("en direct : reçu %s, attendu %d", got.ID, live)
	}
	s.expectNothing(t) // ni doublon entre le rattrapage et le direct
}

func TestWithoutLastEventIDOnlyNewEventsArrive(t *testing.T) {
	w := newWorld(t, nil)
	w.append(t, w.declared(w.incident, w.citizen))

	s := w.connect(t, w.hero, "")
	s.expectNothing(t)
	fresh := w.append(t, w.started(w.incident, w.hero))
	if got := s.next(t); got.ID != strconv.FormatInt(fresh, 10) {
		t.Errorf("reçu %s, attendu %d", got.ID, fresh)
	}
}

func TestAStreamHasAMaximumAgeSoTheSessionIsRecheckedOnReconnection(t *testing.T) {
	w := newWorld(t, func(c *config.Events) { c.StreamMaxAge = 150 * time.Millisecond })
	s := w.connect(t, w.hero, "")

	select {
	case <-s.closed:
	case <-time.After(3 * time.Second):
		t.Fatal("le flux doit se fermer à son âge maximal")
	}
}

func TestAnIdleStreamSendsHeartbeats(t *testing.T) {
	w := newWorld(t, func(c *config.Events) { c.Heartbeat = 50 * time.Millisecond })
	s := w.connect(t, w.hero, "")

	deadline := time.After(3 * time.Second)
	for {
		select {
		case comment := <-s.comments:
			if strings.Contains(comment, "signal de vie") {
				return
			}
		case <-deadline:
			t.Fatal("aucun signal de vie reçu")
		}
	}
}

func TestAUserCannotOpenMoreStreamsThanAllowed(t *testing.T) {
	w := newWorld(t, func(c *config.Events) { c.MaxStreamsPerUser = 1 })
	w.connect(t, w.hero, "")

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, w.url, http.NoBody)
	request.Header.Set("X-Test-User", w.hero.String())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusTooManyRequests {
		t.Errorf("second flux : statut %d, attendu 429", response.StatusCode)
	}
}

func TestClosingTheHubEndsOpenStreams(t *testing.T) {
	w := newWorld(t, nil)
	s := w.connect(t, w.hero, "")
	s.expectNothing(t)

	w.hub.Close()
	select {
	case <-s.closed:
	case <-time.After(3 * time.Second):
		t.Fatal("l'arrêt du hub doit terminer les flux ouverts")
	}
}
