package server_test

import (
	"net/http"
	"strings"
	"testing"
)

type typeView struct {
	ID            int16
	Code          string
	Label         string
	EndangersLife bool
}

type villainView struct {
	ID, Alias, ThreatLevel, Status string
	TypeIDs                        []int16
	Specialties                    []struct {
		ID          int16
		Code, Label string
	}
	ReportedIncidentsCount int64
}

func (e *e2e) createType(a *actor, code, label string, endangersLife bool) typeView {
	e.t.Helper()
	got := a.post("/admin/intervention-types", map[string]any{
		"code": code, "label": label, "endangersLife": endangersLife,
	})
	var created typeView
	got.decode(e.t, &created)
	if got.status != http.StatusCreated || created.ID <= 0 {
		e.t.Fatalf("création du type : %d %s", got.status, got.body)
	}
	return created
}

func validIncident(typeID int16) map[string]any {
	return map[string]any{
		"typeId":      typeID,
		"description": "Deux individus forcent la porte de la bijouterie.",
		"location": map[string]any{
			"address": "12 rue des Lilas", "postalCode": "69001", "city": "Lyon", "country": "FR",
		},
		"occurredAt":  nil,
		"villainIds":  []string{},
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

	view := e.createType(moderator, "hostage_taking", "Prise d'otages", true)
	var types struct{ Items []typeView }
	if got := citizen.get("/intervention-types"); got.status != http.StatusOK {
		t.Fatalf("liste des types : %d %s", got.status, got.body)
	} else {
		got.decode(t, &types)
	}
	if len(types.Items) != 1 || types.Items[0] != view {
		t.Errorf("type listé : %+v, attendu %+v", types.Items, view)
	}
	if got := moderator.post("/admin/intervention-types", map[string]string{"code": "hostage_taking", "label": "Autre libellé"}); got.status != http.StatusConflict ||
		got.code() != "incident_type_code_already_used" {
		t.Errorf("code dupliqué : %d %s", got.status, got.body)
	}
}

func TestVillainsAreCreatedByUsers(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	other := e.signUpAndLogin("other@example.org")
	moderator := e.newModerator("fury@shield.example")
	theft := e.createType(moderator, "theft", "Vol", false)
	burglary := e.createType(moderator, "burglary", "Cambriolage", false)

	created := citizen.post("/villains", map[string]any{
		"alias": "Le Corbeau Gris", "mainSector": "Lyon", "description": "Voleur de nuit",
		"typeIds": []int16{theft.ID, burglary.ID},
	})
	var villain villainView
	created.decode(t, &villain)
	if created.status != http.StatusCreated || villain.Alias != "Le Corbeau Gris" ||
		len(villain.TypeIDs) != 2 || villain.TypeIDs[0] != theft.ID || villain.TypeIDs[1] != burglary.ID ||
		villain.ThreatLevel != "faible" || villain.Status != "en-liberte" {
		t.Fatalf("création : %d %s", created.status, created.body)
	}
	// Spécialités détaillées, par libellé : « Cambriolage » avant « Vol ».
	if s := villain.Specialties; len(s) != 2 || s[0].ID != burglary.ID || s[0].Label != "Cambriolage" ||
		s[0].Code != "burglary" || s[1].ID != theft.ID || s[1].Label != "Vol" {
		t.Errorf("spécialités détaillées : %+v", s)
	}
	if got := citizen.request(http.MethodPatch, "/villains/"+villain.ID, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{burglary.ID},
	}); got.status != http.StatusOK {
		t.Fatalf("remplacement des spécialités : %d %s", got.status, got.body)
	} else {
		got.decode(t, &villain)
		if len(villain.TypeIDs) != 1 || villain.TypeIDs[0] != burglary.ID {
			t.Errorf("spécialités après remplacement : %v", villain.TypeIDs)
		}
	}
	if got := other.request(http.MethodPatch, "/villains/"+villain.ID, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{theft.ID},
	}); got.status != http.StatusForbidden {
		t.Errorf("un autre utilisateur ne modifie pas les spécialités : %d", got.status)
	}
	if got := citizen.request(http.MethodPatch, "/villains/"+villain.ID, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{burglary.ID}, "threatLevel": "critique",
	}); got.status != http.StatusForbidden {
		t.Errorf("le créateur ne modifie pas le niveau de menace : %d", got.status)
	}
	if got := moderator.request(http.MethodPatch, "/villains/"+villain.ID, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{burglary.ID},
		"threatLevel": "eleve", "status": "emprisonne",
	}); got.status != http.StatusOK {
		t.Fatalf("modification des valeurs de modération : %d %s", got.status, got.body)
	} else {
		got.decode(t, &villain)
		if villain.ThreatLevel != "eleve" || villain.Status != "emprisonne" {
			t.Errorf("valeurs renvoyées : threatLevel=%q status=%q", villain.ThreatLevel, villain.Status)
		}
	}
	if got := moderator.request(http.MethodPatch, "/villains/"+villain.ID, map[string]any{
		"alias": "Le Corbeau Gris", "typeIds": []int16{burglary.ID},
		"threatLevel": "apocalypse", "status": "libre",
	}); got.status != http.StatusUnprocessableEntity || got.fields()["threatLevel"] != "invalid" || got.fields()["status"] != "invalid" {
		t.Errorf("enums invalides : %d %s", got.status, got.body)
	}
	if got := moderator.post("/villains", map[string]string{"alias": "Le Corbeau Gris"}); got.status != http.StatusConflict ||
		got.code() != "villain_alias_already_used" {
		t.Errorf("alias dupliqué : %d %s", got.status, got.body)
	}
	if got := citizen.post("/villains", map[string]string{"alias": " "}); got.status != http.StatusUnprocessableEntity || got.fields()["alias"] != "required" {
		t.Errorf("alias requis : %d %s", got.status, got.body)
	}
	if got := citizen.post("/villains", map[string]any{"alias": "Inconnu", "typeIds": []int16{32000}}); got.status != http.StatusUnprocessableEntity ||
		got.fields()["typeIds"] != "invalid" {
		t.Errorf("spécialité inconnue : %d %s", got.status, got.body)
	}
	if got := moderator.post("/villains", map[string]string{"alias": "Docteur Chaos"}); got.status != http.StatusCreated {
		t.Fatalf("création par modérateur : %d %s", got.status, got.body)
	}

	var page struct {
		Items []villainView
		Total int
	}
	citizen.get("/villains?q=Le%20Co").decode(t, &page)
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

	// Lecture publique (encyclopédie) : liste et fiche sans session.
	anonymous := e.newActor()
	if got := anonymous.get("/villains"); got.status != http.StatusOK {
		t.Errorf("liste anonyme : %d %s", got.status, got.body)
	}
	var found villainView
	if got := anonymous.get("/villains/" + villain.ID); got.status != http.StatusOK {
		t.Errorf("fiche anonyme : %d %s", got.status, got.body)
	} else if got.decode(t, &found); found.ID != villain.ID || found.ThreatLevel != "eleve" {
		t.Errorf("fiche : %+v", found)
	}
	if got := anonymous.get("/villains/00000000-0000-0000-0000-000000000000"); got.status != http.StatusNotFound {
		t.Errorf("vilain inconnu : %d", got.status)
	}
	if got := anonymous.get("/villains/pas-un-uuid"); got.status != http.StatusNotFound {
		t.Errorf("identifiant invalide : %d", got.status)
	}
}

