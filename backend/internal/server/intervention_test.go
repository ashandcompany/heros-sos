package server_test

import (
	"net/http"
	"testing"
)

type interventionView struct {
	ID, IncidentID, Status string
	Participants           []struct {
		HeroID, Role, Participation, OutcomeValidation string
		IsActive                                       bool
	}
}

func (e *e2e) approvedHero(moderator *actor, email, alias string) *actor {
	e.t.Helper()
	hero := e.signUpAndLogin(email)
	if got := hero.post("/heroes/requests", map[string]string{"alias": alias}); got.status != http.StatusCreated {
		e.t.Fatalf("demande héros %s : %d %s", email, got.status, got.body)
	}
	if got := moderator.post("/admin/hero-requests/"+hero.userID()+"/approve", nil); got.status != http.StatusNoContent {
		e.t.Fatalf("approbation héros %s : %d %s", email, got.status, got.body)
	}
	return hero
}

func (e *e2e) declareIncident(reporter, moderator *actor, code string) string {
	e.t.Helper()
	typeID := e.createType(moderator, code, code, false).ID
	created := reporter.post("/incidents", validIncident(typeID))
	var incident struct{ ID string }
	created.decode(e.t, &incident)
	if created.status != http.StatusCreated {
		e.t.Fatalf("déclaration d'incident : %d %s", created.status, created.body)
	}
	if got := moderator.request(http.MethodPut, "/admin/incidents/"+incident.ID+"/moderation", map[string]string{
		"decision": "approved",
	}); got.status != http.StatusNoContent {
		e.t.Fatalf("approbation de l'incident : %d %s", got.status, got.body)
	}
	return incident.ID
}

