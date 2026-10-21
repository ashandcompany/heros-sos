package server_test

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/mail"
)

// signedLinkToken retrouve le jeton signé du lien du dernier mail (il est bien plus long que les jetons
// de 43 caractères de l'activation).
var signedLinkToken = regexp.MustCompile(`/newsletter/confirmation\?token=([A-Za-z0-9_-]+)`)

func (e *e2e) lastConfirmationToken() string {
	e.t.Helper()
	message, ok := e.mailer.Last()
	if !ok {
		e.t.Fatal("aucun mail envoyé")
	}
	match := signedLinkToken.FindStringSubmatch(message.Text)
	if match == nil {
		e.t.Fatalf("aucun lien de confirmation dans le mail :\n%s", message.Text)
	}
	return match[1]
}

func (a *actor) subscribeToNewsletter(email string) response {
	return a.post("/newsletter/subscriptions", map[string]string{"email": email})
}

func (a *actor) confirmNewsletter(token string) response {
	return a.post("/newsletter/subscriptions/confirm", map[string]string{"token": token})
}

type subscriptionRow struct {
	Status, Source         string
	Confirmed, HasUnsubbed bool
	ConfirmedAt            *time.Time
}

func (e *e2e) subscription(email string) (row subscriptionRow, found bool) {
	e.t.Helper()
	result := e.db.Raw(`SELECT status, source, confirmed_at IS NOT NULL AS confirmed, unsubscribed_at IS NOT NULL AS has_unsubbed,
		confirmed_at FROM newsletter_subscriptions WHERE email = ?`, email).Scan(&row)
	if result.Error != nil {
		e.t.Fatal(result.Error)
	}
	return row, result.RowsAffected > 0
}

// ---- Newsletter : abonnement d'un visiteur ---------------------------------------------------

func TestNewsletterSubscriptionNeedsEmailConfirmation(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()

	if got := visitor.subscribeToNewsletter("Eve@Example.org"); got.status != http.StatusAccepted || len(got.body) != 0 {
		t.Fatalf("demande : %d %s", got.status, got.body)
	}
	messages := e.mailer.Messages()
	if len(messages) != 1 || messages[0].To != "eve@example.org" || messages[0].ReplyTo != "" ||
		!strings.Contains(messages[0].Subject, "newsletter") || !strings.Contains(messages[0].HTML, "/newsletter/confirmation?token=") {
		t.Fatalf("mail de confirmation = %+v", messages)
	}

	// Avant le clic, aucun consentement : l'adresse n'est pas abonnée.
	row, found := e.subscription("eve@example.org")
	if !found || row.Status != "pending" || row.Source != "footer" || row.Confirmed {
		t.Fatalf("avant confirmation : %+v (trouvée : %t)", row, found)
	}

	token := e.lastConfirmationToken()
	if got := visitor.confirmNewsletter(token); got.status != http.StatusNoContent {
		t.Fatalf("confirmation : %d %s", got.status, got.body)
	}
	row, _ = e.subscription("eve@example.org")
	if row.Status != "subscribed" || !row.Confirmed || row.ConfirmedAt == nil || !row.ConfirmedAt.Equal(startTime) {
		t.Errorf("après confirmation : %+v", row)
	}

	// Un second clic réussit sans rien changer : la date de consentement d'origine est conservée.
	e.clock.Advance(time.Hour)
	if got := visitor.confirmNewsletter(token); got.status != http.StatusNoContent {
		t.Errorf("second clic : %d %s", got.status, got.body)
	}
	if again, _ := e.subscription("eve@example.org"); again.ConfirmedAt == nil || !again.ConfirmedAt.Equal(startTime) {
		t.Errorf("la date de consentement a changé : %+v", again)
	}

	// Une adresse déjà abonnée qui se réinscrit ne reçoit pas de mail de plus.
	if got := visitor.subscribeToNewsletter("eve@example.org"); got.status != http.StatusAccepted {
		t.Errorf("réinscription : %d %s", got.status, got.body)
	}
	if len(e.mailer.Messages()) != 1 {
		t.Errorf("une adresse déjà abonnée ne doit pas recevoir de mail : %d mails", len(e.mailer.Messages()))
	}
	if kept, _ := e.subscription("eve@example.org"); kept.Status != "subscribed" {
		t.Errorf("l'abonnement confirmé ne doit pas être remis en attente : %+v", kept)
	}
}

