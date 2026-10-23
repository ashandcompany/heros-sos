package server_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestExportThenDeleteOwnAccount(t *testing.T) {
	e := newE2E(t, nil)
	const email = "steve@avengers.example"
	citizen := e.signUpAndLogin(email)
	moderator := e.newModerator("mod@shield.example")
	incidentID := e.declareIncident(citizen, moderator, "rescue")

	got := citizen.get("/me/export")
	var export struct {
		Account   accountView
		Incidents []struct{ ID string }
	}
	got.decode(t, &export)
	if got.status != http.StatusOK || export.Account.Email != email || len(export.Incidents) != 1 ||
		!strings.Contains(got.header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("export : %d %s", got.status, got.body)
	}

	if got := citizen.request(http.MethodDelete, "/me", map[string]string{"password": "Mauvais1234abc"}); got.status != http.StatusUnprocessableEntity || got.fields()["password"] != "invalid" {
		t.Fatalf("mauvais mot de passe : %d %s", got.status, got.body)
	}
	if got := citizen.request(http.MethodDelete, "/me", map[string]string{"password": goodPassword}); got.status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", got.status, got.body)
	}
	if got := citizen.get("/me"); got.status != http.StatusUnauthorized {
		t.Errorf("la session est fermée : %d", got.status)
	}
	if got := e.newActor().login(email, goodPassword); got.status != http.StatusUnauthorized {
		t.Errorf("l'adresse ne permet plus de se connecter : %d", got.status)
	}
	if mails := e.mailsTo(email, "a été supprimé"); len(mails) != 1 {
		t.Errorf("mail de suppression : %d", len(mails))
	}
	if got := moderator.get("/incidents/" + incidentID); got.status != http.StatusOK {
		t.Errorf("l'incident reste : %d %s", got.status, got.body)
	}
}

func TestModeratorAnonymizesAHero(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("mod@shield.example")
	heroID := e.approvedHero(moderator, "steve@avengers.example", "Captain").userID()

	if got := moderator.post("/admin/users/"+moderator.userID()+"/anonymize", nil); got.status != http.StatusConflict || got.code() != "account_not_erasable" {
		t.Errorf("son propre compte : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/users/"+heroID+"/anonymize", nil); got.status != http.StatusNoContent {
		t.Fatalf("anonymisation : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/users/"+heroID+"/anonymize", nil); got.status != http.StatusConflict || got.code() != "account_already_erased" {
		t.Errorf("deux fois : %d %s", got.status, got.body)
	}
	if got := e.newActor().get("/heroes/" + heroID); got.status != http.StatusNotFound {
		t.Errorf("la fiche publique disparaît : %d", got.status)
	}
	moderator.post("/admin/users/"+heroID+"/reactivate", nil)
	if got := e.newActor().get("/heroes/" + heroID); got.status != http.StatusNotFound {
		t.Errorf("un compte anonymisé ne se réactive pas : %d", got.status)
	}
}
