package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/signedlink"
)

func (a *actor) unsubscribeFromNewsletter(token string) response {
	return a.post("/newsletter/subscriptions/unsubscribe", map[string]string{"token": token})
}

// TestNewsletterUnsubscribeLinkNeedsNoLogin : le lien d'une newsletter désabonne sans connexion, une ou
// plusieurs fois, et le compte du titulaire cesse d'être abonné. Le lien se fabrique ici comme le fera
// l'envoi de la newsletter (Service.UnsubscribeLink, testé dans le paquet newsletter).
func TestNewsletterUnsubscribeLinkNeedsNoLogin(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	if got := citizen.put("/me/newsletter", map[string]any{"subscribed": true}); got.status != http.StatusNoContent {
		t.Fatalf("abonnement : %d %s", got.status, got.body)
	}
	var stored struct{ ID uuid.UUID }
	if err := e.db.Raw(`SELECT id FROM newsletter_subscriptions WHERE email = 'steve@avengers.example'`).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	subscriptionID := stored.ID
	signer, err := signedlink.NewSigner(e.cfg.Crypto.Keys, e.cfg.Crypto.ActiveKeyID)
	if err != nil {
		t.Fatal(err)
	}
	token := signer.Sign(signedlink.PurposeNewsletterUnsubscribe, subscriptionID, time.Time{})
	reader := e.newActor() // le lien s'ouvre sur un autre appareil : aucune session

	forged := []byte(token)
	forged[len(forged)/2] ^= 0x01
	for name, bad := range map[string]string{
		"vide": "", "falsifié": string(forged), "tronqué": token[:len(token)-3],
		"lien de confirmation": signer.Sign(signedlink.PurposeNewsletterConfirm, subscriptionID, time.Time{}),
		"abonnement inconnu":   signer.Sign(signedlink.PurposeNewsletterUnsubscribe, uuid.New(), time.Time{}),
	} {
		if got := reader.unsubscribeFromNewsletter(bad); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if !citizen.account(t).NewsletterSubscribed {
		t.Fatal("aucun lien invalide ne doit désabonner")
	}

	// Des années plus tard, et deux fois de suite : toujours un succès.
	e.clock.Advance(3 * 365 * 24 * time.Hour)
	for i := range 2 {
		if got := reader.unsubscribeFromNewsletter(token); got.status != http.StatusNoContent {
			t.Fatalf("désinscription %d : %d %s", i+1, got.status, got.body)
		}
	}
	if row, found := e.subscription("steve@avengers.example"); !found || row.Status != "unsubscribed" || !row.HasUnsubbed || !row.Confirmed {
		t.Errorf("abonnement après le lien : %+v", row)
	}
	if got := citizen.login("steve@avengers.example", goodPassword); got.status != http.StatusOK {
		t.Fatalf("reconnexion : %d %s", got.status, got.body)
	}
	if citizen.account(t).NewsletterSubscribed {
		t.Error("le compte ne doit plus être abonné")
	}

	// Se réabonner depuis le pied de page redemande le consentement : le vieux lien ne le donne pas.
	if got := reader.subscribeToNewsletter("steve@avengers.example"); got.status != http.StatusAccepted {
		t.Fatalf("nouvelle demande : %d %s", got.status, got.body)
	}
	if row, _ := e.subscription("steve@avengers.example"); row.Status != "pending" {
		t.Errorf("après la nouvelle demande : %+v", row)
	}
}

func TestNewsletterUnsubscribeRefusesMalformedRequests(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()

	if got := visitor.post("/newsletter/subscriptions/unsubscribe", map[string]string{"inconnu": "x"}); got.status != http.StatusBadRequest || got.code() != "bad_request" {
		t.Errorf("corps mal formé : %d %s", got.status, got.body)
	}
	// Pas d'action par simple GET : un robot qui ouvre le lien d'un mail ne doit désabonner personne.
	e.spec = nil // le contrat ne décrit pas cette méthode
	if got := visitor.get("/newsletter/subscriptions/unsubscribe"); got.status != http.StatusMethodNotAllowed {
		t.Errorf("GET : %d", got.status)
	}
}
