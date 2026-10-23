package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/mail"
)

type issueBody struct {
	ID              string  `json:"id"`
	Subject         string  `json:"subject"`
	Body            string  `json:"body"`
	RecipientUserID *string `json:"recipientUserId"`
	Sent            bool    `json:"sent"`
	SentAt          *string `json:"sentAt"`
}

func (e *e2e) userID(email string) string {
	e.t.Helper()
	var id string
	if err := e.db.Raw(`SELECT id::text FROM users WHERE email = ?`, email).Scan(&id).Error; err != nil || id == "" {
		e.t.Fatalf("compte %s introuvable : %v", email, err)
	}
	return id
}

// newsletterMessages attend que l'envoi en arrière-plan ait déposé want mails de newsletter après les
// skip premiers (ceux des envois précédents), et retourne ces want mails.
func (e *e2e) newsletterMessages(skip, want int) []mail.Message {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var found []mail.Message
		for _, m := range e.mailer.Messages() {
			if m.ListUnsubscribe != "" {
				found = append(found, m)
			}
		}
		if len(found) >= skip+want || time.Now().After(deadline) {
			if len(found) != skip+want {
				e.t.Fatalf("%d mails de newsletter, attendu %d", len(found), skip+want)
			}
			return found[skip:]
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestNewsletterIssuesLifecycle(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	steve := e.signUpAndLogin("steve@avengers.example")
	tony := e.signUpAndLogin("tony@avengers.example")
	e.signUpAndLogin("bruce@avengers.example") // jamais abonné
	for _, a := range []*actor{steve, tony} {
		if got := a.put("/me/newsletter", map[string]any{"subscribed": true}); got.status != http.StatusNoContent {
			t.Fatalf("abonnement : %d %s", got.status, got.body)
		}
	}
	visitor := e.newActor()
	visitor.subscribeToNewsletter("eve@example.org") // en attente : pas de consentement

	// Réservé aux modérateurs.
	if got := steve.get("/admin/newsletters"); got.status != http.StatusForbidden {
		t.Errorf("citoyen : %d", got.status)
	}
	if got := e.newActor().get("/admin/newsletters"); got.status != http.StatusUnauthorized {
		t.Errorf("visiteur : %d", got.status)
	}

	// Brouillon : le HTML est assaini, les champs sont validés.
	if got := moderator.post("/admin/newsletters", map[string]any{"subject": " ", "body": ""}); got.status != http.StatusUnprocessableEntity {
		t.Fatalf("saisie vide : %d %s", got.status, got.body)
	}
	var draft issueBody
	created := moderator.post("/admin/newsletters", map[string]any{
		"subject": "Les nouvelles du mois", "body": `<p>Bonjour <strong>à tous</strong></p><script>alert(1)</script>`,
	})
	if created.status != http.StatusCreated {
		t.Fatalf("création : %d %s", created.status, created.body)
	}
	created.decode(t, &draft)
	if draft.Sent || draft.SentAt != nil || strings.Contains(draft.Body, "script") || !strings.Contains(draft.Body, "<strong>") {
		t.Fatalf("brouillon : %+v", draft)
	}

	// Un destinataire inexistant est refusé ; un titulaire non abonné aussi, à l'envoi.
	if got := moderator.put("/admin/newsletters/"+draft.ID, map[string]any{
		"subject": "Les nouvelles du mois", "body": "<p>Texte</p>", "recipientUserId": "00000000-0000-4000-8000-000000000000",
	}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("destinataire inconnu : %d %s", got.status, got.body)
	}
	if got := moderator.put("/admin/newsletters/"+draft.ID, map[string]any{
		"subject": "Les nouvelles du mois", "body": "<p>Texte</p>", "recipientUserId": e.userID("bruce@avengers.example"),
	}); got.status != http.StatusOK {
		t.Fatalf("destinataire non abonné : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/newsletters/"+draft.ID+"/send", nil); got.status != http.StatusUnprocessableEntity || got.code() != "validation_failed" {
		t.Fatalf("envoi à un non-abonné : %d %s", got.status, got.body)
	}
	if got := moderator.get("/admin/newsletters/" + draft.ID); got.status != http.StatusOK || strings.Contains(string(got.body), `"sent":true`) {
		t.Errorf("le numéro doit rester un brouillon : %d %s", got.status, got.body)
	}

	// Envoi à tous les abonnés confirmés (titulaires de compte ou non).
	if got := moderator.put("/admin/newsletters/"+draft.ID, map[string]any{
		"subject": "Les nouvelles du mois", "body": "<p>Bonjour <strong>à tous</strong></p>", "recipientUserId": nil,
	}); got.status != http.StatusOK {
		t.Fatalf("retour à tous : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/newsletters/"+draft.ID+"/send", nil); got.status != http.StatusAccepted {
		t.Fatalf("envoi : %d %s", got.status, got.body)
	}
	messages := e.newsletterMessages(0, 2)
	seen := map[string]bool{}
	for _, m := range messages {
		seen[m.To] = true
		if m.Subject != "Les nouvelles du mois" || !strings.Contains(m.HTML, "<strong>à tous</strong>") {
			t.Errorf("mail : %+v", m)
		}
	}
	if !seen["steve@avengers.example"] || !seen["tony@avengers.example"] {
		t.Errorf("destinataires : %v", seen)
	}

	// Envoyé : lecture seule et un seul envoi.
	var sent issueBody
	moderator.get("/admin/newsletters/" + draft.ID).decode(t, &sent)
	if !sent.Sent || sent.SentAt == nil {
		t.Errorf("numéro après envoi : %+v", sent)
	}
	if got := moderator.put("/admin/newsletters/"+draft.ID, map[string]any{"subject": "x", "body": "<p>y</p>"}); got.status != http.StatusConflict || got.code() != "newsletter_already_sent" {
		t.Errorf("modification : %d %s", got.status, got.body)
	}
	if got := moderator.request(http.MethodDelete, "/admin/newsletters/"+draft.ID, nil); got.status != http.StatusConflict {
		t.Errorf("suppression : %d", got.status)
	}
	if got := moderator.post("/admin/newsletters/"+draft.ID+"/send", nil); got.status != http.StatusConflict {
		t.Errorf("second envoi : %d", got.status)
	}

	// Envoi à un seul abonné.
	var single issueBody
	moderator.post("/admin/newsletters", map[string]any{
		"subject": "Pour Tony", "body": "<p>Salut</p>", "recipientUserId": e.userID("tony@avengers.example"),
	}).decode(t, &single)
	if got := moderator.post("/admin/newsletters/"+single.ID+"/send", nil); got.status != http.StatusAccepted {
		t.Fatalf("envoi individuel : %d %s", got.status, got.body)
	}
	if only := e.newsletterMessages(2, 1); only[0].To != "tony@avengers.example" {
		t.Errorf("destinataire : %s", only[0].To)
	}

	// Liste : les plus récents d'abord, corps exclu ; un brouillon se supprime.
	var other issueBody
	moderator.post("/admin/newsletters", map[string]any{"subject": "À jeter", "body": "<p>Brouillon</p>"}).decode(t, &other)
	if got := moderator.request(http.MethodDelete, "/admin/newsletters/"+other.ID, nil); got.status != http.StatusNoContent {
		t.Errorf("suppression du brouillon : %d", got.status)
	}
	if got := moderator.get("/admin/newsletters/" + other.ID); got.status != http.StatusNotFound {
		t.Errorf("brouillon supprimé : %d", got.status)
	}
	list := moderator.get("/admin/newsletters")
	var page struct {
		Items []issueBody `json:"items"`
		Total int         `json:"total"`
	}
	list.decode(t, &page)
	if page.Total != 2 || len(page.Items) != 2 || page.Items[0].ID != single.ID || page.Items[0].Body != "" {
		t.Errorf("liste : %s", list.body)
	}
	if got := moderator.get("/admin/newsletters/not-a-uuid"); got.status != http.StatusNotFound {
		t.Errorf("identifiant invalide : %d", got.status)
	}
}

func TestNewsletterIssueWithoutSubscriberStaysDraft(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	var draft issueBody
	moderator.post("/admin/newsletters", map[string]any{"subject": "Seul au monde", "body": "<p>Personne</p>"}).decode(t, &draft)
	if got := moderator.post("/admin/newsletters/"+draft.ID+"/send", nil); got.status != http.StatusUnprocessableEntity {
		t.Fatalf("envoi sans abonné : %d %s", got.status, got.body)
	}
	if e.count("newsletters", "sent_at IS NOT NULL") != 0 {
		t.Error("le numéro ne doit pas être marqué envoyé")
	}
}