func TestInterventionLifecycle(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	invitee := e.approvedHero(moderator, "invitee@example.org", "Invitee")
	outsider := e.approvedHero(moderator, "outsider@example.org", "Outsider")
	incidentID := e.declareIncident(reporter, moderator, "rescue")
	secondIncidentID := e.declareIncident(reporter, moderator, "burglary")
	if got := reporter.post("/incidents/"+incidentID+"/interventions", nil); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne peut pas démarrer une intervention : %d", got.status)
	}
	var page struct {
		Items []struct{ ID string }
		Total int
	}
	if got := lead.get("/incidents"); got.status != http.StatusOK {
		t.Fatalf("liste des incidents : %d %s", got.status, got.body)
	} else {
		got.decode(t, &page)
		if page.Total != 2 {
			t.Fatalf("incidents disponibles : %d", page.Total)
		}
	}

	started := lead.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	if started.status != http.StatusCreated || intervention.Status != "in_progress" || len(intervention.Participants) != 1 ||
		intervention.Participants[0].HeroID != lead.userID() || intervention.Participants[0].Role != "lead" ||
		intervention.Participants[0].Participation != "confirmed" || !intervention.Participants[0].IsActive {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	var incidentDetail struct {
		InterventionID string
	}
	if got := lead.get("/incidents/" + incidentID); got.status != http.StatusOK {
		t.Fatalf("détail de l'incident : %d %s", got.status, got.body)
	} else {
		got.decode(t, &incidentDetail)
		if incidentDetail.InterventionID != intervention.ID {
			t.Errorf("ID de l'intervention active : %q", incidentDetail.InterventionID)
		}
	}
	if got := moderator.post("/admin/interventions/"+intervention.ID+"/decision", map[string]string{"status": "failed"}); got.status != http.StatusConflict || got.code() != "intervention_dispute_not_found" {
		t.Errorf("arbitrage sans désaccord : %d %s", got.status, got.body)
	}
	if got := lead.post("/incidents/"+secondIncidentID+"/interventions", nil); got.status != http.StatusConflict || got.code() != "hero_already_in_intervention" {
		t.Errorf("un héros ne peut pas démarrer deux interventions : %d %s", got.status, got.body)
	}
	if got := member.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion directe : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+intervention.ID+"/invitations", map[string]any{"heroIds": []string{invitee.userID()}}); got.status != http.StatusNoContent {
		t.Fatalf("invitation : %d %s", got.status, got.body)
	}
	if got := invitee.post("/interventions/"+intervention.ID+"/invitations/me/decline", nil); got.status != http.StatusNoContent {
		t.Fatalf("refus : %d %s", got.status, got.body)
	}
	if e.count("intervention_heroes", "intervention_id = ? AND hero_id = ?", intervention.ID, invitee.userID()) != 0 {
		t.Error("le refus doit supprimer la ligne de liaison")
	}
	if got := lead.post("/interventions/"+intervention.ID+"/invitations", map[string]any{"heroIds": []string{invitee.userID()}}); got.status != http.StatusNoContent {
		t.Fatalf("seconde invitation : %d %s", got.status, got.body)
	}
	if got := invitee.post("/interventions/"+intervention.ID+"/invitations/me/confirm", nil); got.status != http.StatusNoContent {
		t.Fatalf("confirmation : %d %s", got.status, got.body)
	}
	if got := lead.get("/interventions/" + intervention.ID); got.status != http.StatusOK {
		t.Fatalf("lecture des participants : %d %s", got.status, got.body)
	} else {
		got.decode(t, &intervention)
		if len(intervention.Participants) != 3 {
			t.Errorf("équipe hydratée : %+v", intervention.Participants)
		}
	}
	if got := outsider.get("/interventions/" + intervention.ID); got.status != http.StatusOK {
		t.Errorf("un héros peut consulter l'équipe avant de rejoindre : %d", got.status)
	}

	if got := lead.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "succeeded", "notes": ""}); got.status != http.StatusNoContent {
		t.Fatalf("proposition : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("vote lead : %d %s", got.status, got.body)
	}
	if got := member.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": false}); got.status != http.StatusNoContent {
		t.Fatalf("désaccord : %d %s", got.status, got.body)
	}
	if e.count("interventions", "id = ? AND proposed_status IS NULL AND status = 'in_progress'", intervention.ID) != 1 {
		t.Error("un refus annule la proposition et laisse l'intervention en cours")
	}
	if len(e.mailer.Messages()) == 0 {
		t.Error("les invitations et le désaccord doivent envoyer des mails")
	}
	if got := moderator.post("/admin/interventions/"+intervention.ID+"/decision", map[string]string{"status": "failed", "notes": "Arbitrage"}); got.status != http.StatusNoContent {
		t.Fatalf("arbitrage : %d %s", got.status, got.body)
	}
	if e.count("interventions", "id = ? AND status = 'failed' AND decided_by IS NOT NULL AND ended_at IS NOT NULL", intervention.ID) != 1 {
		t.Error("l'arbitrage doit trancher et clôturer l'intervention")
	}
	if e.count("incidents", "id = ? AND status = 'failed'", incidentID) != 1 {
		t.Error("le statut de l'incident doit suivre l'intervention")
	}
	if e.count("intervention_heroes", "intervention_id = ? AND is_active = TRUE", intervention.ID) != 0 {
		t.Error("les participants ne doivent plus être actifs après clôture")
	}
	if e.count("audit_logs", "action = 'intervention.outcome_arbitrate' AND entity_id = ?", intervention.ID) != 1 {
		t.Error("l'arbitrage doit être audité")
	}

	soloIncidentID := e.declareIncident(reporter, moderator, "solo_rescue")
	solo := lead.post("/incidents/"+soloIncidentID+"/interventions", nil)
	var soloIntervention interventionView
	solo.decode(t, &soloIntervention)
	if solo.status != http.StatusCreated {
		t.Fatalf("démarrage solo : %d %s", solo.status, solo.body)
	}
	if got := lead.post("/interventions/"+soloIntervention.ID+"/outcome", map[string]string{"status": "succeeded"}); got.status != http.StatusNoContent {
		t.Fatalf("clôture solo : %d %s", got.status, got.body)
	}
	if e.count("incidents", "id = ? AND status = 'succeeded'", soloIncidentID) != 1 {
		t.Error("le héros seul doit clore directement l'incident")
	}

	teamIncidentID := e.declareIncident(reporter, moderator, "team_success")
	team := lead.post("/incidents/"+teamIncidentID+"/interventions", nil)
	var teamIntervention interventionView
	team.decode(t, &teamIntervention)
	if team.status != http.StatusCreated {
		t.Fatalf("démarrage équipe : %d %s", team.status, team.body)
	}
	if got := member.post("/interventions/"+teamIntervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion équipe : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+teamIntervention.ID+"/outcome", map[string]string{"status": "succeeded"}); got.status != http.StatusNoContent {
		t.Fatalf("proposition équipe : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+teamIntervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("premier vote équipe : %d %s", got.status, got.body)
	}
	if e.count("incidents", "id = ? AND status = 'in_progress'", teamIncidentID) != 1 {
		t.Error("un seul vote ne doit pas clôturer l'incident")
	}
	if got := member.post("/interventions/"+teamIntervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("vote final équipe : %d %s", got.status, got.body)
	}
	if e.count("incidents", "id = ? AND status = 'succeeded'", teamIncidentID) != 1 {
		t.Error("l'unanimité doit clôturer l'incident comme réussi")
	}

	ids := func(query string) map[string]bool {
		got := lead.get("/incidents" + query)
		if got.status != http.StatusOK {
			t.Fatalf("liste %q : %d %s", query, got.status, got.body)
		}
		var page struct{ Items []struct{ ID string } }
		got.decode(t, &page)
		found := map[string]bool{}
		for _, item := range page.Items {
			found[item.ID] = true
		}
		return found
	}
	if all := ids(""); len(all) != 4 || !all[soloIncidentID] || !all[teamIncidentID] {
		t.Errorf("la liste doit contenir les incidents résolus : %v", all)
	}
	if got := ids("?status=succeeded"); len(got) != 2 || !got[soloIncidentID] || !got[teamIncidentID] {
		t.Errorf("filtre status=succeeded : %v", got)
	}
	if got := ids("?status=failed&status=reported"); len(got) != 2 || !got[incidentID] || !got[secondIncidentID] {
		t.Errorf("filtre status multiple : %v", got)
	}
	if got := ids("?heroId=" + member.userID()); len(got) != 2 || !got[incidentID] || !got[teamIncidentID] {
		t.Errorf("filtre heroId : %v", got)
	}
	if got := ids("?heroId=" + member.userID() + "&status=succeeded"); len(got) != 1 || !got[teamIncidentID] {
		t.Errorf("filtres combinés : %v", got)
	}
	if got := ids("?heroId=" + outsider.userID()); len(got) != 0 {
		t.Errorf("héros sans intervention : %v", got)
	}
	for _, query := range []string{"?status=inconnu", "?heroId=x", "?villainId=x"} {
		if got := lead.get("/incidents" + query); got.status != http.StatusUnprocessableEntity {
			t.Errorf("%s : %d", query, got.status)
		}
	}
}

