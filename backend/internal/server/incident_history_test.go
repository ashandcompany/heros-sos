package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type incidentSummaryView struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	TypeLabel  string  `json:"typeLabel"`
	Status     string  `json:"status"`
	City       string  `json:"city"`
	ResolvedAt *string `json:"resolvedAt"`
	Lead       *struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
	} `json:"lead"`
	Villains []struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
	} `json:"villains"`
}

type incidentSummaryPage struct {
	Items []incidentSummaryView `json:"items"`
	Total int                   `json:"total"`
}

func TestPublicIncidentHistoryOfHeroesAndVillains(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	reporter := e.signUpAndLogin("reporter@example.org")
	lead := e.approvedHero(moderator, "lead@example.org", "Lead")
	member := e.approvedHero(moderator, "member@example.org", "Member")
	idle := e.approvedHero(moderator, "idle@example.org", "Idle")
	typeID := e.createType(moderator, "theft", "Vol", false).ID

	created := moderator.post("/villains", map[string]any{"alias": "Docteur Chaos"})
	var villain struct{ ID string }
	created.decode(t, &villain)
	if created.status != http.StatusCreated {
		t.Fatalf("création du vilain : %d %s", created.status, created.body)
	}
	declare := func(title string) string {
		body := validIncident(typeID)
		body["title"], body["villainIds"] = title, []string{villain.ID}
		got := reporter.post("/incidents", body)
		var incident struct{ ID string }
		got.decode(t, &incident)
		if got.status != http.StatusCreated {
			t.Fatalf("déclaration : %d %s", got.status, got.body)
		}
		if approved := moderator.request(http.MethodPut, "/admin/incidents/"+incident.ID+"/moderation", map[string]string{
			"decision": "approved",
		}); approved.status != http.StatusNoContent {
			t.Fatalf("approbation : %d %s", approved.status, approved.body)
		}
		return incident.ID
	}

	// Un incident résolu par Lead et Member, puis un incident encore ouvert.
	resolvedID := declare("Braquage résolu")
	started := lead.post("/incidents/"+resolvedID+"/interventions", nil)
	var intervention struct{ ID string }
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	for _, step := range []struct {
		who  *actor
		path string
		body any
	}{
		{member, "/join", nil},
		{lead, "/outcome", map[string]string{"status": "succeeded"}},
		{lead, "/outcome/validate", map[string]bool{"approved": true}},
		{member, "/outcome/validate", map[string]bool{"approved": true}},
	} {
		if got := step.who.post("/interventions/"+intervention.ID+step.path, step.body); got.status != http.StatusNoContent {
			t.Fatalf("%s : %d %s", step.path, got.status, got.body)
		}
	}
	openID := declare("Braquage en cours")

	anonymous := e.newActor()
	for _, hero := range []*actor{lead, member} {
		var page incidentSummaryPage
		got := anonymous.get("/heroes/" + hero.userID() + "/incidents")
		got.decode(t, &page)
		if got.status != http.StatusOK || page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("incidents résolus du héros : %d %s", got.status, got.body)
		}
		item := page.Items[0]
		if item.ID != resolvedID || item.Status != "succeeded" || item.ResolvedAt == nil || item.City != "Lyon" ||
			item.TypeLabel != "Vol" || item.Lead == nil || item.Lead.Alias != "Lead" ||
			len(item.Villains) != 1 || item.Villains[0].Alias != "Docteur Chaos" {
			t.Errorf("incident résolu : %+v", item)
		}
		for _, private := range []string{"description", "12 rue des Lilas", "reporter"} {
			if strings.Contains(string(got.body), private) {
				t.Errorf("l'historique public expose %q : %s", private, got.body)
			}
		}
	}

	var idlePage incidentSummaryPage
	idleList := anonymous.get("/heroes/" + idle.userID() + "/incidents")
	idleList.decode(t, &idlePage)
	if idleList.status != http.StatusOK || idlePage.Total != 0 || idlePage.Items == nil {
		t.Errorf("héros sans intervention : %d %s", idleList.status, idleList.body)
	}
	for _, path := range []string{
		"/heroes/" + reporter.userID() + "/incidents", // un citoyen n'a pas de fiche publique
		"/heroes/" + uuid.NewString() + "/incidents",
		"/heroes/inconnu/incidents",
		"/villains/" + uuid.NewString() + "/incidents",
		"/villains/inconnu/incidents",
	} {
		if got := anonymous.get(path); got.status != http.StatusNotFound {
			t.Errorf("%s : %d %s", path, got.status, got.body)
		}
	}

	var villainPage incidentSummaryPage
	got := anonymous.get("/villains/" + villain.ID + "/incidents")
	got.decode(t, &villainPage)
	if got.status != http.StatusOK || villainPage.Total != 2 || len(villainPage.Items) != 2 {
		t.Fatalf("incidents du vilain : %d %s", got.status, got.body)
	}
	open, resolved := villainPage.Items[0], villainPage.Items[1]
	if open.ID != openID || open.Status != "reported" || open.ResolvedAt != nil || open.Lead != nil {
		t.Errorf("incident ouvert, le plus récent d'abord : %+v", open)
	}
	if resolved.ID != resolvedID || resolved.ResolvedAt == nil || resolved.Lead == nil || resolved.Lead.ID != lead.userID() {
		t.Errorf("incident résolu du vilain : %+v", resolved)
	}

	var paged incidentSummaryPage
	pagedList := anonymous.get("/villains/" + villain.ID + "/incidents?limit=1&offset=1")
	pagedList.decode(t, &paged)
	if pagedList.status != http.StatusOK || paged.Total != 2 || len(paged.Items) != 1 || paged.Items[0].ID != resolvedID {
		t.Errorf("pagination : %d %s", pagedList.status, pagedList.body)
	}
}