// La réponse ne doit pas permettre de savoir si une adresse est déjà dans la liste.
func TestNewsletterSubscriptionResponseDoesNotRevealTheAddressState(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()

	unknown := visitor.subscribeToNewsletter("nouvelle@example.org")
	pending := visitor.subscribeToNewsletter("nouvelle@example.org")
	visitor.confirmNewsletter(e.lastConfirmationToken())
	subscribed := visitor.subscribeToNewsletter("nouvelle@example.org")

	for name, got := range map[string]response{"en attente": pending, "abonnée": subscribed} {
		if got.status != unknown.status || string(got.body) != string(unknown.body) {
			t.Errorf("adresse %s : %d %q, attendu comme une adresse inconnue : %d %q", name, got.status, got.body, unknown.status, unknown.body)
		}
	}
	// Relancer une demande en attente renvoie un lien (le premier a pu se perdre).
	if len(e.mailer.Messages()) != 2 {
		t.Errorf("mails = %d, attendu 2 (première demande, relance)", len(e.mailer.Messages()))
	}
}

func TestNewsletterConfirmationLinkIsStrict(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	visitor.subscribeToNewsletter("eve@example.org")
	token := e.lastConfirmationToken()

	flipped := []byte(token)
	flipped[len(flipped)/2] ^= 0x01
	for name, bad := range map[string]string{
		"vide": "", "texte arbitraire": "collez-ici-le-jeton-du-lien-recu-par-mail", "falsifié": string(flipped),
		"tronqué": token[:len(token)-3], "jeton d'activation": strings.Repeat("a", 43),
	} {
		if got := visitor.confirmNewsletter(bad); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if row, _ := e.subscription("eve@example.org"); row.Status != "pending" {
		t.Fatalf("aucun lien invalide ne doit confirmer : %+v", row)
	}

	// Le lien expire (48 h par défaut) : une fois expiré, il ne confirme plus.
	e.clock.Advance(49 * time.Hour)
	if got := visitor.confirmNewsletter(token); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
		t.Errorf("lien expiré : %d %s", got.status, got.body)
	}
	if row, _ := e.subscription("eve@example.org"); row.Status != "pending" {
		t.Errorf("un lien expiré ne doit pas confirmer : %+v", row)
	}

	// Une nouvelle demande envoie un lien valide.
	visitor.subscribeToNewsletter("eve@example.org")
	if got := visitor.confirmNewsletter(e.lastConfirmationToken()); got.status != http.StatusNoContent {
		t.Errorf("nouveau lien : %d %s", got.status, got.body)
	}
}

func TestNewsletterConfirmationLinkOfADeletedRequestIsInvalid(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	visitor.subscribeToNewsletter("eve@example.org")
	token := e.lastConfirmationToken()
	if err := e.db.Exec(`DELETE FROM newsletter_subscriptions`).Error; err != nil {
		t.Fatal(err)
	}
	if got := visitor.confirmNewsletter(token); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
		t.Errorf("demande supprimée : %d %s", got.status, got.body)
	}
}

// Quelqu'un qui s'est retiré ne doit pas être réabonné par un vieux lien resté dans sa boîte.
func TestNewsletterOldLinkCannotResubscribeAnUnsubscribedAddress(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	visitor := e.newActor()

	visitor.subscribeToNewsletter("steve@avengers.example")
	oldLink := e.lastConfirmationToken()
	if got := citizen.put("/me/newsletter", map[string]any{"subscribed": false}); got.status != http.StatusNoContent {
		t.Fatal(got.status, string(got.body))
	}
	if row, _ := e.subscription("steve@avengers.example"); row.Status != "unsubscribed" {
		t.Fatalf("après désabonnement : %+v", row)
	}

	if got := visitor.confirmNewsletter(oldLink); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
		t.Errorf("vieux lien : %d %s", got.status, got.body)
	}
	if row, _ := e.subscription("steve@avengers.example"); row.Status != "unsubscribed" {
		t.Errorf("l'adresse a été réabonnée par un vieux lien : %+v", row)
	}
}

