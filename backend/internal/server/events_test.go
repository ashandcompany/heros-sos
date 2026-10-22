package server_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
)

func mustParseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *e2e) journal() *events.Store { return events.NewStore(e.db, e.clock) }

func (e *e2e) typesOfJournal() []string {
	e.t.Helper()
	recorded, err := e.journal().Since(context.Background(), 0, 1000)
	if err != nil {
		e.t.Fatal(err)
	}
	types := make([]string, len(recorded))
	for i, event := range recorded {
		types[i] = event.Type
	}
	return types
}

func TestEveryActionOfAnInterventionLeavesOneEvent(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	invitee := e.approvedHero(moderator, "invitee@example.org", "Invitee")
	typeID := e.createType(moderator, "rescue", "Sauvetage", true).ID
	created := reporter.post("/incidents", validIncident(typeID))
	var incident struct{ ID string }
	created.decode(t, &incident)

	var intervention interventionView
	started := lead.post("/incidents/"+incident.ID+"/interventions", nil)
	started.decode(t, &intervention)
	steps := []struct {
		name string
		do   func() response
	}{
		{"un héros rejoint", func() response { return member.post("/interventions/"+intervention.ID+"/join", nil) }},
		{"invitation", func() response {
			return lead.post("/interventions/"+intervention.ID+"/invitations", map[string]any{"heroIds": []string{invitee.userID()}})
		}},
		{"confirmation", func() response {
			return invitee.post("/interventions/"+intervention.ID+"/invitations/me/confirm", nil)
		}},
		{"proposition", func() response {
			return lead.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "succeeded"})
		}},
		{"validation du lead", func() response {
			return lead.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
		{"validation du membre", func() response {
			return member.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
		{"validation de l'invité", func() response {
			return invitee.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
	}
	for _, step := range steps {
		if got := step.do(); got.status != http.StatusNoContent {
			t.Fatalf("%s : %d %s", step.name, got.status, got.body)
		}
	}

	want := []string{
		events.IncidentDeclared, events.InterventionStarted, events.InterventionJoined, events.InterventionInvited,
		events.InterventionInvitationConfirmed, events.InterventionOutcomeProposed, events.InterventionOutcomeValidated,
		events.InterventionOutcomeValidated, events.InterventionClosed,
	}
	if got := e.typesOfJournal(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("journal = %v\nattendu    %v", got, want)
	}

	history, err := e.journal().OfIncident(context.Background(), mustParseUUID(t, incident.ID))
	if err != nil || len(history) != len(want) {
		t.Fatalf("histoire de l'incident : %d événements, %v", len(history), err)
	}
	for i, event := range history {
		if event.Seq != int64(i+1) {
			t.Errorf("événement %d : numéro %d, attendu %d", i, event.Seq, i+1)
		}
		if event.ActorID == nil {
			t.Errorf("%s : pas d'auteur", event.Type)
		}
	}
	// Versions : par agrégat, sans trou. L'incident n'a que sa déclaration ; l'intervention compte le reste.
	if history[0].Version != 1 || history[len(history)-1].Version != len(want)-1 {
		t.Errorf("versions : déclaration %d, dernier événement %d", history[0].Version, history[len(history)-1].Version)
	}
	closed := history[len(history)-1]
	if closed.Payload["status"] != "succeeded" || closed.Payload["incidentId"] != incident.ID || closed.Payload["interventionId"] != intervention.ID {
		t.Errorf("contenu de la clôture : %v", closed.Payload)
	}

	// Le journal et la base racontent la même histoire.
	checked, mismatches, err := events.Verify(context.Background(), e.db, e.journal(), 100)
	if err != nil || checked != 1 || len(mismatches) != 0 {
		t.Errorf("Verify : %d incidents, écarts %v, %v", checked, mismatches, err)
	}
}

func TestARejectedActionLeavesNoEvent(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	incidentID := e.declareIncident(reporter, moderator, "theft")
	if got := lead.post("/incidents/"+incidentID+"/interventions", nil); got.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", got.status, got.body)
	}
	before := len(e.typesOfJournal())

	// Un second démarrage sur le même incident échoue : ni état modifié, ni événement.
	if got := lead.post("/incidents/"+incidentID+"/interventions", nil); got.status == http.StatusCreated {
		t.Fatal("un second démarrage ne doit pas réussir")
	}
	if after := len(e.typesOfJournal()); after != before {
		t.Errorf("%d événements après un refus, %d avant", after, before)
	}
	// Un incident déclaré avec un type inconnu est refusé de la même façon.
	body := validIncident(9999)
	if got := reporter.post("/incidents", body); got.status != http.StatusUnprocessableEntity {
		t.Fatalf("déclaration invalide : %d", got.status)
	}
	if after := len(e.typesOfJournal()); after != before {
		t.Errorf("une déclaration refusée a laissé un événement : %d, attendu %d", after, before)
	}
}

func TestVerifyReportsAStateThatTheJournalDoesNotExplain(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	incidentID := e.declareIncident(reporter, moderator, "theft")

	// Changer un statut sans passer par l'application (donc sans événement) est exactement ce que Verify détecte.
	if err := e.db.Exec(`UPDATE incidents SET status = 'succeeded' WHERE id = ?`, incidentID).Error; err != nil {
		t.Fatal(err)
	}

	checked, mismatches, err := events.Verify(context.Background(), e.db, e.journal(), 100)
	if err != nil || checked != 1 || len(mismatches) != 1 {
		t.Fatalf("Verify : %d incidents, écarts %v, %v", checked, mismatches, err)
	}
	if got := mismatches[0]; got.IncidentID.String() != incidentID || got.Journal != "reported" || got.State != "succeeded" {
		t.Errorf("écart = %+v", got)
	}
}

func TestTheShipperWritesTheJournalToTheLogs(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	e.declareIncident(reporter, moderator, "theft")

	var logs bytes.Buffer
	shipper := events.NewShipper(e.journal(), e.db, slog.New(slog.NewJSONHandler(&logs, nil)),
		config.Events{PollInterval: 1, BatchSize: 100, Heartbeat: 1})
	shipped, err := shipper.ShipOnce(context.Background())
	if err != nil || shipped != 1 {
		t.Fatalf("journalisés : %d, %v", shipped, err)
	}
	for _, want := range []string{`"event_type":"incident.declared"`, `"aggregate_type":"incident"`, `"incident_id"`} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("la ligne de log ne contient pas %s : %s", want, logs.String())
		}
	}
	// Ni description ni adresse : le journal part tel quel vers le monitoring.
	for _, secret := range []string{"rue des Lilas", "forcent la porte"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("le log contient %q : les événements ne portent que des identifiants et des états", secret)
		}
	}
}
