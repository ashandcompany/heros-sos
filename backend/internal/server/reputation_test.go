package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/reputation"
)

func TestHeroReputationWorkflow(t *testing.T) {
	e := newE2E(t, nil)
	reporter := e.signUpAndLogin("reporter@example.org")
	otherCitizen := e.signUpAndLogin("other@example.org")
	moderator := e.newModerator("moderator@shield.example")
	lead := e.approvedHero(moderator, "lead@example.org", "Lumen")
	member := e.approvedHero(moderator, "member@example.org", "Rempart")
	outsider := e.approvedHero(moderator, "outsider@example.org", "Sentinelle")
	incidentID := e.declareIncident(reporter, moderator, "reputation")

	started := lead.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	if got := member.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "succeeded"}); got.status != http.StatusNoContent {
		t.Fatalf("proposition de résultat : %d %s", got.status, got.body)
	}
	if got := lead.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("validation lead : %d %s", got.status, got.body)
	}
	if got := member.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("validation membre : %d %s", got.status, got.body)
	}

	var reviewMailCount int
	for _, message := range e.mailer.Messages() {
		if message.Subject == "Évaluez l'intervention de Heros-SOS" {
			reviewMailCount++
			if message.To != "reporter@example.org" {
				t.Errorf("courriel envoyé à %q plutôt qu'au déclarant", message.To)
			}
		}
	}
	if reviewMailCount != 2 {
		t.Fatalf("courriels de notation = %d, attendu un par héros confirmé", reviewMailCount)
	}

	type reviewView struct {
		ID      string  `json:"id"`
		HeroID  string  `json:"heroId"`
		Rating  int     `json:"rating"`
		Comment *string `json:"comment"`
		Status  string  `json:"status"`
		Reply   *struct {
			Content  string  `json:"content"`
			Status   string  `json:"status"`
			EditedAt *string `json:"editedAt"`
		} `json:"reply"`
	}
	type rateablePage struct {
		Items []struct {
			InterventionID string      `json:"interventionId"`
			HeroID         string      `json:"heroId"`
			Review         *reviewView `json:"review"`
		} `json:"items"`
		Total int `json:"total"`
	}
	listed := reporter.get("/me/interventions")
	var rateable rateablePage
	listed.decode(t, &rateable)
	if listed.status != http.StatusOK || rateable.Total != 2 || len(rateable.Items) != 2 {
		t.Fatalf("interventions à noter : %d %+v %s", listed.status, rateable, listed.body)
	}
	if got := otherCitizen.get("/me/interventions"); got.status != http.StatusOK || len(decodeItems(t, got.body)) != 0 {
		t.Errorf("un autre citoyen voit les interventions d'autrui : %d %s", got.status, got.body)
	}

	leadReviewPath := "/interventions/" + intervention.ID + "/reviews"
	if got := reporter.post(leadReviewPath, map[string]any{
		"heroId": lead.userID(), "rating": 5, "comment": "Intervention rapide et efficace.",
	}); got.status != http.StatusNoContent {
		t.Fatalf("note du lead : %d %s", got.status, got.body)
	}
	if got := reporter.post(leadReviewPath, map[string]any{"heroId": lead.userID(), "rating": 4}); got.status != http.StatusConflict || got.code() != reputation.CodeAlreadyReviewed {
		t.Errorf("doublon : %d %s", got.status, got.body)
	}
	if got := otherCitizen.post(leadReviewPath, map[string]any{"heroId": member.userID(), "rating": 4}); got.status != http.StatusForbidden {
		t.Errorf("tiers pouvant noter : %d %s", got.status, got.body)
	}
	if got := reporter.post(leadReviewPath, map[string]any{"heroId": outsider.userID(), "rating": 4}); got.status != http.StatusForbidden {
		t.Errorf("héros non participant pouvant être noté : %d %s", got.status, got.body)
	}
	if got := reporter.post(leadReviewPath, map[string]any{"heroId": member.userID(), "rating": 3}); got.status != http.StatusNoContent {
		t.Fatalf("note du membre sans commentaire : %d %s", got.status, got.body)
	}
	if got := reporter.post(leadReviewPath, map[string]any{"heroId": outsider.userID(), "rating": 6}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("note hors bornes : %d %s", got.status, got.body)
	}

	listed = reporter.get("/me/interventions")
	listed.decode(t, &rateable)
	var memberReviewID string
	for _, item := range rateable.Items {
		if item.HeroID == member.userID() && item.Review != nil {
			memberReviewID = item.Review.ID
			break
		}
	}
	if memberReviewID == "" {
		t.Fatal("avis sans commentaire absent des interventions du déclarant")
	}
	var public struct {
		AverageRating float64      `json:"averageRating"`
		RatingsCount  int          `json:"ratingsCount"`
		Reviews       []reviewView `json:"reviews"`
	}
	memberPublic := e.newActor().get("/heroes/" + member.userID() + "/reviews")
	memberPublic.decode(t, &public)
	if memberPublic.status != http.StatusOK || public.RatingsCount != 1 || public.AverageRating != 3 ||
		len(public.Reviews) != 1 || public.Reviews[0].Status != reputation.ReviewPublished {
		t.Fatalf("la note sans commentaire doit être publiée dès sa création : %d %+v %s", memberPublic.status, public, memberPublic.body)
	}
	if got := reporter.request(http.MethodPut, "/me/reviews/"+memberReviewID, map[string]any{"rating": 4}); got.status != http.StatusNoContent {
		t.Fatalf("modification de la note sans commentaire : %d %s", got.status, got.body)
	}

	publicAfterLead := public
	publicResponse := e.newActor().get("/heroes/" + lead.userID() + "/reviews")
	publicResponse.decode(t, &publicAfterLead)
	if publicResponse.status != http.StatusOK || publicAfterLead.RatingsCount != 0 || publicAfterLead.AverageRating != 0 {
		t.Fatalf("réputation publique : %d %+v %s", publicResponse.status, publicAfterLead, publicResponse.body)
	}
	memberPublic = e.newActor().get("/heroes/" + member.userID() + "/reviews")
	memberPublic.decode(t, &public)
	if memberPublic.status != http.StatusOK || public.RatingsCount != 1 || public.AverageRating != 4 {
		t.Fatalf("la note sans commentaire modifiée doit rester publiée : %d %+v %s", memberPublic.status, public, memberPublic.body)
	}
	var heroPage struct {
		AverageRating float64      `json:"averageRating"`
		RatingsCount  int          `json:"ratingsCount"`
		Ratings       []reviewView `json:"ratings"`
		Total         int          `json:"total"`
		Limit         int          `json:"limit"`
		Offset        int          `json:"offset"`
	}
	heroReviews := lead.get("/hero/reviews")
	heroReviews.decode(t, &heroPage)
	if heroReviews.status != http.StatusOK || heroPage.AverageRating != 0 || heroPage.RatingsCount != 0 ||
		heroPage.Total != 1 || len(heroPage.Ratings) != 1 || heroPage.Ratings[0].Comment == nil ||
		heroPage.Ratings[0].Status != reputation.ReviewPending {
		t.Fatalf("avis du héros : %d %+v %s", heroReviews.status, heroPage, heroReviews.body)
	}
	if got := reporter.get("/hero/reviews"); got.status != http.StatusForbidden {
		t.Errorf("un citoyen consulte l'espace héros : %d", got.status)
	}

	reviewID := heroPage.Ratings[0].ID
	if got := otherCitizen.request(http.MethodPut, "/me/reviews/"+reviewID, map[string]any{"rating": 2}); got.status != http.StatusForbidden {
		t.Errorf("un autre citoyen modifie l'avis : %d %s", got.status, got.body)
	}
	if got := reporter.request(http.MethodPut, "/me/reviews/"+reviewID, map[string]any{
		"rating": 4, "comment": "Intervention bien menée, avis mis à jour.",
	}); got.status != http.StatusNoContent {
		t.Fatalf("modification de l'avis commenté : %d %s", got.status, got.body)
	}
	heroReviews = lead.get("/hero/reviews")
	heroReviews.decode(t, &heroPage)
	if heroPage.Ratings[0].Status != reputation.ReviewPending || heroPage.Ratings[0].Rating != 4 ||
		heroPage.Ratings[0].Comment == nil || *heroPage.Ratings[0].Comment != "Intervention bien menée, avis mis à jour." {
		t.Fatalf("l'avis modifié avec texte doit être remis en attente : %+v", heroPage.Ratings[0])
	}
	publicResponse = e.newActor().get("/heroes/" + lead.userID() + "/reviews")
	publicResponse.decode(t, &publicAfterLead)
	if publicAfterLead.RatingsCount != 0 || len(publicAfterLead.Reviews) != 0 {
		t.Fatalf("un avis modifié en attente reste public : %+v", publicAfterLead)
	}
	moderationReviews := moderator.get("/admin/hero-reviews?status=pending")
	var reviewPage struct {
		Items []reviewView `json:"items"`
		Total int          `json:"total"`
	}
	moderationReviews.decode(t, &reviewPage)
	if moderationReviews.status != http.StatusOK || reviewPage.Total != 1 ||
		len(reviewPage.Items) != 1 || reviewPage.Items[0].ID != reviewID {
		t.Fatalf("avis commenté à modérer : %d %+v %s", moderationReviews.status, reviewPage, moderationReviews.body)
	}
	if got := moderator.request(http.MethodPatch, "/admin/hero-reviews/"+reviewID, map[string]string{"status": reputation.ReviewPublished}); got.status != http.StatusNoContent {
		t.Fatalf("publication de l'avis commenté : %d %s", got.status, got.body)
	}
	publicResponse = e.newActor().get("/heroes/" + lead.userID() + "/reviews")
	publicResponse.decode(t, &publicAfterLead)
	if publicAfterLead.RatingsCount != 1 || publicAfterLead.AverageRating != 4 {
		t.Fatalf("l'avis validé doit réapparaître au public : %+v", publicAfterLead)
	}
	heroReviews = lead.get("/hero/reviews")
	heroReviews.decode(t, &heroPage)
	if heroPage.RatingsCount != 1 || heroPage.AverageRating != 4 || heroPage.Ratings[0].Status != reputation.ReviewPublished {
		t.Fatalf("l'avis validé doit réintégrer la moyenne : %+v", heroPage)
	}

	replyPath := "/hero/reviews/" + reviewID + "/reply"
	if got := outsider.request(http.MethodPut, replyPath, map[string]string{"content": "Merci pour votre retour."}); got.status != http.StatusForbidden {
		t.Errorf("un autre héros répond à l'avis : %d %s", got.status, got.body)
	}
	if got := lead.request(http.MethodPut, replyPath, map[string]string{"content": "Merci pour votre retour."}); got.status != http.StatusNoContent {
		t.Fatalf("réponse initiale : %d %s", got.status, got.body)
	}
	if got := lead.request(http.MethodPut, replyPath, map[string]string{"content": "Merci, réponse mise à jour."}); got.status != http.StatusNoContent {
		t.Fatalf("modification de la réponse : %d %s", got.status, got.body)
	}
	heroReviews = lead.get("/hero/reviews")
	heroReviews.decode(t, &heroPage)
	if heroPage.Ratings[0].Reply == nil || heroPage.Ratings[0].Reply.Status != reputation.ReplyPending || heroPage.Ratings[0].Reply.EditedAt == nil {
		t.Fatalf("édition doit remettre la réponse en attente : %+v", heroPage.Ratings[0].Reply)
	}
	moderationReplies := moderator.get("/admin/hero-review-replies?status=pending")
	var replyPage struct {
		Items []struct {
			Reply struct {
				ReviewID string `json:"reviewId"`
				Status   string `json:"status"`
			} `json:"reply"`
		} `json:"items"`
		Total int `json:"total"`
	}
	moderationReplies.decode(t, &replyPage)
	if moderationReplies.status != http.StatusOK || replyPage.Total != 1 || replyPage.Items[0].Reply.Status != reputation.ReplyPending {
		t.Fatalf("réponse en modération : %d %+v %s", moderationReplies.status, replyPage, moderationReplies.body)
	}
	if got := moderator.request(http.MethodPatch, "/admin/hero-review-replies/"+reviewID, map[string]string{"status": reputation.ReplyPublished}); got.status != http.StatusNoContent {
		t.Fatalf("publication réponse : %d %s", got.status, got.body)
	}
	publicResponse = e.newActor().get("/heroes/" + lead.userID() + "/reviews")
	publicResponse.decode(t, &publicAfterLead)
	if len(publicAfterLead.Reviews) != 1 || publicAfterLead.Reviews[0].Reply == nil || publicAfterLead.Reviews[0].Reply.Content != "Merci, réponse mise à jour." {
		t.Fatalf("réponse publiée absente du public : %+v", publicAfterLead)
	}
	if got := moderator.request(http.MethodPatch, "/admin/hero-reviews/"+reviewID, map[string]string{"status": reputation.ReviewHidden}); got.status != http.StatusNoContent {
		t.Fatalf("masquage avis : %d %s", got.status, got.body)
	}
	heroReviews = lead.get("/hero/reviews")
	heroReviews.decode(t, &heroPage)
	if heroReviews.status != http.StatusOK || heroPage.RatingsCount != 0 || heroPage.AverageRating != 0 ||
		heroPage.Total != 1 || heroPage.Ratings[0].Status != reputation.ReviewHidden {
		t.Fatalf("avis masqué conservé dans la liste privée mais exclu du score : %d %+v", heroReviews.status, heroPage)
	}
	publicResponse = e.newActor().get("/heroes/" + lead.userID() + "/reviews")
	publicResponse.decode(t, &publicAfterLead)
	if publicResponse.status != http.StatusOK || publicAfterLead.RatingsCount != 0 || publicAfterLead.AverageRating != 0 || len(publicAfterLead.Reviews) != 0 {
		t.Fatalf("l'avis masqué doit sortir du public et de la moyenne : %d %+v", publicResponse.status, publicAfterLead)
	}
}

