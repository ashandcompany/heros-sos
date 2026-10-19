package server_test

import (
	"net/http"
	"strings"
	"testing"
)

type typeView struct {
	ID, Label string
	Active    bool
}

type villainView struct {
	ID, Alias, ThreatLevel string
}

func (e *e2e) firstActiveType(a *actor) typeView {
	e.t.Helper()
	var page struct{ Items []typeView }
	got := a.get("/intervention-types")
	got.decode(e.t, &page)
	if got.status != http.StatusOK || len(page.Items) == 0 {
		e.t.Fatalf("types actifs : %d %s", got.status, got.body)
	}
	return page.Items[0]
}

func validIncident(typeID string) map[string]any {
	return map[string]any{
		"interventionTypeId": typeID,
		"title":              "Cambriolage en cours",
		"description":        "Deux individus forcent la porte de la bijouterie.",
		"reporterRole":       "temoin",
		"location":           map[string]any{"address": "12 rue des Lilas", "city": "Lyon"},
		"occurredAt":         nil,
		"villainId":          nil,
	}
}

func TestInterventionTypesAreManagedByModerators(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")

	if got := e.newActor().get("/intervention-types"); got.status != http.StatusUnauthorized {
		t.Errorf("anonyme : %d", got.status)
	}
	if got := citizen.post("/admin/intervention-types", map[string]string{"label": "Évasion"}); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne crée pas de type : %d", got.status)
	}
	if got := citizen.get("/admin/intervention-types"); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne liste pas tout : %d", got.status)
	}

	created := moderator.post("/admin/intervention-types", map[string]string{"label": "Évasion", "description": "Fuite d'un prisonnier"})
	var view typeView
	created.decode(t, &view)
	if created.status != http.StatusCreated || !view.Active {
		t.Fatalf("création : %d %s", created.status, created.body)
	}
	if got := moderator.post("/admin/intervention-types", map[string]string{"label": "évasion"}); got.status != http.StatusConflict ||
		got.code() != "intervention_type_label_already_used" {
		t.Errorf("doublon (casse ignorée) : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/intervention-types", map[string]string{"label": " "}); got.status != http.StatusUnprocessableEntity ||
		got.fields()["label"] != "required" {
		t.Errorf("libellé vide : %d %s", got.status, got.body)
	}


	// Désactivé : disparu du choix des utilisateurs, toujours visible des modérateurs.
	for range 2 { // idempotent
		if got := moderator.post("/admin/intervention-types/"+view.ID+"/deactivate", nil); got.status != http.StatusNoContent {
			t.Fatalf("désactivation : %d %s", got.status, got.body)
		}
	}
	if e.count("audit_logs", "action = 'intervention_type.deactivate'") != 1 {
		t.Error("une seule trace d'audit pour une désactivation répétée")
	}
	var active, all struct{ Items []typeView }
	citizen.get("/intervention-types").decode(t, &active)
	moderator.get("/admin/intervention-types").decode(t, &all)
	if len(all.Items) != len(active.Items)+1 {
		t.Errorf("actifs = %d, tous = %d", len(active.Items), len(all.Items))
	}
	for _, item := range active.Items {
		if item.ID == view.ID {
			t.Error("un type désactivé ne doit pas être proposé")
		}
	}
	if got := moderator.post("/admin/intervention-types/00000000-0000-0000-0000-000000000000/deactivate", nil); got.status != http.StatusNotFound {
		t.Errorf("type inconnu : %d", got.status)
	}
}

func TestVillainsAreCreatedByUsers(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")

	created := citizen.post("/villains", map[string]string{"alias": "Le Corbeau Gris", "description": "Voleur de nuit"})
	var villain villainView
	created.decode(t, &villain)
	if created.status != http.StatusCreated || villain.ThreatLevel != "faible" {
		t.Fatalf("création : %d %s", created.status, created.body)
	}
	if got := moderator.post("/villains", map[string]string{"alias": "le corbeau gris"}); got.status != http.StatusConflict ||
		got.code() != "villain_alias_already_used" {
		t.Errorf("doublon (casse ignorée) : %d %s", got.status, got.body)
	}
	if got := citizen.post("/villains", map[string]string{"alias": "x"}); got.status != http.StatusUnprocessableEntity || got.fields()["alias"] != "invalid" {
		t.Errorf("pseudo invalide : %d %s", got.status, got.body)
	}
	moderator.post("/villains", map[string]string{"alias": "Docteur Chaos"})

	var page struct {
		Items []villainView
		Total int
	}
	citizen.get("/villains?q=le%20co").decode(t, &page)
	if page.Total != 1 || page.Items[0].Alias != "Le Corbeau Gris" {
		t.Errorf("recherche par préfixe : %+v", page)
	}
	if got := citizen.get("/villains?q=%25"); got.status != http.StatusOK {
		t.Errorf("les jokers LIKE sont neutralisés : %d", got.status)
	} else {
		got.decode(t, &page)
		if page.Total != 0 {
			t.Errorf("« %% » ne doit rien trouver : %+v", page)
		}
	}
}

