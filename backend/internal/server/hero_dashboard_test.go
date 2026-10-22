package server_test

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

type dashboardIncidentView struct {
	ID             string
	Status         string
	InterventionID string
}

type dashboardView struct {
	Global struct {
		Interventions   int64
		IncidentsByType []struct {
			Type  struct{ Code, Label string }
			Count int64
		}
		RescuedCitizens    int64
		VillainsIdentified int64
		VillainsImprisoned int64
	}
	Personal struct {
		Interventions, SuccessfulInterventions, FailedInterventions int64
		AverageRating                                               float64
		RatingsCount, Videos, PublishedVideos                       int64
		MainEnemies                                                 []struct {
			Alias      string
			Encounters int64
		}
	}
	MyOngoingIncidents []dashboardIncidentView
	RecentIncidents    []dashboardIncidentView
}

func (e *e2e) dashboardOf(hero *actor) dashboardView {
	e.t.Helper()
	got := hero.get("/hero/dashboard")
	if got.status != http.StatusOK {
		e.t.Fatalf("tableau de bord : %d %s", got.status, got.body)
	}
	var dashboard dashboardView
	got.decode(e.t, &dashboard)
	return dashboard
}

func (e *e2e) createVillain(creator *actor, alias string) string {
	e.t.Helper()
	got := creator.post("/villains", map[string]any{"alias": alias, "typeIds": []int16{}})
	var created struct{ ID string }
	got.decode(e.t, &created)
	if got.status != http.StatusCreated {
		e.t.Fatalf("création du vilain %s : %d %s", alias, got.status, got.body)
	}
	return created.ID
}

// submitIncident déclare un incident d'un type existant, lié aux vilains donnés : il reste en attente de modération.
func (e *e2e) submitIncident(reporter *actor, typeID int16, villainIDs ...string) string {
	e.t.Helper()
	body := validIncident(typeID)
	body["villainIds"] = append([]string{}, villainIDs...)
	got := reporter.post("/incidents", body)
	var created struct{ ID string }
	got.decode(e.t, &created)
	if got.status != http.StatusCreated {
		e.t.Fatalf("déclaration d'incident : %d %s", got.status, got.body)
	}
	return created.ID
}

// declareIncidentOfType déclare un incident puis le fait approuver par la modération.
func (e *e2e) declareIncidentOfType(reporter, moderator *actor, typeID int16, villainIDs ...string) string {
	e.t.Helper()
	id := e.submitIncident(reporter, typeID, villainIDs...)
	e.moderateIncident(moderator, id, "approved")
	return id
}

func (e *e2e) moderateIncident(moderator *actor, incidentID, decision string) {
	e.t.Helper()
	if got := moderator.request(http.MethodPut, "/admin/incidents/"+incidentID+"/moderation",
		map[string]string{"decision": decision}); got.status != http.StatusNoContent {
		e.t.Fatalf("modération de l'incident (%s) : %d %s", decision, got.status, got.body)
	}
}

func (e *e2e) startIntervention(hero *actor, incidentID string) interventionView {
	e.t.Helper()
	got := hero.post("/incidents/"+incidentID+"/interventions", nil)
	var started interventionView
	got.decode(e.t, &started)
	if got.status != http.StatusCreated {
		e.t.Fatalf("démarrage de l'intervention : %d %s", got.status, got.body)
	}
	return started
}

func (e *e2e) mustPost(a *actor, path string, body any, want int) {
	e.t.Helper()
	if got := a.post(path, body); got.status != want {
		e.t.Fatalf("POST %s : %d %s", path, got.status, got.body)
	}
}

func TestHeroDashboardRequiresAHero(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")
	if got := e.newActor().get("/hero/dashboard"); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d, attendu 401", got.status)
	}
	if got := citizen.get("/hero/dashboard"); got.status != http.StatusForbidden {
		t.Errorf("citoyen : %d, attendu 403", got.status)
	}
	if got := moderator.get("/hero/dashboard"); got.status != http.StatusForbidden {
		t.Errorf("modérateur : %d, attendu 403", got.status)
	}
}