func TestFooterSubscriptionAfterUnsubscribingNeedsNewConsent(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	visitor := e.newActor()
	citizen.put("/me/newsletter", map[string]any{"subscribed": true})
	citizen.put("/me/newsletter", map[string]any{"subscribed": false})
	before := len(e.mailer.Messages())

	if got := visitor.subscribeToNewsletter("steve@avengers.example"); got.status != http.StatusAccepted {
		t.Fatalf("nouvelle demande : %d %s", got.status, got.body)
	}
	if len(e.mailer.Messages()) != before+1 {
		t.Error("un abonné qui s'était retiré doit confirmer de nouveau")
	}
	row, _ := e.subscription("steve@avengers.example")
	if row.Status != "pending" || row.Confirmed || row.HasUnsubbed || row.Source != "footer" {
		t.Errorf("après la nouvelle demande : %+v", row)
	}
	if citizen.account(t).NewsletterSubscribed {
		t.Error("tant que le lien n'est pas confirmé, le compte n'est pas abonné")
	}

	visitor.confirmNewsletter(e.lastConfirmationToken())
	if !citizen.account(t).NewsletterSubscribed {
		t.Error("une fois confirmé, le titulaire du compte est abonné (même adresse)")
	}
	// Et il peut s'en retirer depuis son compte.
	citizen.put("/me/newsletter", map[string]any{"subscribed": false})
	if row, _ := e.subscription("steve@avengers.example"); row.Status != "unsubscribed" || !row.HasUnsubbed {
		t.Errorf("après désabonnement : %+v", row)
	}
}

func TestNewsletterSubscriptionValidation(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()

	for email, want := range map[string]string{"": "required", "pas-un-email": "invalid", "eve@": "invalid", strings.Repeat("a", 250) + "@x.io": "too_long"} {
		got := visitor.subscribeToNewsletter(email)
		if got.status != http.StatusUnprocessableEntity || got.fields()["email"] != want {
			t.Errorf("%q : %d %s", email, got.status, got.body)
		}
	}
	if got := visitor.post("/newsletter/subscriptions", map[string]any{"email": "eve@example.org", "extra": 1}); got.status != http.StatusBadRequest {
		t.Errorf("champ inconnu : %d", got.status)
	}
	if got := visitor.request(http.MethodPost, "/newsletter/subscriptions", nil, func(r *http.Request) {
		r.Header.Set("Content-Type", "text/plain")
		r.Body = http.NoBody
	}); got.status != http.StatusUnsupportedMediaType {
		t.Errorf("corps non JSON : %d", got.status)
	}
	if len(e.mailer.Messages()) != 0 || e.count("newsletter_subscriptions", "true") != 0 {
		t.Error("une demande invalide ne doit ni écrire ni envoyer de mail")
	}
}

func TestNewsletterSubscriptionsAreRateLimited(t *testing.T) {
	t.Run("par adresse", func(t *testing.T) {
		e := newE2E(t, map[string]string{"HEROS_RATE_NEWSLETTER_PER_EMAIL": "2/1h"})
		visitor := e.newActor()
		for range 2 {
			if got := visitor.subscribeToNewsletter("victime@example.org"); got.status != http.StatusAccepted {
				t.Fatalf("demande : %d %s", got.status, got.body)
			}
		}
		third := visitor.subscribeToNewsletter("VICTIME@example.org")
		if third.status != http.StatusTooManyRequests || third.code() != "rate_limited" || third.header.Get("Retry-After") == "" {
			t.Errorf("troisième demande : %d %s", third.status, third.body)
		}
		if len(e.mailer.Messages()) != 2 {
			t.Errorf("la boîte d'autrui ne doit pas être inondée : %d mails", len(e.mailer.Messages()))
		}
		if got := visitor.subscribeToNewsletter("autre@example.org"); got.status != http.StatusAccepted {
			t.Errorf("une autre adresse n'est pas concernée : %d", got.status)
		}
	})
	t.Run("par IP", func(t *testing.T) {
		e := newE2E(t, map[string]string{"HEROS_RATE_NEWSLETTER_PER_IP": "2/1h"})
		visitor := e.newActor()
		visitor.subscribeToNewsletter("a@example.org")
		visitor.subscribeToNewsletter("b@example.org")
		if got := visitor.subscribeToNewsletter("c@example.org"); got.status != http.StatusTooManyRequests {
			t.Errorf("troisième demande : %d %s", got.status, got.body)
		}
	})
}

