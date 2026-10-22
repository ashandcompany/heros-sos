package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type incidentCursorPage struct {
	Items []struct {
		ID, Title string
	}
	Total      int
	Limit      int
	NextCursor string
}

func (e *e2e) incidentPage(a *actor, query string) incidentCursorPage {
	e.t.Helper()
	got := a.get("/incidents?" + query)
	var page incidentCursorPage
	got.decode(e.t, &page)
	if got.status != http.StatusOK {
		e.t.Fatalf("GET /incidents?%s : %d %s", query, got.status, got.body)
	}
	return page
}

func TestIncidentListIsBrowsedByCursorWithoutGapsOrRepeats(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	declare := func(n int) {
		body := validIncident(typeID)
		body["title"] = fmt.Sprintf("Incident %d", n)
		if got := reporter.post("/incidents", body); got.status != http.StatusCreated {
			t.Fatalf("déclaration %d : %d %s", n, got.status, got.body)
		}
		e.clock.Advance(time.Minute)
	}
	for n := 1; n <= 5; n++ {
		declare(n)
	}

	first := e.incidentPage(reporter, "limit=2")
	if len(first.Items) != 2 || first.Total != 5 || first.NextCursor == "" ||
		first.Items[0].Title != "Incident 5" || first.Items[1].Title != "Incident 4" {
		t.Fatalf("première page = %+v", first)
	}

	// Un incident déclaré entre deux pages ne décale pas la suite : avec un décalage (offset), « Incident 4 »
	// reviendrait en tête de la deuxième page.
	declare(6)
	second := e.incidentPage(reporter, "limit=2&cursor="+url.QueryEscape(first.NextCursor))
	if len(second.Items) != 2 || second.Items[0].Title != "Incident 3" || second.Items[1].Title != "Incident 2" ||
		second.NextCursor == "" {
		t.Fatalf("deuxième page = %+v", second)
	}
	if second.Total != 6 {
		t.Errorf("total = %d : il compte les incidents du filtre, curseur ou non (6 après la sixième déclaration)", second.Total)
	}

	last := e.incidentPage(reporter, "limit=2&cursor="+url.QueryEscape(second.NextCursor))
	if len(last.Items) != 1 || last.Items[0].Title != "Incident 1" || last.NextCursor != "" {
		t.Errorf("dernière page = %+v : un seul incident et pas de page suivante", last)
	}

	// Une liste qui tient dans une page n'a pas de curseur suivant.
	if all := e.incidentPage(reporter, "limit=100"); len(all.Items) != 6 || all.NextCursor != "" {
		t.Errorf("liste complète = %+v", all)
	}
}

func TestIncidentCursorIsValidated(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	typeID := e.createType(moderator, "theft", "Vol", false).ID
	for range 2 {
		if got := reporter.post("/incidents", validIncident(typeID)); got.status != http.StatusCreated {
			t.Fatalf("déclaration : %d %s", got.status, got.body)
		}
	}
	valid := url.QueryEscape(e.incidentPage(reporter, "limit=1").NextCursor)
	if valid == "" {
		t.Fatal("deux incidents et limit=1 : il doit y avoir une page suivante")
	}

	for _, tc := range []struct{ name, query, field string }{
		{"curseur illisible", "cursor=n-importe-quoi", "cursor"},
		{"base64 sans champs", "cursor=YWJj", "cursor"},
		{"curseur et offset ensemble", "cursor=" + valid + "&offset=1", "offset"},
	} {
		got := reporter.get("/incidents?" + tc.query)
		if got.status != http.StatusUnprocessableEntity || got.fields()[tc.field] != "invalid" {
			t.Errorf("%s : %d %s, attendu un 422 sur %q", tc.name, got.status, got.body, tc.field)
		}
	}
}

func TestIncidentTotalsFollowTheStatusFilter(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("mod@shield.example")
	hero := e.approvedHero(moderator, "hero@example.org", "Hero")
	typeID := e.createType(moderator, "rescue", "Sauvetage", true).ID
	var firstID string
	for n := range 3 {
		got := reporter.post("/incidents", validIncident(typeID))
		var created struct{ ID string }
		got.decode(t, &created)
		if n == 0 {
			firstID = created.ID
		}
		e.moderateIncident(moderator, created.ID, "approved")
	}
	if got := hero.post("/incidents/"+firstID+"/interventions", nil); got.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", got.status, got.body)
	}

	// Les totaux viennent des compteurs : ils doivent suivre les changements de statut.
	for query, want := range map[string]int{
		"":                              3,
		"status=reported":               2,
		"status=in_progress":            1,
		"status=reported&status=failed": 2,
		"status=succeeded":              0,
	} {
		if got := e.incidentPage(hero, query+"&limit=1").Total; got != want {
			t.Errorf("total de %q = %d, attendu %d", query, got, want)
		}
	}

	// Un incident en attente de modération : son auteur et un modérateur le comptent, un héros non.
	if got := reporter.post("/incidents", validIncident(typeID)); got.status != http.StatusCreated {
		t.Fatalf("déclaration : %d %s", got.status, got.body)
	}
	for name, want := range map[string]struct {
		viewer *actor
		total  int
	}{
		"héros": {hero, 3}, "auteur": {reporter, 4}, "modérateur": {moderator, 4},
	} {
		if got := e.incidentPage(want.viewer, "&limit=1").Total; got != want.total {
			t.Errorf("total vu par le %s = %d, attendu %d", name, got, want.total)
		}
	}
}