func TestHeroDashboardAggregatesPlatformAndHeroStatistics(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("fury@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lumen")
	member := e.approvedHero(moderator, "member@example.org", "Rempart")
	outsider := e.approvedHero(moderator, "outsider@example.org", "Sentinelle")

	theft := e.createType(moderator, "theft", "Vol", false)
	burglary := e.createType(moderator, "burglary", "Cambriolage", false)
	fire := e.createType(moderator, "fire", "Incendie", false)
	e.createType(moderator, "flood", "Inondation", false)

	raven := e.createVillain(reporter, "Le Corbeau Gris")
	joker := e.createVillain(reporter, "Le Farceur")
	e.createVillain(reporter, "L'Ombre")
	if got := moderator.request(http.MethodPatch, "/villains/"+raven, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{}, "status": "emprisonne",
	}); got.status != http.StatusOK {
		t.Fatalf("emprisonnement du vilain : %d %s", got.status, got.body)
	}

	// Les incidents sont déclarés à des minutes différentes : « les derniers » sont alors sans ambiguïté.
	declare := func(typeID int16, villainIDs ...string) string {
		e.clock.Advance(time.Minute)
		return e.declareIncidentOfType(reporter, moderator, typeID, villainIDs...)
	}
	succeededID := declare(theft.ID, raven)
	failedID := declare(theft.ID, raven)
	ongoingID := declare(burglary.ID, joker)
	reportedID := declare(fire.ID)
	// Deux incidents que les héros ne voient pas, les plus récents : l'un n'a jamais été approuvé, l'autre a été
	// réussi puis rejeté par la modération. Ni leurs types, ni leurs citoyens, ni leur intervention ne comptent.
	e.clock.Advance(time.Minute)
	pendingID := e.submitIncident(reporter, theft.ID, raven)
	rejectedID := declare(theft.ID, raven)

	// Réussie à deux : le héros de l'équipe compte autant que le chef.
	team := e.startIntervention(lead, succeededID)
	e.mustPost(member, "/interventions/"+team.ID+"/join", nil, http.StatusNoContent)
	e.mustPost(lead, "/interventions/"+team.ID+"/outcome", map[string]string{"status": "succeeded"}, http.StatusNoContent)
	e.mustPost(lead, "/interventions/"+team.ID+"/outcome/validate", map[string]bool{"approved": true}, http.StatusNoContent)
	e.mustPost(member, "/interventions/"+team.ID+"/outcome/validate", map[string]bool{"approved": true}, http.StatusNoContent)
	// Échouée en solo : close directement.
	solo := e.startIntervention(lead, failedID)
	e.mustPost(lead, "/interventions/"+solo.ID+"/outcome", map[string]string{"status": "failed"}, http.StatusNoContent)
	excluded := e.startIntervention(lead, rejectedID)
	e.mustPost(lead, "/interventions/"+excluded.ID+"/outcome", map[string]string{"status": "succeeded"}, http.StatusNoContent)
	e.moderateIncident(moderator, rejectedID, "rejected")
	// En cours : le chef intervient, l'invité n'a encore rien confirmé.
	current := e.startIntervention(lead, ongoingID)
	e.mustPost(lead, "/interventions/"+current.ID+"/invitations", map[string]any{"heroIds": []string{outsider.userID()}}, http.StatusNoContent)

	// Seuls les incidents approuvés et réussis comptent comme citoyens secourus.
	for id, rescued := range map[string]int{succeededID: 4, failedID: 9, ongoingID: 5, reportedID: 7, pendingID: 30, rejectedID: 20} {
		if err := e.db.Exec("UPDATE incidents SET citizens_rescued = ? WHERE id = ?", rescued, id).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Une note seule est publiée tout de suite; une note commentée attend la modération et n'entre pas encore
	// dans la moyenne.
	for _, review := range []struct {
		interventionID, comment string
		rating                  int
	}{{team.ID, "", 5}, {solo.ID, "Merci", 3}} {
		e.mustPost(reporter, "/interventions/"+review.interventionID+"/reviews",
			map[string]any{"heroId": lead.userID(), "rating": review.rating, "comment": review.comment}, http.StatusNoContent)
	}
	// Deux films : seul celui que la modération a approuvé est publié.
	for _, status := range []string{"approved", "pending"} {
		filmID := status + "-film"
		if err := e.db.Exec(`INSERT INTO films (id, video_url, title, created_at) VALUES (gen_random_uuid(), ?, ?, now())`,
			"https://example.org/"+filmID, filmID).Error; err != nil {
			t.Fatal(err)
		}
		if err := e.db.Exec(`INSERT INTO hero_films (hero_id, film_id, status, created_at)
			SELECT ?, id, ?, now() FROM films WHERE title = ?`, lead.userID(), status, filmID).Error; err != nil {
			t.Fatal(err)
		}
	}

	t.Run("chef d'équipe", func(t *testing.T) {
		dashboard := e.dashboardOf(lead)
		global := dashboard.Global
		if global.Interventions != 2 || global.RescuedCitizens != 4 || global.VillainsIdentified != 3 || global.VillainsImprisoned != 1 {
			t.Errorf("statistiques globales : %+v", global)
		}
		var byType []string
		var counts []int64
		for _, item := range global.IncidentsByType {
			byType, counts = append(byType, item.Type.Label), append(counts, item.Count)
		}
		// Du plus fréquent au moins fréquent, à égalité par libellé; un type sans incident y figure.
		if want := []string{"Vol", "Cambriolage", "Incendie", "Inondation"}; !reflect.DeepEqual(byType, want) ||
			!reflect.DeepEqual(counts, []int64{2, 1, 1, 0}) {
			t.Errorf("incidents par type : %v %v", byType, counts)
		}
		personal := dashboard.Personal
		if personal.Interventions != 2 || personal.SuccessfulInterventions != 1 || personal.FailedInterventions != 1 ||
			personal.AverageRating != 5 || personal.RatingsCount != 1 || personal.Videos != 2 || personal.PublishedVideos != 1 {
			t.Errorf("statistiques du héros : %+v", personal)
		}
		// Le Corbeau Gris deux fois (une réussite, un échec), Le Farceur une fois (intervention en cours).
		if enemies := personal.MainEnemies; len(enemies) != 2 || enemies[0].Alias != "Le Corbeau Gris" || enemies[0].Encounters != 2 ||
			enemies[1].Alias != "Le Farceur" || enemies[1].Encounters != 1 {
			t.Errorf("principaux ennemis : %+v", enemies)
		}
		if ongoing := dashboard.MyOngoingIncidents; len(ongoing) != 1 || ongoing[0].ID != ongoingID ||
			ongoing[0].Status != "in_progress" || ongoing[0].InterventionID != current.ID {
			t.Errorf("incidents en cours : %+v", ongoing)
		}
		// Le plus récent d'abord; l'incident réussi n'y est pas, l'échoué si (il peut être repris).
		if recent := dashboard.RecentIncidents; len(recent) != 3 || recent[0].ID != reportedID ||
			recent[1].ID != ongoingID || recent[2].ID != failedID {
			t.Errorf("derniers incidents non résolus : %+v", recent)
		}
	})

	t.Run("membre d'équipe", func(t *testing.T) {
		dashboard := e.dashboardOf(member)
		personal := dashboard.Personal
		if personal.Interventions != 1 || personal.SuccessfulInterventions != 1 || personal.FailedInterventions != 0 ||
			personal.RatingsCount != 0 || personal.AverageRating != 0 || personal.Videos != 0 {
			t.Errorf("statistiques du membre : %+v", personal)
		}
		if len(personal.MainEnemies) != 1 || personal.MainEnemies[0].Alias != "Le Corbeau Gris" || personal.MainEnemies[0].Encounters != 1 {
			t.Errorf("principaux ennemis du membre : %+v", personal.MainEnemies)
		}
		if len(dashboard.MyOngoingIncidents) != 0 {
			t.Errorf("le membre n'intervient plus nulle part : %+v", dashboard.MyOngoingIncidents)
		}
		if dashboard.Global.Interventions != 2 {
			t.Errorf("les statistiques globales sont les mêmes pour tous les héros : %+v", dashboard.Global)
		}
	})

	t.Run("héros invité sans participation confirmée", func(t *testing.T) {
		dashboard := e.dashboardOf(outsider)
		personal := dashboard.Personal
		if personal.Interventions != 0 || personal.RatingsCount != 0 || personal.AverageRating != 0 || personal.Videos != 0 ||
			len(personal.MainEnemies) != 0 || len(dashboard.MyOngoingIncidents) != 0 {
			t.Errorf("une invitation n'est pas une participation : %+v %+v", personal, dashboard.MyOngoingIncidents)
		}
		if len(dashboard.RecentIncidents) != 3 {
			t.Errorf("les derniers incidents sont communs à tous : %+v", dashboard.RecentIncidents)
		}
	})
}