func TestIncidentDeclaration(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")
	incidentType := e.firstActiveType(citizen)

	var villain villainView
	citizen.post("/villains", map[string]string{"alias": "Le Corbeau Gris"}).decode(t, &villain)

	if got := e.newActor().post("/incidents", validIncident(incidentType.ID)); got.status != http.StatusUnauthorized && got.status != http.StatusForbidden {
		t.Errorf("anonyme : %d", got.status)
	}

	body := validIncident(incidentType.ID)
	body["villainId"] = villain.ID
	body["occurredAt"] = "2026-10-18T22:30:00+02:00"
	before := len(e.mailer.Messages())
	created := citizen.post("/incidents", body)
	var incident struct {
		ID, Status string
		Villain    *struct{ ID, Alias string }
		Location   struct{ Address, City string }
	}
	created.decode(t, &incident)
	if created.status != http.StatusCreated || incident.Status != "signale" || incident.Villain == nil ||
		incident.Villain.Alias != "Le Corbeau Gris" || incident.Location.City != "Lyon" {
		t.Fatalf("déclaration : %d %s", created.status, created.body)
	}

	// Le lieu est chiffré en base.
	var stored struct{ Address []byte }
	if err := e.db.Table("incidents").Select("address").Where("id = ?", incident.ID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored.Address) == 0 || strings.Contains(string(stored.Address), "Lilas") {
		t.Error("l'adresse doit être chiffrée en base")
	}

	// Seuls les modérateurs sont prévenus (aucun héros ici), jamais le déclarant, sans le lieu.
	sent := e.mailer.Messages()[before:]
	if len(sent) != 1 || sent[0].To != "fury@shield.example" || strings.Contains(sent[0].Text, "Lilas") {
		t.Errorf("mails envoyés : %+v", sent)
	}
	if e.count("audit_logs", "action = 'incident.declare'") != 1 {
		t.Error("la déclaration doit être journalisée")
	}

	// Un héros ou un modérateur peut aussi déclarer.
	if got := moderator.post("/incidents", validIncident(incidentType.ID)); got.status != http.StatusCreated {
		t.Errorf("modérateur déclarant : %d %s", got.status, got.body)
	}
}

func TestIncidentDeclarationValidation(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")
	incidentType := e.firstActiveType(citizen)

	bad := map[string]any{
		"interventionTypeId": "pas-un-uuid", "title": "ab", "description": "trop court",
		"reporterRole": "passant", "location": map[string]any{"address": "", "city": ""},
		"occurredAt": "2999-01-01T00:00:00Z", "villainId": "00000000-0000-0000-0000-000000000000",
	}
	got := citizen.post("/incidents", bad)
	fields := got.fields()
	want := map[string]string{
		"interventionTypeId": "invalid", "title": "too_short", "description": "too_short",
		"reporterRole": "invalid", "location.address": "required", "location.city": "required",
		"occurredAt": "invalid", "villainId": "invalid",
	}
	if got.status != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", got.status, got.body)
	}
	for field, code := range want {
		if fields[field] != code {
			t.Errorf("%s = %q, attendu %q", field, fields[field], code)
		}
	}

	// Un type désactivé n'est plus acceptable.
	moderator.post("/admin/intervention-types/"+incidentType.ID+"/deactivate", nil)
	if got := citizen.post("/incidents", validIncident(incidentType.ID)); got.status != http.StatusUnprocessableEntity ||
		got.fields()["interventionTypeId"] != "invalid" {
		t.Errorf("type désactivé : %d %s", got.status, got.body)
	}
	if e.count("incidents", "true") != 0 {
		t.Error("aucun incident ne doit être créé sur une saisie invalide")
	}
}

func TestIncidentDeclarationIsRateLimitedPerUser(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_INCIDENT_CREATE_PER_USER": "2/1h"})
	citizen := e.signUpAndLogin("citizen@example.org")
	other := e.signUpAndLogin("other@example.org")
	typeID := e.firstActiveType(citizen).ID

	for range 2 {
		if got := citizen.post("/incidents", validIncident(typeID)); got.status != http.StatusCreated {
			t.Fatalf("%d %s", got.status, got.body)
		}
	}
	if got := citizen.post("/incidents", validIncident(typeID)); got.status != http.StatusTooManyRequests || got.code() != "rate_limited" {
		t.Errorf("la troisième déclaration doit être limitée : %d %s", got.status, got.body)
	}
	if got := other.post("/incidents", validIncident(typeID)); got.status != http.StatusCreated {
		t.Errorf("la limite est par utilisateur : %d", got.status)
	}
}
