package server_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/idempotency"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

func withKey(key string, extra ...func(*http.Request)) []func(*http.Request) {
	return append([]func(*http.Request){func(r *http.Request) { r.Header.Set(idempotency.KeyHeader, key) }}, extra...)
}

func (a *actor) postWithKey(path string, body any, key string, extra ...func(*http.Request)) response {
	a.e.t.Helper()
	return a.request(http.MethodPost, path, body, withKey(key, extra...)...)
}

func TestReplayedCommandIsAppliedOnceAndAnswersTheSame(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()
	body := validIncident(typeID)

	first := reporter.postWithKey("/incidents", body, key)
	second := reporter.postWithKey("/incidents", body, key)

	if first.status != http.StatusCreated || second.status != http.StatusCreated {
		t.Fatalf("statuts %d et %d, attendu 201 deux fois", first.status, second.status)
	}
	if string(first.body) != string(second.body) {
		t.Errorf("le rejeu doit rendre la même réponse :\n%s\n%s", first.body, second.body)
	}
	if first.header.Get(idempotency.ReplayedHeader) != "" || second.header.Get(idempotency.ReplayedHeader) != "true" {
		t.Errorf("en-tête de rejeu : premier %q, second %q", first.header.Get(idempotency.ReplayedHeader), second.header.Get(idempotency.ReplayedHeader))
	}
	if got := e.count("incidents", "true"); got != 1 {
		t.Errorf("%d incidents créés, attendu 1", got)
	}
	if got := len(e.typesOfJournal()); got != 1 {
		t.Errorf("%d événements, attendu 1 : un rejeu n'écrit rien", got)
	}
	testutil.AssertReadCounters(t, e.db)
}

func TestACommandKeepsItsKeyAndClientClockInTheJournal(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()
	created := e.clock.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339Nano)

	got := reporter.postWithKey("/incidents", validIncident(typeID), key, func(r *http.Request) {
		r.Header.Set(idempotency.ClientCreatedAtHeader, created)
	})
	if got.status != http.StatusCreated {
		t.Fatalf("déclaration : %d %s", got.status, got.body)
	}

	recorded, err := e.journal().Since(context.Background(), 0, 10)
	if err != nil || len(recorded) != 1 {
		t.Fatalf("journal : %v, %v", recorded, err)
	}
	event := recorded[0]
	if event.CommandID == nil || event.CommandID.String() != key {
		t.Errorf("commande de l'événement = %v, attendu %s", event.CommandID, key)
	}
	if delay, ok := event.SyncDelay(); !ok || delay != 10*time.Minute {
		t.Errorf("délai de synchronisation = %v (%v), attendu 10 minutes", delay, ok)
	}
}

func TestAKeyCannotBeReusedForAnotherRequest(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()
	if got := reporter.postWithKey("/incidents", validIncident(typeID), key); got.status != http.StatusCreated {
		t.Fatalf("première déclaration : %d %s", got.status, got.body)
	}

	other := validIncident(typeID)
	other["title"] = "Un tout autre incident"
	got := reporter.postWithKey("/incidents", other, key)
	if got.status != http.StatusUnprocessableEntity || got.code() != idempotency.CodeKeyReused {
		t.Errorf("même clé, autre corps : %d %s, attendu un 422 %s", got.status, got.body, idempotency.CodeKeyReused)
	}
	if count := e.count("incidents", "true"); count != 1 {
		t.Errorf("%d incidents, attendu 1", count)
	}
}

func TestAnInvalidKeyIsRefused(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID

	for name, key := range map[string]string{"pas un UUID": "abc", "UUID nul": uuid.Nil.String()} {
		if got := reporter.postWithKey("/incidents", validIncident(typeID), key); got.status != http.StatusBadRequest {
			t.Errorf("%s : %d %s, attendu 400", name, got.status, got.body)
		}
	}
	bad := reporter.postWithKey("/incidents", validIncident(typeID), uuid.NewString(), func(r *http.Request) {
		r.Header.Set(idempotency.ClientCreatedAtHeader, "hier")
	})
	if bad.status != http.StatusBadRequest {
		t.Errorf("date illisible : %d %s, attendu 400", bad.status, bad.body)
	}
	if count := e.count("incidents", "true"); count != 0 {
		t.Errorf("%d incidents créés malgré les refus", count)
	}
}

