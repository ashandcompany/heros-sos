package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/herolocation"
)

func (a *actor) shareLocation(lat, lon float64) response {
	return a.put("/hero/location", map[string]any{"latitude": lat, "longitude": lon})
}

func (a *actor) consentToLocation(t *testing.T) {
	t.Helper()
	if got := a.put("/hero/location/consent", nil); got.status != http.StatusNoContent {
		t.Fatalf("consentement : %d %s", got.status, got.body)
	}
}

func (e *e2e) incidentAt(reporter *actor, typeID int16, lat, lon float64) {
	e.t.Helper()
	body := validIncident(typeID)
	body["location"].(map[string]any)["latitude"] = lat
	body["location"].(map[string]any)["longitude"] = lon
	if got := reporter.post("/incidents", body); got.status != http.StatusCreated {
		e.t.Fatalf("déclaration : %d %s", got.status, got.body)
	}
}

func (e *e2e) recipientsSince(before int) map[string]bool {
	recipients := map[string]bool{}
	for _, message := range e.mailer.Messages()[before:] {
		recipients[message.To] = true
	}
	return recipients
}

// nearbyMentionsSince retourne, par destinataire, si son mail d'incident cite la proximité.
func (e *e2e) nearbyMentionsSince(before int) map[string]bool {
	mentions := map[string]bool{}
	for _, message := range e.mailer.Messages()[before:] {
		mentions[message.To] = strings.Contains(message.Text, "à moins de 10 km de votre dernière position") &&
			strings.Contains(message.HTML, "à moins de 10 km")
	}
	return mentions
}

func TestHeroLocationIsSharedReplacedAndCleared(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("citizen@example.org")
	moderator := e.newModerator("fury@shield.example")
	hero := e.approvedHero(moderator, "hero@example.org", "Hero")

	if got := e.newActor().shareLocation(45.7, 4.8); got.status != http.StatusUnauthorized && got.status != http.StatusForbidden {
		t.Errorf("anonyme : %d", got.status)
	}
	if got := citizen.shareLocation(45.7, 4.8); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne partage pas de position : %d", got.status)
	}
	if got := hero.shareLocation(91, 4.8); got.status != http.StatusUnprocessableEntity {
		t.Errorf("latitude invalide : %d %s", got.status, got.body)
	}
	if got := hero.put("/hero/location", map[string]any{"latitude": 45.7}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("longitude manquante : %d %s", got.status, got.body)
	}

	// Sans consentement, aucune position n'est acceptée.
	if view := hero.account(t); view.Hero == nil || view.Hero.LocationConsentAt != "" {
		t.Errorf("le profil ne doit pas annoncer de consentement : %+v", view.Hero)
	}
	if got := hero.shareLocation(45.7, 4.8); got.status != http.StatusConflict || got.code() != "hero_location_consent_required" {
		t.Errorf("sans consentement : %d %s", got.status, got.body)
	}
	if got := citizen.put("/hero/location/consent", nil); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne consent pas : %d", got.status)
	}
	hero.consentToLocation(t)
	var consentedAt string
	e.db.Table("hero_profiles").Select("location_consent_at::text").Where("user_id = ?", hero.userID()).Take(&consentedAt)
	hero.consentToLocation(t) // idempotent : la date d'origine est conservée
	var again string
	e.db.Table("hero_profiles").Select("location_consent_at::text").Where("user_id = ?", hero.userID()).Take(&again)
	if consentedAt == "" || consentedAt != again {
		t.Errorf("date de consentement : %q puis %q", consentedAt, again)
	}
	if view := hero.account(t); view.Hero == nil || view.Hero.LocationConsentAt == "" {
		t.Errorf("le profil doit annoncer le consentement : %+v", view.Hero)
	}

	if got := hero.shareLocation(45.764043, 4.835659); got.status != http.StatusNoContent {
		t.Fatalf("partage : %d %s", got.status, got.body)
	}
	if got := hero.shareLocation(48.8566, 2.3522); got.status != http.StatusNoContent {
		t.Fatalf("mise à jour : %d %s", got.status, got.body)
	}
	var stored []struct{ Latitude, Longitude float64 }
	e.db.Table("hero_locations").Select("latitude, longitude").Find(&stored)
	if len(stored) != 1 || stored[0].Latitude != 48.857 || stored[0].Longitude != 2.352 {
		t.Errorf("la position doit être remplacée et arrondie : %+v", stored)
	}

	if got := hero.request(http.MethodDelete, "/hero/location", nil); got.status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", got.status, got.body)
	}
	if e.count("hero_locations", "true") != 0 {
		t.Error("la position doit être supprimée")
	}

	// Retirer le consentement supprime la position et bloque les envois suivants.
	hero.shareLocation(45.7, 4.8)
	if got := hero.request(http.MethodDelete, "/hero/location/consent", nil); got.status != http.StatusNoContent {
		t.Fatalf("retrait : %d %s", got.status, got.body)
	}
	if e.count("hero_locations", "true") != 0 || e.count("hero_profiles", "location_consent_at IS NOT NULL") != 0 {
		t.Error("le retrait doit supprimer la position et le consentement")
	}
	if view := hero.account(t); view.Hero == nil || view.Hero.LocationConsentAt != "" {
		t.Errorf("le profil ne doit plus annoncer de consentement : %+v", view.Hero)
	}
	if got := hero.shareLocation(45.7, 4.8); got.status != http.StatusConflict {
		t.Errorf("envoi après retrait : %d", got.status)
	}
}