func TestIncidentInterventionsHistory(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	incidentID := e.declareIncident(reporter, moderator, "rescue")
	path := "/incidents/" + incidentID + "/interventions"

	var history struct{ Items []interventionView }
	if got := lead.get(path); got.status != http.StatusOK {
		t.Fatalf("historique vide : %d %s", got.status, got.body)
	} else if got.decode(t, &history); len(history.Items) != 0 {
		t.Fatalf("aucune intervention attendue : %s", got.body)
	}

	var failed, retried interventionView
	lead.post(path, nil).decode(t, &failed)
	if got := lead.post("/interventions/"+failed.ID+"/outcome", map[string]string{"status": "failed"}); got.status != http.StatusNoContent {
		t.Fatalf("échec de la première intervention : %d %s", got.status, got.body)
	}
	if got := lead.post(path, nil); got.status != http.StatusCreated {
		t.Fatalf("nouvelle intervention après un échec : %d %s", got.status, got.body)
	} else {
		got.decode(t, &retried)
	}

	for _, viewer := range []*actor{lead, moderator} {
		got := viewer.get(path)
		got.decode(t, &history)
		if got.status != http.StatusOK || len(history.Items) != 2 ||
			history.Items[0].ID != retried.ID || history.Items[0].Status != "in_progress" ||
			history.Items[1].ID != failed.ID || history.Items[1].Status != "failed" {
			t.Fatalf("historique, de la plus récente à la plus ancienne : %d %s", got.status, got.body)
		}
	}
	if got := reporter.get(path); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne lit pas les interventions : %d", got.status)
	}
	if got := lead.get("/incidents/00000000-0000-0000-0000-000000000000/interventions"); got.status != http.StatusNotFound {
		t.Errorf("incident inconnu : %d %s", got.status, got.body)
	}
}
