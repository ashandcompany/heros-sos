package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// ---- Changement du mot de passe ---------------------------------------------------------------

func TestChangePasswordClosesOtherSessions(t *testing.T) {
	e := newE2E(t, nil)
	const email, newPassword = "steve@avengers.example", "Nouveau5678xyz"
	laptop := e.signUpAndLogin(email)
	phone := e.newActor()
	if got := phone.login(email, goodPassword); got.status != http.StatusOK {
		t.Fatalf("connexion du second appareil : %d %s", got.status, got.body)
	}

	if got := laptop.post("/me/password", map[string]string{"currentPassword": "Mauvais1234abc", "newPassword": newPassword}); got.status != http.StatusUnprocessableEntity || got.fields()["currentPassword"] != "invalid" {
		t.Fatalf("mot de passe actuel faux : %d %s", got.status, got.body)
	}
	if got := laptop.post("/me/password", map[string]string{"currentPassword": goodPassword, "newPassword": "court"}); got.status != http.StatusUnprocessableEntity || got.fields()["newPassword"] == "" {
		t.Fatalf("nouveau mot de passe trop faible : %d %s", got.status, got.body)
	}

	got := laptop.post("/me/password", map[string]string{"currentPassword": goodPassword, "newPassword": newPassword})
	var session struct{ CSRFToken, MFA string }
	got.decode(t, &session)
	if got.status != http.StatusOK || session.CSRFToken == "" || session.MFA != "not_enrolled" {
		t.Fatalf("changement : %d %s", got.status, got.body)
	}
	laptop.csrf = session.CSRFToken

	if got := laptop.get("/me"); got.status != http.StatusOK {
		t.Errorf("la session courante continue avec son nouveau cookie : %d", got.status)
	}
	if got := laptop.put("/me/newsletter", map[string]bool{"subscribed": true}); got.status != http.StatusNoContent {
		t.Errorf("le nouveau jeton CSRF vaut pour les écritures : %d %s", got.status, got.body)
	}
	if got := phone.get("/me"); got.status != http.StatusUnauthorized {
		t.Errorf("l'autre appareil est déconnecté : %d", got.status)
	}
	if got := e.newActor().login(email, goodPassword); got.status != http.StatusUnauthorized {
		t.Errorf("l'ancien mot de passe ne vaut plus : %d", got.status)
	}
	if got := e.newActor().login(email, newPassword); got.status != http.StatusOK {
		t.Errorf("connexion avec le nouveau mot de passe : %d %s", got.status, got.body)
	}
	if mails := e.mailsTo(email, "a été modifié"); len(mails) != 1 {
		t.Errorf("mails de changement de mot de passe : %d", len(mails))
	}
}

func TestChangePasswordRequiresASession(t *testing.T) {
	e := newE2E(t, nil)
	got := e.newActor().post("/me/password", map[string]string{"currentPassword": goodPassword, "newPassword": "Nouveau5678xyz"})
	if got.status != http.StatusUnauthorized {
		t.Fatalf("sans session : %d", got.status)
	}
}

// ---- Pseudo et description du super-héros -----------------------------------------------------

func TestHeroUpdatesAliasAndBio(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("mod@shield.example")
	hero := e.approvedHero(moderator, "steve@avengers.example", "Captain")
	e.approvedHero(moderator, "tony@avengers.example", "Iron Man")
	citizen := e.signUpAndLogin("peter@avengers.example")

	got := hero.patch("/me", map[string]string{"alias": "  Captain America ", "bio": "Le bouclier."})
	var view accountView
	got.decode(t, &view)
	if got.status != http.StatusOK || view.Hero == nil || view.Hero.Alias != "Captain America" || view.Hero.Bio != "Le bouclier." {
		t.Fatalf("modification : %d %s", got.status, got.body)
	}
	var public struct{ Alias, Power string }
	e.newActor().get("/heroes/"+hero.userID()).decode(t, &public)
	if public.Alias != "Captain America" || public.Power != "Le bouclier." {
		t.Errorf("fiche publique : %+v", public)
	}

	if got := hero.patch("/me", map[string]string{"alias": "iron man"}); got.status != http.StatusConflict || got.code() != "alias_already_used" {
		t.Errorf("pseudo déjà pris (sans tenir compte de la casse) : %d %s", got.status, got.body)
	}
	if got := hero.patch("/me", map[string]string{"alias": "Cap!", "bio": strings.Repeat("a", 1001)}); got.status != http.StatusUnprocessableEntity ||
		got.fields()["alias"] == "" || got.fields()["bio"] == "" {
		t.Errorf("pseudo et description invalides : %d %s", got.status, got.body)
	}
	if got := hero.patch("/me", map[string]string{"bio": ""}); got.status != http.StatusOK {
		t.Errorf("description effacée : %d %s", got.status, got.body)
	} else if got.decode(t, &view); view.Hero.Bio != "" || view.Hero.Alias != "Captain America" {
		t.Errorf("seule la description change : %s", got.body)
	}
	if got := citizen.patch("/me", map[string]string{"alias": "Spider-Man"}); got.status != http.StatusForbidden || got.code() != "hero_profile_required" {
		t.Errorf("un citoyen n'a pas de pseudo : %d %s", got.status, got.body)
	}
}

// ---- Retrait de la photo ----------------------------------------------------------------------

func TestRemoveAvatarDeletesTheFile(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	if got := citizen.raw(http.MethodPut, "/me/avatar", "image/png", pngBytes(t, 32, 32)); got.status != http.StatusOK {
		t.Fatalf("envoi : %d %s", got.status, got.body)
	}
	if files := avatarFiles(t, e); len(files) != 1 {
		t.Fatalf("fichiers avant le retrait : %v", files)
	}

	if got := citizen.request(http.MethodDelete, "/me/avatar", nil); got.status != http.StatusNoContent {
		t.Fatalf("retrait : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.AvatarURL != "" {
		t.Errorf("avatarUrl après le retrait : %q", view.AvatarURL)
	}
	if files := avatarFiles(t, e); len(files) != 0 {
		t.Errorf("fichiers après le retrait : %v", files)
	}
	if got := citizen.request(http.MethodDelete, "/me/avatar", nil); got.status != http.StatusNoContent {
		t.Errorf("retrait sans photo : %d %s", got.status, got.body)
	}
}