func TestIncidentDeclaration(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")
	incidentType := e.createType(moderator, "burglary", "Cambriolage", false)

	var villain villainView
	citizen.post("/villains", map[string]string{"alias": "Le Corbeau Gris"}).decode(t, &villain)

	if got := e.newActor().post("/incidents", validIncident(incidentType.ID)); got.status != http.StatusUnauthorized && got.status != http.StatusForbidden {
		t.Errorf("anonyme : %d", got.status)
	}

	body := validIncident(incidentType.ID)
	body["villainIds"] = []string{villain.ID}
	body["occurredAt"] = "2026-10-18T22:30:00+02:00"
	before := len(e.mailer.Messages())
	created := citizen.post("/incidents", body)
	var incident struct {
		ID, Status, Priority string
		TypeID               int16
		Location             struct{ Address, City string }
		VillainIDs           []string
	}
	created.decode(t, &incident)
	if created.status != http.StatusCreated || incident.Status != "reported" || incident.Priority != "normal" ||
		incident.TypeID != incidentType.ID || incident.Location.Address != "12 rue des Lilas" || incident.Location.City != "Lyon" ||
		len(incident.VillainIDs) != 1 || incident.VillainIDs[0] != villain.ID {
		t.Fatalf("déclaration : %d %s", created.status, created.body)
	}

	var stored struct {
		Street          string
		Priority        string
		CitizensRescued int16
	}
	if err := e.db.Table("incidents").Select("street, priority, citizens_rescued").Where("id = ?", incident.ID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Street != "12 rue des Lilas" || stored.Priority != "normal" || stored.CitizensRescued != 0 {
		t.Errorf("valeurs documentées non persistées : %+v", stored)
	}

	// Seuls les modérateurs sont prévenus (aucun héros ici), jamais le déclarant, sans le lieu.
	sent := e.mailer.Messages()[before:]
	if len(sent) != 1 || sent[0].To != "fury@shield.example" || strings.Contains(sent[0].Text, "Lilas") {
		t.Errorf("mails envoyés : %+v", sent)
	}
	if e.count("audit_logs", "action = 'incident.declare'") != 1 {
		t.Error("la déclaration doit être journalisée")
	}
	var ownIncidents struct {
		Items []struct{ ID string }
		Total int
	}
	citizen.get("/incidents?reporterId=" + citizen.userID()).decode(t, &ownIncidents)
	if ownIncidents.Total != 1 || ownIncidents.Items[0].ID != incident.ID {
		t.Errorf("liste personnelle filtrée : %+v", ownIncidents)
	}
	other := e.signUpAndLogin("other-reporter@example.org")
	if got := other.get("/incidents?reporterId=" + citizen.userID()); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne filtre pas sur un autre déclarant : %d", got.status)
	}
	var villains struct{ Items []villainView }
	if got := citizen.get("/villains?q=Le%20Corbeau"); got.status != http.StatusOK {
		t.Fatalf("liste des vilains : %d %s", got.status, got.body)
	} else {
		got.decode(t, &villains)
		if len(villains.Items) != 1 || villains.Items[0].ReportedIncidentsCount != 1 {
			t.Errorf("nombre d'incidents du vilain : %+v", villains.Items)
		}
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
	incidentType := e.createType(moderator, "theft", "Vol", false)

	bad := map[string]any{
		"typeId": 0, "description": "", "citizensRescued": -1,
		"location": map[string]string{"address": "", "city": ""}, "occurredAt": "2999-01-01T00:00:00Z",
		"villainIds": []string{"00000000-0000-0000-0000-000000000000"},
	}
	got := citizen.post("/incidents", bad)
	fields := got.fields()
	want := map[string]string{
		"typeId": "invalid", "description": "required",
		"citizensRescued": "invalid", "location.address": "required", "location.city": "required",
		"occurredAt": "invalid", "villainIds": "invalid",
	}
	if got.status != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", got.status, got.body)
	}
	for field, code := range want {
		if fields[field] != code {
			t.Errorf("%s = %q, attendu %q", field, fields[field], code)
		}
	}

	if got := citizen.post("/incidents", validIncident(incidentType.ID+1000)); got.status != http.StatusUnprocessableEntity ||
		got.fields()["typeId"] != "invalid" {
		t.Errorf("type inconnu : %d %s", got.status, got.body)
	}
	if e.count("incidents", "true") != 0 {
		t.Error("aucun incident ne doit être créé sur une saisie invalide")
	}
}

func TestIncidentDeclarationIsRateLimitedPerUser(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_INCIDENT_CREATE_PER_USER": "2/1h"})
	citizen := e.signUpAndLogin("citizen@example.org")
	other := e.signUpAndLogin("other@example.org")
	moderator := e.newModerator("fury@shield.example")
	typeID := e.createType(moderator, "murder", "Tentative de meurtre", true).ID

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
