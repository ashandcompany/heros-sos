package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type publicHeroView struct {
	ID                 string  `json:"id"`
	Alias              string  `json:"alias"`
	AvatarURL          *string `json:"avatarUrl"`
	Power              string  `json:"power"`
	AverageRating      float64 `json:"averageRating"`
	InterventionsCount int     `json:"interventionsCount"`
}

func TestPublicHeroListingAndDetail(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	reporter := e.signUpAndLogin("reporter@example.org")
	alpha := e.approvedHero(moderator, "alpha@example.org", "Alpha")
	beta := e.approvedHero(moderator, "beta@example.org", "Beta")
	inactive := e.approvedHero(moderator, "inactive@example.org", "Inactive")
	pending := e.signUpAndLogin("pending@example.org")
	if got := pending.post("/heroes/requests", map[string]string{"alias": "Pending"}); got.status != http.StatusCreated {
		t.Fatalf("demande en attente : %d %s", got.status, got.body)
	}

	for _, update := range []struct {
		heroID, bio, avatarPath string
	}{
		{alpha.userID(), "Décrit Alpha", "/avatars/alpha.png"},
		{beta.userID(), "Décrit Beta", `C:\private\beta.png`},
	} {
		if err := e.db.Exec("UPDATE hero_profiles SET bio = ? WHERE user_id = ?", update.bio, update.heroID).Error; err != nil {
			t.Fatal(err)
		}
		if err := e.db.Exec("UPDATE users SET avatar_path = ? WHERE id = ?", update.avatarPath, update.heroID).Error; err != nil {
			t.Fatal(err)
		}
	}
	// La session d'un compte désactivé est refusée : on garde son identifiant avant de le désactiver.
	inactiveID := inactive.userID()
	pendingID := pending.userID()
	if err := e.db.Exec("UPDATE users SET status = 'deactivated' WHERE id = ?", inactiveID).Error; err != nil {
		t.Fatal(err)
	}

	incidentID := e.declareIncident(reporter, moderator, "public-hero")
	started := alpha.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention struct{ ID string }
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage de l'intervention : %d %s", started.status, started.body)
	}
	if got := beta.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion de Beta : %d %s", got.status, got.body)
	}
	for _, review := range []struct {
		heroID string
		rating int
	}{
		{alpha.userID(), 4},
		{beta.userID(), 5},
	} {
		if err := e.db.Exec(
			"INSERT INTO hero_reviews (id, intervention_id, hero_id, rating, status) VALUES (?, ?, ?, ?, 'published')",
			uuid.New(), intervention.ID, review.heroID, review.rating,
		).Error; err != nil {
			t.Fatal(err)
		}
	}

	anonymous := e.newActor()
	var aliasSorted struct {
		Items  []publicHeroView `json:"items"`
		Total  int              `json:"total"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
	}
	listed := anonymous.get("/heroes")
	listed.decode(t, &aliasSorted)
	if listed.status != http.StatusOK || aliasSorted.Total != 2 || len(aliasSorted.Items) != 2 ||
		aliasSorted.Items[0].Alias != "Alpha" || aliasSorted.Items[1].Alias != "Beta" {
		t.Fatalf("liste publique triée par pseudo : %d %+v %s", listed.status, aliasSorted, listed.body)
	}
	if aliasSorted.Items[0].Power != "Décrit Alpha" || aliasSorted.Items[0].AverageRating != 4 ||
		aliasSorted.Items[0].InterventionsCount != 1 || aliasSorted.Items[0].AvatarURL == nil ||
		*aliasSorted.Items[0].AvatarURL != "/avatars/alpha.png" {
		t.Errorf("profil public Alpha : %+v", aliasSorted.Items[0])
	}
	if aliasSorted.Items[1].AvatarURL != nil {
		t.Errorf("un chemin local ne doit pas devenir une URL publique : %+v", aliasSorted.Items[1])
	}
	for _, privateField := range []string{"alpha@example.org", "firstName", "lastName", "avatarPath"} {
		if strings.Contains(string(listed.body), privateField) {
			t.Errorf("la liste publique expose %q : %s", privateField, listed.body)
		}
	}

	var ratingSorted struct {
		Items []publicHeroView `json:"items"`
	}
	ratingList := anonymous.get("/heroes?sort=-averageRating")
	ratingList.decode(t, &ratingSorted)
	if ratingList.status != http.StatusOK || len(ratingSorted.Items) != 2 ||
		ratingSorted.Items[0].Alias != "Beta" || ratingSorted.Items[0].AverageRating != 5 {
		t.Fatalf("liste triée par note : %d %+v %s", ratingList.status, ratingSorted, ratingList.body)
	}

	var page struct {
		Items  []publicHeroView `json:"items"`
		Total  int              `json:"total"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
	}
	paged := anonymous.get("/heroes?limit=1&offset=1")
	paged.decode(t, &page)
	if paged.status != http.StatusOK || len(page.Items) != 1 || page.Items[0].Alias != "Beta" ||
		page.Total != 2 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("pagination : %d %+v %s", paged.status, page, paged.body)
	}

	var detail publicHeroView
	gotDetail := anonymous.get("/heroes/" + alpha.userID())
	gotDetail.decode(t, &detail)
	if gotDetail.status != http.StatusOK || detail.ID != alpha.userID() || detail.Alias != "Alpha" ||
		detail.Power != "Décrit Alpha" || detail.AverageRating != 4 || detail.InterventionsCount != 1 {
		t.Fatalf("fiche publique : %d %+v %s", gotDetail.status, detail, gotDetail.body)
	}
	if strings.Contains(string(gotDetail.body), "alpha@example.org") || strings.Contains(string(gotDetail.body), "avatarPath") {
		t.Errorf("la fiche publique expose des données privées : %s", gotDetail.body)
	}
	if err := e.db.Exec("UPDATE hero_reviews SET status = 'hidden' WHERE hero_id = ?", beta.userID()).Error; err != nil {
		t.Fatal(err)
	}
	var betaDetail publicHeroView
	hiddenReviewDetail := anonymous.get("/heroes/" + beta.userID())
	hiddenReviewDetail.decode(t, &betaDetail)
	if hiddenReviewDetail.status != http.StatusOK || betaDetail.AverageRating != 0 || betaDetail.InterventionsCount != 1 {
		t.Errorf("l'avis masqué ne doit pas compter dans la moyenne publique : %d %+v %s",
			hiddenReviewDetail.status, betaDetail, hiddenReviewDetail.body)
	}

	for _, heroID := range []string{pendingID, inactiveID} {
		if got := anonymous.get("/heroes/" + heroID); got.status != http.StatusNotFound {
			t.Errorf("profil non public %s : %d %s", heroID, got.status, got.body)
		}
	}
	if got := anonymous.get("/heroes/not-a-uuid"); got.status != http.StatusNotFound {
		t.Errorf("identifiant invalide : %d %s", got.status, got.body)
	}
	if got := anonymous.get("/heroes?sort=unknown"); got.status != http.StatusUnprocessableEntity || got.fields()["sort"] != "invalid" {
		t.Errorf("tri invalide : %d %s", got.status, got.body)
	}
}