// Les statistiques du tableau de bord (agrégats sur de grosses tables) sont mises en cache : figées pendant la
// fraîcheur demandée, puis recalculées en arrière-plan sans jamais faire attendre l'affichage. Les deux listes
// d'incidents, lues dans des index, restent à jour.
func TestHeroDashboardStatisticsAreCachedAndRefreshedInTheBackground(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_DASHBOARD_STATS_TTL": "30s"})
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("fury@shield.example")
	hero := e.approvedHero(moderator, "hero@example.org", "Lumen")
	theft := e.createType(moderator, "theft", "Vol", false)
	incidentID := e.declareIncidentOfType(reporter, moderator, theft.ID)

	before := e.dashboardOf(hero)
	if before.Personal.SuccessfulInterventions != 0 || before.Global.Interventions != 0 || len(before.RecentIncidents) != 1 {
		t.Fatalf("départ : %+v", before)
	}

	started := e.startIntervention(hero, incidentID)
	e.mustPost(hero, "/interventions/"+started.ID+"/outcome", map[string]string{"status": "succeeded"}, http.StatusNoContent)

	// Dans le délai : les listes voient déjà l'intervention terminée, les statistiques sont celles du premier calcul.
	cached := e.dashboardOf(hero)
	if len(cached.RecentIncidents) != 0 || len(cached.MyOngoingIncidents) != 0 {
		t.Errorf("les listes doivent être à jour : %+v", cached)
	}
	if cached.Personal.SuccessfulInterventions != 0 || cached.Global.Interventions != 0 {
		t.Errorf("les statistiques sont servies depuis le cache pendant sa fraîcheur : %+v %+v", cached.Personal, cached.Global)
	}

	// Passé le délai : la valeur périmée est encore servie, et une nouvelle se calcule en arrière-plan.
	e.clock.Advance(31 * time.Second)
	if stale := e.dashboardOf(hero); stale.Personal.SuccessfulInterventions != 0 {
		t.Errorf("une valeur périmée est servie sans attendre le recalcul : %+v", stale.Personal)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		fresh := e.dashboardOf(hero)
		if fresh.Personal.SuccessfulInterventions == 1 && fresh.Global.Interventions == 1 && fresh.Global.RescuedCitizens == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("les statistiques ne se rafraîchissent pas : %+v %+v", fresh.Personal, fresh.Global)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
