package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type adminHeroView struct {
	ID                 string  `json:"id"`
	Alias              string  `json:"alias"`
	AvatarURL          *string `json:"avatarUrl"`
	Power              string  `json:"power"`
	AverageRating      float64 `json:"averageRating"`
	InterventionsCount int     `json:"interventionsCount"`
	Civility           string  `json:"civility"`
	FirstName          string  `json:"firstName"`
	LastName           string  `json:"lastName"`
	BirthDate          string  `json:"birthDate"`
	Email              string  `json:"email"`
	PendingEmail       string  `json:"pendingEmail"`
	MobilePhone        string  `json:"mobilePhone"`
	Address            struct {
		Street, Complement, PostalCode, City, Country string
	} `json:"address"`
	AccountStatus      string     `json:"accountStatus"`
	CreatedAt          time.Time  `json:"createdAt"`
	LastInterventionAt *time.Time `json:"lastInterventionAt"`
}

type adminHeroPage struct {
	Items  []adminHeroView `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func TestAdminHeroDirectoryShowsWhatOnlyModeratorsMaySee(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	reporter := e.signUpAndLogin("reporter@example.org")
	alpha := e.approvedHero(moderator, "alpha@example.org", "Alpha")
	beta := e.approvedHero(moderator, "beta@example.org", "Beta")
	retired := e.approvedHero(moderator, "retired@example.org", "Retired")
	candidate := e.signUpAndLogin("candidate@example.org")
	if got := candidate.post("/heroes/requests", map[string]string{"alias": "Candidate"}); got.status != http.StatusCreated {
		t.Fatalf("demande en attente : %d %s", got.status, got.body)
	}

	if got := alpha.patch("/me", map[string]any{"mobilePhone": "0611223344", "address": map[string]any{
		"street": "1 Stark Tower", "complement": "Étage 90", "postalCode": "10001", "city": "New York", "country": "US",
	}}); got.status != http.StatusOK {
		t.Fatalf("coordonnées d'Alpha : %d %s", got.status, got.body)
	}
	if got := alpha.raw(http.MethodPut, "/me/avatar", "image/png", pngBytes(t, 32, 32)); got.status != http.StatusOK {
		t.Fatalf("photo d'Alpha : %d %s", got.status, got.body)
	}
	if got := alpha.post("/me/email-change-requests", map[string]string{"email": "alpha.new@example.org"}); got.status != http.StatusCreated {
		t.Fatalf("changement d'adresse d'Alpha : %d %s", got.status, got.body)
	}
	if err := e.db.Exec("UPDATE hero_profiles SET bio = 'Décrit Alpha' WHERE user_id = ?", alpha.userID()).Error; err != nil {
		t.Fatal(err)
	}
	retiredID := retired.userID()
	if err := e.db.Exec("UPDATE users SET status = 'deactivated' WHERE id = ?", retiredID).Error; err != nil {
		t.Fatal(err)
	}

	incidentID := e.declareIncident(reporter, moderator, "directory")
	started := alpha.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention struct{ ID string }
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("intervention : %d %s", started.status, started.body)
	}
	if got := beta.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion de Beta : %d %s", got.status, got.body)
	}
	if err := e.db.Exec(
		"INSERT INTO hero_reviews (id, intervention_id, hero_id, rating, status) VALUES (gen_random_uuid(), ?, ?, 4, 'published')",
		intervention.ID, alpha.userID(),
	).Error; err != nil {
		t.Fatal(err)
	}

	var listed adminHeroPage
	got := moderator.get("/admin/heroes")
	got.decode(t, &listed)
	if got.status != http.StatusOK || listed.Total != 3 || len(listed.Items) != 3 {
		t.Fatalf("annuaire : %d %s", got.status, got.body)
	}
	// Par pseudo ; la demande en attente (Candidate) n'y figure pas, le héros désactivé oui.
	if names := []string{listed.Items[0].Alias, listed.Items[1].Alias, listed.Items[2].Alias}; names[0] != "Alpha" || names[1] != "Beta" || names[2] != "Retired" {
		t.Fatalf("pseudos = %v", names)
	}

	alphaView := listed.Items[0]
	if alphaView.ID != alpha.userID() || alphaView.Power != "Décrit Alpha" || alphaView.Civility != "monsieur" ||
		alphaView.FirstName != "Steve" || alphaView.LastName != "Rogers" || alphaView.BirthDate != "1990-07-04" ||
		alphaView.Email != "alpha@example.org" || alphaView.MobilePhone != "0611223344" || alphaView.AccountStatus != "active" {
		t.Errorf("identité d'Alpha = %+v", alphaView)
	}
	if alphaView.Address.Street != "1 Stark Tower" || alphaView.Address.Complement != "Étage 90" || alphaView.Address.Country != "US" {
		t.Errorf("adresse d'Alpha = %+v", alphaView.Address)
	}
	if alphaView.PendingEmail != "alpha.new@example.org" {
		t.Errorf("pendingEmail = %q", alphaView.PendingEmail)
	}
	if alphaView.AverageRating != 4 || alphaView.InterventionsCount != 1 || alphaView.LastInterventionAt == nil {
		t.Errorf("activité d'Alpha = %+v", alphaView)
	}
	if alphaView.AvatarURL == nil || !strings.HasPrefix(*alphaView.AvatarURL, apiBase+"/avatars/") {
		t.Fatalf("avatarUrl = %v : la photo envoyée par /me/avatar doit devenir une adresse publique", alphaView.AvatarURL)
	}
	if listed.Items[1].InterventionsCount != 1 || listed.Items[1].LastInterventionAt == nil {
		t.Errorf("Beta a participé à l'intervention : %+v", listed.Items[1])
	}
	if listed.Items[1].AvatarURL != nil || listed.Items[1].PendingEmail != "" {
		t.Errorf("Beta n'a ni photo ni demande : %+v", listed.Items[1])
	}
	if retiredView := listed.Items[2]; retiredView.AccountStatus != "deactivated" || retiredView.LastInterventionAt != nil || retiredView.InterventionsCount != 0 {
		t.Errorf("héros désactivé = %+v", retiredView)
	}

	// Régression : la liste publique doit aussi montrer la photo envoyée par /me/avatar (son nom de
	// fichier n'est ni une URL ni un chemin absolu).
	var public struct {
		Items []publicHeroView `json:"items"`
	}
	publicList := e.newActor().get("/heroes")
	publicList.decode(t, &public)
	if publicList.status != http.StatusOK || len(public.Items) != 2 || public.Items[0].Alias != "Alpha" ||
		public.Items[0].AvatarURL == nil || *public.Items[0].AvatarURL != *alphaView.AvatarURL {
		t.Errorf("liste publique : %d %s", publicList.status, publicList.body)
	}
	if strings.Contains(string(publicList.body), "alpha@example.org") || strings.Contains(string(publicList.body), "Rogers") {
		t.Errorf("la liste publique ne doit montrer aucune donnée de l'annuaire : %s", publicList.body)
	}

	var paged adminHeroPage
	pageTwo := moderator.get("/admin/heroes?limit=1&offset=1")
	pageTwo.decode(t, &paged)
	if pageTwo.status != http.StatusOK || len(paged.Items) != 1 || paged.Items[0].Alias != "Beta" || paged.Total != 3 || paged.Limit != 1 || paged.Offset != 1 {
		t.Errorf("pagination : %d %s", pageTwo.status, pageTwo.body)
	}
	if got := moderator.get("/admin/heroes?limit=0"); got.status != http.StatusUnprocessableEntity || got.code() != "validation_failed" {
		t.Errorf("limite invalide : %d %s", got.status, got.body)
	}
}

func TestAdminHeroDirectoryIsReservedToModerators(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	hero := e.approvedHero(moderator, "alpha@example.org", "Alpha")
	citizen := e.signUpAndLogin("citizen@example.org")

	if got := e.newActor().get("/admin/heroes"); got.status != http.StatusUnauthorized || got.code() != "unauthenticated" {
		t.Errorf("anonyme : %d %s", got.status, got.body)
	}
	for name, actor := range map[string]*actor{"citoyen": citizen, "héros": hero} {
		if got := actor.get("/admin/heroes"); got.status != http.StatusForbidden || got.code() != "forbidden" {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
}