func TestAKeyBelongsToItsAuthor(t *testing.T) {
	e := newE2E(t, nil)
	alice := e.signUpAndLogin("alice@example.org")
	bob := e.signUpAndLogin("bob@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()

	if got := alice.postWithKey("/incidents", validIncident(typeID), key); got.status != http.StatusCreated {
		t.Fatalf("Alice : %d %s", got.status, got.body)
	}
	// La même clé chez un autre utilisateur est une autre commande : elle ne rejoue ni ne bloque celle d'Alice.
	got := bob.postWithKey("/incidents", validIncident(typeID), key)
	if got.status != http.StatusCreated || got.header.Get(idempotency.ReplayedHeader) != "" {
		t.Errorf("Bob : %d %s (rejoué : %q)", got.status, got.body, got.header.Get(idempotency.ReplayedHeader))
	}
	if count := e.count("incidents", "true"); count != 2 {
		t.Errorf("%d incidents, attendu 2", count)
	}
}

func TestAFailedCommandRollsBackAndCanBeRetriedWithTheSameKey(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()

	invalid := validIncident(typeID)
	invalid["title"] = ""
	if got := reporter.postWithKey("/incidents", invalid, key); got.status != http.StatusUnprocessableEntity {
		t.Fatalf("déclaration invalide : %d %s", got.status, got.body)
	}
	if count := e.count("processed_commands", "true"); count != 0 {
		t.Errorf("un échec ne doit pas conserver la clé (%d lignes)", count)
	}

	// Le client corrige et rejoue avec la même clé : la commande s'exécute.
	if got := reporter.postWithKey("/incidents", validIncident(typeID), key); got.status != http.StatusCreated {
		t.Errorf("rejeu après correction : %d %s", got.status, got.body)
	}
	if count := e.count("incidents", "true"); count != 1 {
		t.Errorf("%d incidents, attendu 1", count)
	}
}

func TestSimultaneousDuplicatesRunTheCommandOnce(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	key := uuid.NewString()
	body := validIncident(typeID)

	const duplicates = 8
	responses := make([]response, duplicates)
	var wg sync.WaitGroup
	for i := range duplicates {
		wg.Go(func() { responses[i] = reporter.postWithKey("/incidents", body, key) })
	}
	wg.Wait()

	for i, got := range responses {
		if got.status != http.StatusCreated || string(got.body) != string(responses[0].body) {
			t.Errorf("réponse %d : %d %s", i, got.status, got.body)
		}
	}
	if count := e.count("incidents", "true"); count != 1 {
		t.Errorf("%d incidents créés par %d requêtes simultanées, attendu 1", count, duplicates)
	}
	if got := len(e.typesOfJournal()); got != 1 {
		t.Errorf("%d événements, attendu 1", got)
	}
}

func TestEveryActionOfAHeroCanBeReplayed(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	incidentID := e.declareIncident(reporter, moderator, "theft")

	startKey, joinKey := uuid.NewString(), uuid.NewString()
	first := lead.postWithKey("/incidents/"+incidentID+"/interventions", nil, startKey)
	again := lead.postWithKey("/incidents/"+incidentID+"/interventions", nil, startKey)
	var intervention interventionView
	first.decode(t, &intervention)
	if first.status != http.StatusCreated || again.status != http.StatusCreated || string(first.body) != string(again.body) {
		t.Fatalf("démarrage rejoué : %d / %d\n%s\n%s", first.status, again.status, first.body, again.body)
	}
	if got := e.count("interventions", "incident_id = ?", incidentID); got != 1 {
		t.Errorf("%d interventions, attendu 1", got)
	}

	// Une action sans contenu (204) se rejoue de la même façon.
	for range 2 {
		if got := member.postWithKey("/interventions/"+intervention.ID+"/join", nil, joinKey); got.status != http.StatusNoContent {
			t.Fatalf("rejoindre : %d %s", got.status, got.body)
		}
	}
	if got := e.count("intervention_heroes", "intervention_id = ?", intervention.ID); got != 2 {
		t.Errorf("%d participants, attendu 2", got)
	}
	if got := len(e.typesOfJournal()); got != 4 {
		t.Errorf("%d événements (déclaration, modération, démarrage, participation), attendu 4", got)
	}
	testutil.AssertReadCounters(t, e.db)
}

func TestTheClientCanChooseTheIncidentIdentifier(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	other := e.signUpAndLogin("other@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	chosen := uuid.NewString()

	body := validIncident(typeID)
	body["id"] = chosen
	got := reporter.post("/incidents", body)
	var created struct{ ID string }
	got.decode(t, &created)
	if got.status != http.StatusCreated || created.ID != chosen {
		t.Fatalf("déclaration : %d %s, attendu l'identifiant %s", got.status, got.body, chosen)
	}
	if found := reporter.get("/incidents/" + chosen); found.status != http.StatusOK {
		t.Errorf("l'incident est introuvable sous l'identifiant choisi : %d", found.status)
	}

	// Un identifiant déjà pris est un conflit, y compris pour un autre déclarant.
	if clash := other.post("/incidents", body); clash.status != http.StatusConflict || clash.code() != "incident_already_exists" {
		t.Errorf("identifiant déjà pris : %d %s", clash.status, clash.body)
	}
	body["id"] = uuid.Nil.String()
	if nilID := reporter.post("/incidents", body); nilID.status != http.StatusUnprocessableEntity || nilID.fields()["id"] != "invalid" {
		t.Errorf("identifiant nul : %d %s", nilID.status, nilID.body)
	}
	if count := e.count("incidents", "true"); count != 1 {
		t.Errorf("%d incidents, attendu 1", count)
	}
}

func TestAnActionOnAStaleVersionIsRefusedAndChangesNothing(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	incidentID := e.declareIncident(reporter, moderator, "theft")

	started := lead.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	var viewed struct{ Version int }
	started.decode(t, &viewed)
	if viewed.Version != 1 {
		t.Fatalf("version à la création = %d, attendu 1 (%s)", viewed.Version, started.body)
	}

	// Pendant que le lead (hors ligne) préparait son action sur la version 1, un membre a rejoint : version 2.
	if got := member.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("rejoindre : %d %s", got.status, got.body)
	}
	var current struct{ Version int }
	reading := lead.get("/interventions/" + intervention.ID)
	reading.decode(t, &current)
	if current.Version != 2 {
		t.Fatalf("version lue = %d, attendu 2 (%s)", current.Version, reading.body)
	}

	eventsBefore := len(e.typesOfJournal())
	stale := lead.post("/interventions/"+intervention.ID+"/outcome", map[string]any{"status": "succeeded", "expectedVersion": 1})
	if stale.status != http.StatusConflict || stale.code() != "intervention_version_conflict" {
		t.Fatalf("version périmée : %d %s, attendu un 409 intervention_version_conflict", stale.status, stale.body)
	}
	if got := len(e.typesOfJournal()); got != eventsBefore {
		t.Errorf("une action refusée a laissé un événement (%d, attendu %d)", got, eventsBefore)
	}
	var status string
	if err := e.db.Raw(`SELECT status FROM interventions WHERE id = ?`, intervention.ID).Scan(&status).Error; err != nil || status != "in_progress" {
		t.Errorf("l'intervention doit rester en cours : %q, %v", status, err)
	}
	var proposed *string
	if err := e.db.Raw(`SELECT proposed_status FROM interventions WHERE id = ?`, intervention.ID).Scan(&proposed).Error; err != nil || proposed != nil {
		t.Errorf("aucun résultat ne doit être proposé : %v, %v", proposed, err)
	}

	// La version à jour est acceptée.
	fresh := lead.post("/interventions/"+intervention.ID+"/outcome", map[string]any{"status": "succeeded", "expectedVersion": current.Version})
	if fresh.status != http.StatusNoContent {
		t.Errorf("version à jour : %d %s", fresh.status, fresh.body)
	}
}

func TestOldCommandsArePurged(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	if got := reporter.postWithKey("/incidents", validIncident(typeID), uuid.NewString()); got.status != http.StatusCreated {
		t.Fatalf("déclaration : %d %s", got.status, got.body)
	}

	middleware := idempotency.New(idempotency.Deps{DB: e.db, Clock: e.clock})
	if purged, err := middleware.PurgeOlderThan(context.Background(), time.Hour); err != nil || purged != 0 {
		t.Fatalf("purge d'une commande récente : %d, %v", purged, err)
	}
	e.clock.Advance(2 * time.Hour)
	if purged, err := middleware.PurgeOlderThan(context.Background(), time.Hour); err != nil || purged != 1 {
		t.Errorf("purge d'une commande ancienne : %d, %v, attendu 1", purged, err)
	}
}