// L'envoi du mail n'est pas ce que le visiteur attend en retour : un échec d'envoi n'annule pas la demande,
// qu'il peut simplement renouveler.
func TestNewsletterMailFailureDoesNotFailTheRequest(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	e.mailer.FailWith(mail.ErrQueueFull)

	if got := visitor.subscribeToNewsletter("eve@example.org"); got.status != http.StatusAccepted {
		t.Fatalf("demande : %d %s", got.status, got.body)
	}
	if row, found := e.subscription("eve@example.org"); !found || row.Status != "pending" {
		t.Errorf("la demande doit être enregistrée : %+v", row)
	}

	e.mailer.FailWith(nil)
	if got := visitor.subscribeToNewsletter("eve@example.org"); got.status != http.StatusAccepted || len(e.mailer.Messages()) != 1 {
		t.Errorf("nouvel essai : %d, %d mails", got.status, len(e.mailer.Messages()))
	}
}

func TestNewsletterConfirmationRequiresAnAllowedOrigin(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	visitor.subscribeToNewsletter("eve@example.org")

	got := visitor.request(http.MethodPost, "/newsletter/subscriptions/confirm", map[string]string{"token": e.lastConfirmationToken()},
		func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") })
	if got.status != http.StatusForbidden || got.code() != "origin_not_allowed" {
		t.Errorf("origine étrangère : %d %s", got.status, got.body)
	}
	if row, _ := e.subscription("eve@example.org"); row.Status != "pending" {
		t.Errorf("une page tierce ne doit pas confirmer à la place du visiteur : %+v", row)
	}
}

// ---- Contact ---------------------------------------------------------------------------------

func contactBody() map[string]string {
	return map[string]string{
		"name": "Natasha Romanoff", "email": "Natasha.Romanoff@Example.org", "subject": "compte",
		"message": "Bonjour,\nje n'arrive plus à me connecter à mon compte.\n\nMerci d'avance !",
	}
}

func TestContactMessageIsForwardedToTheTeamOnly(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_CONTACT_RECIPIENT": "bureau@heros-sos.example"})
	visitor := e.newActor()

	if got := visitor.post("/contact/messages", contactBody()); got.status != http.StatusAccepted || len(got.body) != 0 {
		t.Fatalf("envoi : %d %s", got.status, got.body)
	}
	messages := e.mailer.Messages()
	if len(messages) != 1 {
		t.Fatalf("mails = %d, attendu un seul", len(messages))
	}
	message := messages[0]
	if message.To != "bureau@heros-sos.example" {
		t.Errorf("destinataire = %q : le visiteur ne doit jamais recevoir de mail de cette route", message.To)
	}
	if message.ReplyTo != "natasha.romanoff@example.org" {
		t.Errorf("adresse de réponse = %q", message.ReplyTo)
	}
	if message.Subject != "[Heros-SOS] Message de contact : Problème avec mon compte" {
		t.Errorf("sujet = %q", message.Subject)
	}
	for _, want := range []string{"Natasha Romanoff", "natasha.romanoff@example.org", "je n'arrive plus à me connecter à mon compte.\n\nMerci d'avance !"} {
		if !strings.Contains(message.Text, want) {
			t.Errorf("le texte ne contient pas %q :\n%s", want, message.Text)
		}
	}
	if e.count("audit_logs", "true") != 0 {
		t.Error("un message de contact n'est pas une action sensible : rien dans le journal d'audit")
	}
}

// Le formulaire est ouvert à tous : rien de ce qu'un visiteur saisit ne doit atteindre un en-tête du mail.
func TestContactMessageCannotInjectMailHeaders(t *testing.T) {
	e := newE2E(t, nil)
	body := contactBody()
	body["name"] = "Eve\r\nBcc: espion@evil.example"
	body["message"] = "Un message assez long pour passer.\r\nBcc: espion@evil.example\r\nSubject: piraté"

	if got := e.newActor().post("/contact/messages", body); got.status != http.StatusAccepted {
		t.Fatalf("envoi : %d %s", got.status, got.body)
	}
	message, _ := e.mailer.Last()
	if strings.ContainsAny(message.Subject, "\r\n") || strings.Contains(message.Subject, "Eve") || strings.Contains(message.Subject, "piraté") ||
		strings.ContainsAny(message.To, "\r\n") || strings.ContainsAny(message.ReplyTo, "\r\n") {
		t.Errorf("en-têtes = %q %q %q", message.To, message.ReplyTo, message.Subject)
	}

	// Une adresse qui porte un saut de ligne est refusée avant tout envoi.
	body = contactBody()
	body["email"] = "eve@example.org\r\nBcc: espion@evil.example"
	if got := e.newActor().post("/contact/messages", body); got.status != http.StatusUnprocessableEntity || got.fields()["email"] != "invalid" {
		t.Errorf("adresse piégée : %d %s", got.status, got.body)
	}
	if len(e.mailer.Messages()) != 1 {
		t.Errorf("mails = %d", len(e.mailer.Messages()))
	}
}

func TestContactMessageValidation(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	with := func(field, value string) map[string]string {
		body := contactBody()
		body[field] = value
		return body
	}

	cases := map[string]struct {
		body  map[string]string
		field string
		code  string
	}{
		"nom vide":                {with("name", "   "), "name", "required"},
		"nom trop long":           {with("name", strings.Repeat("n", 81)), "name", "too_long"},
		"email vide":              {with("email", ""), "email", "required"},
		"email invalide":          {with("email", "pas-un-email"), "email", "invalid"},
		"objet absent":            {with("subject", ""), "subject", "required"},
		"objet inconnu":           {with("subject", "spam"), "subject", "invalid"},
		"message vide":            {with("message", ""), "message", "required"},
		"message d'espaces":       {with("message", strings.Repeat(" ", 50)), "message", "required"},
		"message trop court":      {with("message", strings.Repeat("m", 19)), "message", "too_short"},
		"message trop long":       {with("message", strings.Repeat("m", 2001)), "message", "too_long"},
		"message court (espaces)": {with("message", "  "+strings.Repeat("m", 10)+"  "), "message", "too_short"},
	}
	for name, c := range cases {
		got := visitor.post("/contact/messages", c.body)
		if got.status != http.StatusUnprocessableEntity || got.code() != "validation_failed" || got.fields()[c.field] != c.code {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if len(e.mailer.Messages()) != 0 {
		t.Error("un message invalide ne doit rien envoyer")
	}

	// Les bornes elles-mêmes sont acceptées, et tous les objets du formulaire existent.
	for _, subject := range []string{"question", "compte", "donnees", "partenariat", "autre"} {
		if got := visitor.post("/contact/messages", with("subject", subject)); got.status != http.StatusAccepted {
			t.Errorf("objet %q : %d %s", subject, got.status, got.body)
		}
	}
	for name, value := range map[string]string{"20 caractères": strings.Repeat("m", 20), "2000 caractères": strings.Repeat("m", 2000)} {
		if got := visitor.post("/contact/messages", with("message", value)); got.status != http.StatusAccepted {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if got := visitor.post("/contact/messages", map[string]any{"name": "x", "email": "e@e.io", "subject": "autre", "message": strings.Repeat("m", 30), "website": "http://spam"}); got.status != http.StatusBadRequest {
		t.Errorf("champ inconnu : %d", got.status)
	}
}

func TestContactMessagesAreRateLimitedPerIP(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_CONTACT_PER_IP": "2/1h"})
	visitor := e.newActor()

	for range 2 {
		if got := visitor.post("/contact/messages", contactBody()); got.status != http.StatusAccepted {
			t.Fatalf("envoi : %d %s", got.status, got.body)
		}
	}
	third := visitor.post("/contact/messages", contactBody())
	if third.status != http.StatusTooManyRequests || third.code() != "rate_limited" || third.header.Get("Retry-After") == "" {
		t.Errorf("troisième envoi : %d %s", third.status, third.body)
	}
	if len(e.mailer.Messages()) != 2 {
		t.Errorf("mails = %d", len(e.mailer.Messages()))
	}
	e.clock.Advance(time.Hour + time.Second)
	if got := visitor.post("/contact/messages", contactBody()); got.status != http.StatusAccepted {
		t.Errorf("après la fenêtre : %d %s", got.status, got.body)
	}
}

// Le mail est la seule trace d'un message de contact : si l'envoi ne peut pas être pris en charge, le
// visiteur doit le savoir plutôt que croire son message parti.
func TestContactDeliveryFailureIsReportedToTheVisitor(t *testing.T) {
	e := newE2E(t, nil)
	visitor := e.newActor()
	e.mailer.FailWith(errors.Join(mail.ErrQueueFull, errors.New("détail interne")))

	got := visitor.post("/contact/messages", contactBody())
	if got.status != http.StatusServiceUnavailable || got.code() != "contact_delivery_unavailable" {
		t.Fatalf("envoi en échec : %d %s", got.status, got.body)
	}
	if strings.Contains(string(got.body), "détail interne") || strings.Contains(string(got.body), "file") {
		t.Errorf("le détail de la panne ne doit pas fuiter : %s", got.body)
	}

	e.mailer.FailWith(nil)
	if got := visitor.post("/contact/messages", contactBody()); got.status != http.StatusAccepted {
		t.Errorf("après rétablissement : %d %s", got.status, got.body)
	}
}

// ---- Exemples du contrat ---------------------------------------------------------------------

// TestAccountAndMailExamplesOfTheContractAreAcceptedByTheAPI envoie à la vraie application chaque exemple de
// requête que Swagger UI pré-remplit pour ces routes : un exemple que l'API refuserait induirait en erreur
// celui qui l'essaie.
func TestAccountAndMailExamplesOfTheContractAreAcceptedByTheAPI(t *testing.T) {
	e := newE2E(t, nil)
	example := func(operationID string) map[string]any { return e.spec.requestExample(t, operationID) }

	visitor := e.newActor()
	if got := visitor.post("/contact/messages", example("sendContactMessage")); got.status != http.StatusAccepted {
		t.Errorf("sendContactMessage : %d %s", got.status, got.body)
	}
	if got := visitor.post("/newsletter/subscriptions", example("subscribeToNewsletter")); got.status != http.StatusAccepted {
		t.Errorf("subscribeToNewsletter : %d %s", got.status, got.body)
	}
	// Le jeton d'exemple est un texte à remplacer : refusé comme jeton invalide, pas comme corps mal formé.
	if got := visitor.post("/newsletter/subscriptions/confirm", example("confirmNewsletterSubscription")); got.status != http.StatusBadRequest || got.code() != "invalid_token" {
		t.Errorf("confirmNewsletterSubscription : %d %s", got.status, got.body)
	}

	citizen := e.signUpAndLogin("steve@avengers.example")
	if got := citizen.patch("/me", example("updateMyContact")); got.status != http.StatusOK {
		t.Errorf("updateMyContact : %d %s", got.status, got.body)
	}
	if got := citizen.post("/me/email-change-requests", example("requestMyEmailChange")); got.status != http.StatusCreated {
		t.Errorf("requestMyEmailChange : %d %s", got.status, got.body)
	}
	if got := citizen.put("/me/newsletter", example("setMyNewsletterPreference")); got.status != http.StatusNoContent {
		t.Errorf("setMyNewsletterPreference : %d %s", got.status, got.body)
	}
}