func TestHeroReporterCanRateOtherParticipantsButNotThemself(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	reporter := e.approvedHero(moderator, "reporter-hero@example.org", "Témoin")
	participant := e.approvedHero(moderator, "participant@example.org", "Protecteur")
	incidentID := e.declareIncident(reporter, moderator, "hero_reporter_review")

	started := participant.post("/incidents/"+incidentID+"/interventions", nil)
	var intervention interventionView
	started.decode(t, &intervention)
	if started.status != http.StatusCreated {
		t.Fatalf("démarrage : %d %s", started.status, started.body)
	}
	if got := reporter.post("/interventions/"+intervention.ID+"/join", nil); got.status != http.StatusNoContent {
		t.Fatalf("adhésion du déclarant : %d %s", got.status, got.body)
	}
	if got := participant.post("/interventions/"+intervention.ID+"/outcome", map[string]string{"status": "failed"}); got.status != http.StatusNoContent {
		t.Fatalf("proposition de résultat : %d %s", got.status, got.body)
	}
	if got := participant.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("validation participant : %d %s", got.status, got.body)
	}
	if got := reporter.post("/interventions/"+intervention.ID+"/outcome/validate", map[string]bool{"approved": true}); got.status != http.StatusNoContent {
		t.Fatalf("validation déclarant : %d %s", got.status, got.body)
	}

	var page struct {
		Items []struct {
			HeroID string `json:"heroId"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if got := reporter.get("/me/interventions"); got.status != http.StatusOK {
		t.Fatalf("liste du héros déclarant : %d %s", got.status, got.body)
	} else {
		got.decode(t, &page)
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].HeroID != participant.userID() {
			t.Fatalf("le héros déclarant ne doit voir que les autres participants : %+v", page)
		}
	}
	reviewPath := "/interventions/" + intervention.ID + "/reviews"
	if got := reporter.post(reviewPath, map[string]any{"heroId": participant.userID(), "rating": 4}); got.status != http.StatusNoContent {
		t.Fatalf("note d'un autre héros : %d %s", got.status, got.body)
	}
	if got := reporter.post(reviewPath, map[string]any{"heroId": reporter.userID(), "rating": 5}); got.status != http.StatusForbidden {
		t.Errorf("le héros déclarant peut se noter lui-même : %d %s", got.status, got.body)
	}
}

func decodeItems(t *testing.T, body []byte) []any {
	t.Helper()
	var page struct {
		Items []any `json:"items"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	return page.Items
}
