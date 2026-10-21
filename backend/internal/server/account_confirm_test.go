package server_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Le lien de confirmation prouve que son porteur a accès à la boîte de la nouvelle adresse : il ne doit
// jamais valoir s'il est falsifié, périmé, ou fabriqué pour une autre action.
func TestEmailChangeConfirmationLinkRefusesForgeryExpiryAndOtherPurposes(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.requestEmailChange(t, "nouvelle@avengers.example")
	token := e.confirmationTokens("nouvelle@avengers.example")[0]
	visitor := e.newActor()
	confirm := func(token string) response {
		return visitor.post("/email-change-requests/confirm", map[string]string{"token": token})
	}

	// Un lien de newsletter (autre intention, même signature) ne confirme pas une adresse.
	if got := visitor.post("/newsletter/subscriptions", map[string]string{"email": "lecteur@example.org"}); got.status != http.StatusAccepted {
		t.Fatalf("abonnement : %d %s", got.status, got.body)
	}
	newsletterLink := signedToken.FindStringSubmatch(e.mailsTo("lecteur@example.org", "newsletter")[0].Text)[1]

	forged := []byte(token)
	forged[len(forged)/2] ^= 1 // un caractère de la signature ou de l'objet change
	for name, bad := range map[string]string{
		"falsifié": string(forged), "d'une autre intention": newsletterLink, "tronqué": token[:len(token)-4],
		"n'importe quoi": "pas-un-lien", "vide": "",
	} {
		if got := confirm(bad); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
			t.Errorf("lien %s : %d %s", name, got.status, got.body)
		}
	}
	if view := citizen.account(t); view.PendingEmailConfirmed {
		t.Fatal("aucun lien invalide ne doit confirmer l'adresse")
	}
	if got := visitor.post("/email-change-requests/confirm", map[string]string{"inconnu": "x"}); got.status != http.StatusBadRequest || got.code() != "bad_request" {
		t.Errorf("corps mal formé : %d %s", got.status, got.body)
	}

	// Périmé : la durée de validité est celle de la configuration (48 h par défaut). Les sessions, elles,
	// expirent après 2 h d'inactivité : la personne se reconnecte pour vérifier son profil.
	e.clock.Advance(48*time.Hour + time.Minute)
	if got := confirm(token); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
		t.Errorf("lien périmé : %d %s", got.status, got.body)
	}
	if got := citizen.login("steve@avengers.example", goodPassword); got.status != http.StatusOK {
		t.Fatalf("reconnexion : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.PendingEmailConfirmed {
		t.Fatal("un lien périmé ne confirme rien")
	}

	// Une nouvelle demande renvoie un lien valable, et l'ancien ne vaut toujours pas.
	if got := citizen.requestEmailChange(t, "nouvelle@avengers.example"); got.status != http.StatusCreated {
		t.Fatalf("nouvelle demande : %d %s", got.status, got.body)
	}
	if got := confirm(token); got.status != http.StatusBadRequest {
		t.Errorf("l'ancien lien ne doit pas valoir pour la nouvelle demande : %d %s", got.status, got.body)
	}
	if got := e.confirmEmailChange("nouvelle@avengers.example"); got.status != http.StatusNoContent {
		t.Errorf("le nouveau lien : %d %s", got.status, got.body)
	}
}

// Une panne d'envoi n'annule pas la demande : la personne peut la renouveler, et la modération n'est pas
// bloquée par un mail perdu.
func TestEmailChangeSurvivesMailFailures(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")

	e.mailer.FailWith(errors.New("serveur de mail en panne"))
	if got := citizen.requestEmailChange(t, "nouvelle@avengers.example"); got.status != http.StatusCreated {
		t.Fatalf("demande pendant la panne : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.PendingEmail != "nouvelle@avengers.example" {
		t.Errorf("la demande doit être enregistrée malgré la panne : %+v", view)
	}
	e.mailer.FailWith(nil)

	// Renouvelée une fois le service rétabli, elle part et aboutit.
	citizen.requestConfirmedEmailChange(t, "nouvelle@avengers.example")
	change := moderator.emailChanges(t, "?status=pending")[0]

	// Les mails de décision sont envoyés après la décision : leur perte ne l'annule pas.
	e.mailer.FailWith(errors.New("serveur de mail en panne"))
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil); got.status != http.StatusNoContent {
		t.Fatalf("validation pendant la panne : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.Email != "nouvelle@avengers.example" {
		t.Errorf("l'adresse doit avoir changé : %+v", view)
	}
}

// Une demande jamais confirmée ne peut plus aboutir une fois son lien expiré : elle ne reste pas dans la
// file des modérateurs. Les demandes confirmées ou déjà décidées ne sont jamais touchées.
func TestJanitorPurgesUnconfirmedEmailChanges(t *testing.T) {
	e := newE2E(t, map[string]string{
		"HEROS_HTTP_ADDR": freeAddress(t), "HEROS_CLEANUP_INTERVAL": "50ms", "HEROS_TOKEN_RETENTION": "1h",
	})
	owners := make([]string, 4)
	for i, email := range []string{"a@avengers.example", "b@avengers.example", "c@avengers.example", "d@avengers.example"} {
		owners[i] = e.signUpAndLogin(email).userID()
	}
	// L'application tourne sur l'horloge fictive du test : les dates se calculent depuis elle, pas depuis celle de la base.
	now := e.clock.Now()
	ago := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }
	for i, row := range []struct {
		name, status string
		createdAt    time.Time
		confirmedAt  *time.Time
	}{
		{"abandonnee", "pending", now.Add(-2 * time.Hour), nil},
		{"recente", "pending", now.Add(-10 * time.Minute), nil},
		{"confirmee", "pending", now.Add(-2 * time.Hour), ago(time.Hour)},
		{"refusee", "rejected", now.Add(-2 * time.Hour), nil},
	} {
		err := e.db.Exec(`INSERT INTO email_change_requests (id, user_id, new_email, status, created_at, confirmed_at) VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New(), owners[i], row.name+"@nouvelle.example", row.status, row.createdAt, row.confirmedAt).Error
		if err != nil {
			t.Fatalf("préparation : %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- e.server.Run(ctx) }()
	defer cancel()

	remaining := func() []string {
		var emails []string
		e.db.Raw(`SELECT new_email FROM email_change_requests ORDER BY new_email`).Scan(&emails)
		return emails
	}
	waitFor(t, startupWait, "la demande abandonnée est supprimée", func() bool { return len(remaining()) == 3 })
	time.Sleep(300 * time.Millisecond) // quelques passages de plus : rien d'autre ne doit disparaître
	got := remaining()
	if len(got) != 3 || got[0] != "confirmee@nouvelle.example" || got[1] != "recente@nouvelle.example" || got[2] != "refusee@nouvelle.example" {
		t.Errorf("demandes restantes = %v", got)
	}

	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run : %v", err)
		}
	case <-time.After(stopWait):
		t.Fatal("la tâche de nettoyage doit s'arrêter avec le serveur")
	}
}
