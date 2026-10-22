package server_test

import (
	"net/http"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

// Les compteurs de lecture (incident_stats, villain_stats, hero_stats) sont tenus à jour par des triggers :
// après chaque étape du cycle de vie d'une intervention, ils doivent égaler un comptage direct des tables.
func TestReadCountersFollowTheWholeLifecycle(t *testing.T) {
	e := newE2E(t, nil)
	check := func(step string) {
		t.Helper()
		if t.Failed() {
			return
		}
		testutil.AssertReadCounters(t, e.db)
		if t.Failed() {
			t.Fatalf("compteurs incohérents après : %s", step)
		}
	}
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	invitee := e.approvedHero(moderator, "invitee@example.org", "Invitee")
	typeID := e.createType(moderator, "rescue", "Sauvetage", true).ID

	villain := moderator.post("/villains", map[string]any{"alias": "Docteur Chaos"})
	var villainBody struct{ ID string }
	villain.decode(t, &villainBody)
	declare := func(title string, withVillain bool) string {
		body := validIncident(typeID)
		body["title"], body["citizensRescued"] = title, 3
		if withVillain {
			body["villainIds"] = []string{villainBody.ID}
		}
		got := reporter.post("/incidents", body)
		var incident struct{ ID string }
		got.decode(t, &incident)
		if got.status != http.StatusCreated {
			t.Fatalf("déclaration : %d %s", got.status, got.body)
		}
		// Un héros n'intervient que sur un incident approuvé par la modération.
		e.moderateIncident(moderator, incident.ID, "approved")
		return incident.ID
	}

	first := declare("Premier", true)
	second := declare("Second", false)
	check("déclarations")
	if got := e.count("incident_stats", "moderation = 'approved' AND status = 'reported' AND type_id = ? AND incidents = 2", typeID); got != 1 {
		t.Errorf("le compteur des incidents signalés doit valoir 2")
	}

	started := lead.post("/incidents/"+first+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	check("démarrage de l'intervention")

	for _, step := range []struct {
		name string
		do   func() response
	}{
		{"un héros rejoint", func() response { return member.post("/interventions/"+intervention.ID+"/join", nil) }},
		{"invitation", func() response {
			return lead.post("/interventions/"+intervention.ID+"/invitations", map[string]any{"heroIds": []string{invitee.userID()}})
		}},
		{"confirmation de l'invitation", func() response {
			return invitee.post("/interventions/"+intervention.ID+"/invitations/me/confirm", nil)
		}},
		{"proposition du résultat", func() response {
			return lead.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "succeeded"})
		}},
		{"validation par le lead", func() response {
			return lead.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
		{"validation par le membre", func() response {
			return member.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
		{"validation par l'invité", func() response {
			return invitee.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true})
		}},
	} {
		if got := step.do(); got.status != http.StatusNoContent {
			t.Fatalf("%s : %d %s", step.name, got.status, got.body)
		}
		check(step.name)
	}

	// Les trois héros ont désormais une intervention résolue, la date de résolution est celle de la clôture.
	for _, hero := range []*actor{lead, member, invitee} {
		if got := e.count("hero_stats", "hero_id = ? AND interventions = 1 AND resolved = 1", hero.userID()); got != 1 {
			t.Errorf("compteur du héros %s : interventions et résolues doivent valoir 1", hero.email)
		}
	}
	if got := e.count("intervention_heroes", "resolved_at IS NOT NULL AND participation = 'confirmed'"); got != 3 {
		t.Errorf("héros avec une date de résolution = %d, attendu 3", got)
	}
	if got := e.count("incident_stats", "status = 'succeeded' AND incidents = 1 AND citizens_rescued = 3"); got != 1 {
		t.Errorf("le compteur des incidents réussis doit valoir 1 pour 3 citoyens secourus")
	}

	// Un échec arbitré par un modérateur : pas de date de résolution, pas de résolution comptée.
	failing := lead.post("/incidents/"+second+"/interventions", nil)
	var secondIntervention interventionView
	failing.decode(t, &secondIntervention)
	if failing.status != http.StatusCreated {
		t.Fatalf("second démarrage : %d %s", failing.status, failing.body)
	}
	check("seconde intervention")
	if got := member.post("/interventions/"+secondIntervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("le membre rejoint la seconde : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+secondIntervention.ID+"/outcome", map[string]string{"status": "failed"}); got.status != http.StatusNoContent {
		t.Fatalf("proposition d'échec : %d %s", got.status, got.body)
	}
	if got := member.post("/interventions/"+secondIntervention.ID+"/outcome/validate", map[string]bool{"approved": false}); got.status != http.StatusNoContent {
		t.Fatalf("refus : %d %s", got.status, got.body)
	}
	check("résultat contesté")
	if got := moderator.post("/admin/interventions/"+secondIntervention.ID+"/decision", map[string]string{"status": "succeeded", "notes": "Arbitrage"}); got.status != http.StatusNoContent {
		t.Fatalf("arbitrage : %d %s", got.status, got.body)
	}
	check("arbitrage en faveur de la réussite")
	if got := e.count("hero_stats", "hero_id = ? AND interventions = 2 AND resolved = 2", lead.userID()); got != 1 {
		t.Errorf("le lead doit avoir deux interventions résolues")
	}
}

func TestTruncatingTablesEmptiesTheirCounters(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	e.declareIncident(reporter, moderator, "theft")
	if got := e.count("incident_stats", "incidents > 0"); got != 1 {
		t.Fatalf("lignes de compteur = %d, attendu 1", got)
	}

	if err := e.db.Exec(`TRUNCATE incidents CASCADE`).Error; err != nil {
		t.Fatal(err)
	}
	if got := e.count("incident_stats", "true"); got != 0 {
		t.Errorf("TRUNCATE incidents laisse %d lignes de compteur", got)
	}
	testutil.AssertReadCounters(t, e.db)
}

// Supprimer un compte met le déclarant de ses incidents à NULL : les compteurs du déclarant suivent, ceux des
// incidents restent (les incidents ne disparaissent pas).
func TestDeletingAReporterKeepsCountersConsistent(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	hero := e.approvedHero(moderator, "hero@example.org", "Hero")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	var firstID string
	for n := range 3 {
		got := reporter.post("/incidents", validIncident(typeID))
		var created struct{ ID string }
		got.decode(t, &created)
		if n == 0 {
			firstID = created.ID
		}
	}
	// Seul le premier est approuvé : les deux autres restent en attente, mais comptent dans « mes incidents ».
	e.moderateIncident(moderator, firstID, "approved")
	if got := hero.post("/incidents/"+firstID+"/interventions", nil); got.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", got.status, got.body)
	}
	testutil.AssertReadCounters(t, e.db)
	if got := e.count("reporter_stats", "reporter_id = ? AND status = 'reported' AND incidents = 2", reporter.userID()); got != 1 {
		t.Fatalf("le déclarant doit avoir 2 incidents signalés")
	}

	if err := e.db.Exec(`DELETE FROM users WHERE id = ?`, reporter.userID()).Error; err != nil {
		t.Fatal(err)
	}
	testutil.AssertReadCounters(t, e.db)
	if got := e.count("incident_stats", "status = 'reported' AND incidents = 2"); got != 1 {
		t.Errorf("les incidents restent après la suppression du compte : le compteur global ne bouge pas")
	}
	if got := e.count("reporter_stats", "incidents <> 0"); got != 0 {
		t.Errorf("%d compteurs de déclarant non nuls pour un compte supprimé", got)
	}
}

// Une décision de modération change ce que le public voit d'un incident : le compteur des vilains, celui des
// résolutions des héros (et leur date), et la ligne de incident_stats qui compte l'incident doivent suivre, dans les
// deux sens. Chaque étape est comparée à un recalcul depuis les tables (qui ne lit pas les copies dénormalisées).
func TestReadCountersFollowModerationDecisions(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	typeID := e.createType(moderator, "rescue", "Sauvetage", true).ID
	villain := moderator.post("/villains", map[string]any{"alias": "Docteur Chaos"})
	var villainBody struct{ ID string }
	villain.decode(t, &villainBody)

	body := validIncident(typeID)
	body["villainIds"] = []string{villainBody.ID}
	created := reporter.post("/incidents", body)
	var incident struct{ ID string }
	created.decode(t, &incident)
	if created.status != http.StatusCreated {
		t.Fatalf("déclaration : %d %s", created.status, created.body)
	}
	villainCount := func() int64 { return e.count("villain_stats", "villain_id = ? AND incidents = 1", villainBody.ID) }
	resolved := func() int64 { return e.count("hero_stats", "hero_id = ? AND resolved = 1", lead.userID()) }
	resolvedAt := func() int64 { return e.count("intervention_heroes", "resolved_at IS NOT NULL") }

	// En attente : l'incident existe pour son auteur et la modération, pas pour les vilains ni les héros.
	testutil.AssertReadCounters(t, e.db)
	if villainCount() != 0 || e.count("incident_stats", "moderation = 'pending' AND status = 'reported' AND incidents = 1") != 1 {
		t.Fatal("un incident en attente ne compte ni pour son vilain, ni dans les statistiques approuvées")
	}

	e.moderateIncident(moderator, incident.ID, "approved")
	testutil.AssertReadCounters(t, e.db)
	if villainCount() != 1 || e.count("incident_stats", "moderation = 'approved' AND status = 'reported' AND incidents = 1") != 1 {
		t.Fatal("l'approbation doit compter l'incident pour son vilain et dans les statistiques approuvées")
	}
	if e.count("incident_stats", "moderation = 'pending' AND incidents <> 0") != 0 {
		t.Fatal("l'incident ne doit plus être compté comme en attente")
	}

	started := lead.post("/incidents/"+incident.ID+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	if got := lead.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "succeeded"}); got.status != http.StatusNoContent {
		t.Fatalf("clôture : %d %s", got.status, got.body)
	}
	testutil.AssertReadCounters(t, e.db)
	if resolved() != 1 || resolvedAt() != 1 {
		t.Fatal("un incident approuvé et réussi compte comme résolu pour son héros")
	}

	// Rejet après coup : l'historique public du héros et du vilain se vide, les copies dénormalisées aussi.
	e.moderateIncident(moderator, incident.ID, "rejected")
	testutil.AssertReadCounters(t, e.db)
	if resolved() != 0 || resolvedAt() != 0 || villainCount() != 0 {
		t.Fatalf("un incident rejeté ne compte plus : résolu=%d dates=%d vilain=%d", resolved(), resolvedAt(), villainCount())
	}
	if e.count("incident_villains", "incident_approved") != 0 {
		t.Error("incident_villains.incident_approved doit suivre le rejet")
	}
	if got := e.count("hero_stats", "hero_id = ? AND interventions = 1", lead.userID()); got != 1 {
		t.Error("le nombre d'interventions du héros ne dépend pas de la modération")
	}

	// Ré-approbation : tout revient.
	e.moderateIncident(moderator, incident.ID, "approved")
	testutil.AssertReadCounters(t, e.db)
	if resolved() != 1 || resolvedAt() != 1 || villainCount() != 1 || e.count("incident_villains", "incident_approved") != 1 {
		t.Fatalf("une ré-approbation doit tout restaurer : résolu=%d dates=%d vilain=%d", resolved(), resolvedAt(), villainCount())
	}
}