func TestHeroLocationIsRateLimitedPerHero(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_HERO_LOCATION_PER_USER": "2/1h"})
	moderator := e.newModerator("fury@shield.example")
	hero := e.approvedHero(moderator, "hero@example.org", "Hero")
	other := e.approvedHero(moderator, "other@example.org", "Other")
	hero.consentToLocation(t)
	other.consentToLocation(t)

	for range 2 {
		if got := hero.shareLocation(45.7, 4.8); got.status != http.StatusNoContent {
			t.Fatalf("envoi sous la limite : %d %s", got.status, got.body)
		}
	}
	if got := hero.shareLocation(45.7, 4.8); got.status != http.StatusTooManyRequests {
		t.Errorf("envoi au-delà de la limite : %d %s", got.status, got.body)
	}
	// La limite est propre à chaque héros.
	if got := other.shareLocation(45.7, 4.8); got.status != http.StatusNoContent {
		t.Errorf("un autre héros n'est pas limité : %d %s", got.status, got.body)
	}
}

func TestIncidentAlertsNearbyHeroesThenFallsBackToAll(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	moderator := e.newModerator("fury@shield.example")
	near := e.approvedHero(moderator, "near@example.org", "Near")
	far := e.approvedHero(moderator, "far@example.org", "Far")
	e.approvedHero(moderator, "silent@example.org", "Silent")

	near.consentToLocation(t)
	far.consentToLocation(t)
	near.shareLocation(45.77, 4.83) // Lyon
	far.shareLocation(48.85, 2.35)  // Paris
	// Les types sont créés d'avance : les sessions expirent quand l'horloge avance.
	types := []int16{
		e.createType(moderator, "burglary", "burglary", false).ID,
		e.createType(moderator, "fire", "fire", false).ID,
		e.createType(moderator, "flood", "flood", false).ID,
	}

	before := len(e.mailer.Messages())
	e.incidentAt(reporter, types[0], 45.764, 4.835)
	got := e.recipientsSince(before)
	if len(got) != 2 || !got["fury@shield.example"] || !got["near@example.org"] {
		t.Errorf("seuls le modérateur et le héros proche sont prévenus : %v", got)
	}
	// Seul le héros proche est informé qu'il est prévenu pour sa position.
	if mentions := e.nearbyMentionsSince(before); !mentions["near@example.org"] || mentions["fury@shield.example"] {
		t.Errorf("mention de la proximité : %v", mentions)
	}

	// Une position périmée n'est plus prise en compte : repli sur tous les héros.
	e.clock.Advance(31 * time.Minute)
	reporter.login("reporter@example.org", goodPassword)
	before = len(e.mailer.Messages())
	e.incidentAt(reporter, types[1], 45.764, 4.835)
	got = e.recipientsSince(before)
	if len(got) != 4 {
		t.Errorf("repli sur tous les héros : %v", got)
	}
	for to, mentioned := range e.nearbyMentionsSince(before) {
		if mentioned {
			t.Errorf("un repli ne parle pas de proximité : %s", to)
		}
	}

	// Sans coordonnées : tous les héros.
	before = len(e.mailer.Messages())
	reporter.post("/incidents", validIncident(types[2]))
	if got = e.recipientsSince(before); len(got) != 4 {
		t.Errorf("sans coordonnées : %v", got)
	}

	// La purge supprime les positions périmées.
	purged, err := herolocation.NewService(e.db, e.clock, e.cfg.Proximity.LocationTTL, e.cfg.Proximity.AlertRadiusKm).PurgeStale(t.Context())
	if err != nil || purged != 2 || e.count("hero_locations", "true") != 0 {
		t.Errorf("purge : %d, %v", purged, err)
	}
}
