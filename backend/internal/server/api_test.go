package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
)

// ---- Infrastructure ---------------------------------------------------------------------------

func TestHealthEndpoints(t *testing.T) {
	e := newE2E(t, nil)
	anonymous := e.newActor()

	for _, path := range []string{"/health", "/ready"} {
		got := anonymous.get(path)
		if got.status != http.StatusOK || !strings.Contains(string(got.body), `"status":"ok"`) {
			t.Errorf("%s : %d %s", path, got.status, got.body)
		}
	}
}

func TestUnknownRoutesAndMethodsAnswerInJSON(t *testing.T) {
	e := newE2E(t, nil)
	e.spec = nil // ce test appelle volontairement des routes que le contrat ne décrit pas
	anonymous := e.newActor()

	notFound := anonymous.get("/n-existe-pas")
	if notFound.status != http.StatusNotFound || notFound.code() != "not_found" {
		t.Errorf("route inconnue : %d %s", notFound.status, notFound.body)
	}
	if !strings.HasPrefix(notFound.header.Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q", notFound.header.Get("Content-Type"))
	}

	wrongMethod := anonymous.get("/auth/login")
	if wrongMethod.status != http.StatusMethodNotAllowed || wrongMethod.code() != "method_not_allowed" {
		t.Errorf("mauvaise méthode : %d %s", wrongMethod.status, wrongMethod.body)
	}
	if got := wrongMethod.header.Get("Allow"); !strings.Contains(got, "POST") {
		t.Errorf("Allow = %q, doit annoncer POST", got)
	}
}

func TestEveryResponseCarriesSecurityHeadersAndRequestID(t *testing.T) {
	e := newE2E(t, nil)
	e.spec = nil // inclut une route inconnue (404)
	anonymous := e.newActor()

	for name, got := range map[string]response{
		"succès":     anonymous.get("/health"),
		"erreur 404": anonymous.get("/inconnu"),
		"erreur 401": anonymous.get("/auth/me"),
	} {
		header := got.header
		if header.Get("X-Content-Type-Options") != "nosniff" || header.Get("Cache-Control") != "no-store" ||
			header.Get("Referrer-Policy") != "no-referrer" || header.Get("X-Request-Id") == "" {
			t.Errorf("%s : en-têtes = %v", name, header)
		}
		if header.Get("Strict-Transport-Security") != "" {
			t.Errorf("%s : HSTS ne doit être envoyé qu'en production", name)
		}
	}
}

// ---- Cycle de vie d'un compte ----------------------------------------------------------------

func TestAccountLifecycleOverHTTP(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.newActor()
	const email = "steve@avengers.example"

	// Inscription : 201, et la réponse ne contient ni le mot de passe ni son empreinte.
	registered := citizen.post("/auth/register", registrationBody(email))
	if registered.status != http.StatusCreated {
		t.Fatalf("inscription : %d %s", registered.status, registered.body)
	}
	if strings.Contains(string(registered.body), goodPassword) {
		t.Fatal("la réponse ne doit jamais contenir le mot de passe")
	}
	var created struct {
		User struct {
			ID, DisplayName, Role, Email string
		}
		ActivationRequired bool
	}
	registered.decode(t, &created)
	if created.User.Role != "citizen" || created.User.DisplayName != "Steve" || !created.ActivationRequired {
		t.Errorf("réponse d'inscription = %+v", created)
	}

	// Le même email ne peut pas s'inscrire deux fois (casse ignorée).
	if dup := citizen.post("/auth/register", registrationBody("STEVE@Avengers.example")); dup.status != http.StatusConflict || dup.code() != auth.CodeEmailAlreadyUsed {
		t.Errorf("doublon : %d %s", dup.status, dup.body)
	}

	// Connexion impossible avant activation, et le cookie n'est pas posé.
	early := citizen.login(email, goodPassword)
	if early.status != http.StatusForbidden || early.code() != auth.CodeNotActivated {
		t.Errorf("avant activation : %d %s", early.status, early.body)
	}
	if early.header.Get("Set-Cookie") != "" {
		t.Error("aucun cookie ne doit être posé quand la connexion est refusée")
	}

	if got := citizen.post("/auth/activate", map[string]string{"token": e.lastToken()}); got.status != http.StatusNoContent {
		t.Fatalf("activation : %d %s", got.status, got.body)
	}
	if again := citizen.post("/auth/activate", map[string]string{"token": e.lastToken()}); again.status != http.StatusBadRequest || again.code() != auth.CodeInvalidToken {
		t.Errorf("réactivation : %d %s", again.status, again.body)
	}

	// Connexion : cookie HttpOnly, SameSite=Lax, sans Domain ; jeton CSRF dans la réponse.
	loggedIn := citizen.login(email, goodPassword)
	if loggedIn.status != http.StatusOK {
		t.Fatalf("connexion : %d %s", loggedIn.status, loggedIn.body)
	}
	setCookie := loggedIn.header.Get("Set-Cookie")
	for _, attribute := range []string{"session=", "HttpOnly", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(setCookie, attribute) {
			t.Errorf("Set-Cookie = %q, %q manquant", setCookie, attribute)
		}
	}
	if strings.Contains(setCookie, "Domain=") {
		t.Errorf("le cookie ne doit pas porter d'attribut Domain : %q", setCookie)
	}
	if citizen.csrf == "" {
		t.Fatal("la réponse de connexion doit fournir le jeton CSRF")
	}

	var me struct {
		User      struct{ ID, DisplayName, Role, Email string }
		CSRFToken string
	}
	meResponse := citizen.get("/auth/me")
	if meResponse.status != http.StatusOK {
		t.Fatalf("/auth/me : %d %s", meResponse.status, meResponse.body)
	}
	meResponse.decode(t, &me)
	if me.User.ID != created.User.ID || me.User.Email != email || me.CSRFToken != citizen.csrf {
		t.Errorf("/auth/me = %+v", me)
	}

	// Déconnexion : exige le jeton CSRF ; ensuite la session est fermée.
	forged := citizen.request(http.MethodPost, "/auth/logout", nil, func(r *http.Request) { r.Header.Del(auth.CSRFHeader) })
	if forged.status != http.StatusForbidden || forged.code() != auth.CodeCSRFInvalid {
		t.Errorf("déconnexion sans CSRF : %d %s", forged.status, forged.body)
	}
	if got := citizen.post("/auth/logout", nil); got.status != http.StatusNoContent {
		t.Fatalf("déconnexion : %d %s", got.status, got.body)
	}
	if got := citizen.get("/auth/me"); got.status != http.StatusUnauthorized || got.code() != "unauthenticated" {
		t.Errorf("après déconnexion : %d %s", got.status, got.body)
	}
}

func TestLoginFailuresDoNotRevealWhichAccountsExist(t *testing.T) {
	e := newE2E(t, nil)
	e.signUp("existe@example.org")
	anonymous := e.newActor()

	wrongPassword := anonymous.post("/auth/login", map[string]any{"identifier": "existe@example.org", "password": otherPassword})
	unknownAccount := anonymous.post("/auth/login", map[string]any{"identifier": "inconnu@example.org", "password": otherPassword})

	for name, got := range map[string]response{"mauvais mot de passe": wrongPassword, "compte inconnu": unknownAccount} {
		if got.status != http.StatusUnauthorized || got.code() != auth.CodeInvalidCredentials {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if string(wrongPassword.body) != string(unknownAccount.body) {
		t.Errorf("les deux réponses doivent être identiques :\n%s\n%s", wrongPassword.body, unknownAccount.body)
	}
}

func TestLoginIsRateLimitedPerIdentifier(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_LOGIN_PER_IDENTIFIER": "3/1m"})
	e.signUp("limite@example.org")
	attacker := e.newActor()

	for attempt := 1; attempt <= 3; attempt++ {
		if got := attacker.post("/auth/login", map[string]any{"identifier": "limite@example.org", "password": otherPassword}); got.status != http.StatusUnauthorized {
			t.Fatalf("tentative %d : %d", attempt, got.status)
		}
	}
	limited := attacker.post("/auth/login", map[string]any{"identifier": "LIMITE@example.org", "password": goodPassword})
	if limited.status != http.StatusTooManyRequests || limited.code() != "rate_limited" {
		t.Fatalf("la quatrième tentative doit être limitée (casse ignorée) : %d %s", limited.status, limited.body)
	}
	if limited.header.Get("Retry-After") == "" {
		t.Error("Retry-After doit indiquer quand réessayer")
	}

	e.clock.Advance(2 * time.Minute) // l'horloge du limiteur est celle de l'application
	if got := attacker.post("/auth/login", map[string]any{"identifier": "limite@example.org", "password": goodPassword}); got.status != http.StatusOK {
		t.Errorf("après la fenêtre, la connexion doit passer : %d %s", got.status, got.body)
	}
}

func TestLoginResetsItsLimiterAfterSuccess(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_LOGIN_PER_IDENTIFIER": "3/1h"})
	e.signUp("reset-limite@example.org")
	user := e.newActor()

	for range 5 {
		if got := user.login("reset-limite@example.org", goodPassword); got.status != http.StatusOK {
			t.Fatalf("une connexion réussie ne doit pas épuiser le quota : %d %s", got.status, got.body)
		}
	}
}

func TestAccountLockoutIsAnnouncedWithRetryAfter(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_LOCKOUT_MAX_FAILURES": "2", "HEROS_LOCKOUT_BASE_DELAY": "5m"})
	e.signUp("verrou@example.org")
	user := e.newActor()

	for range 2 {
		user.login("verrou@example.org", otherPassword)
	}
	locked := user.login("verrou@example.org", goodPassword)
	if locked.status != http.StatusTooManyRequests || locked.code() != auth.CodeAccountLocked {
		t.Fatalf("%d %s", locked.status, locked.body)
	}
	if retry := locked.header.Get("Retry-After"); retry != "300" {
		t.Errorf("Retry-After = %q, attendu 300", retry)
	}
}

// ---- Protection des requêtes -------------------------------------------------------------------

func TestOriginCheckBlocksCrossSiteWrites(t *testing.T) {
	e := newE2E(t, nil)
	anonymous := e.newActor()
	body := map[string]any{"identifier": "a@b.co", "password": goodPassword}

	tests := map[string]struct {
		mutate func(*http.Request)
		want   int
	}{
		"origine étrangère":     {func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
		"origine null":          {func(r *http.Request) { r.Header.Set("Origin", "null") }, http.StatusForbidden},
		"origine mal formée":    {func(r *http.Request) { r.Header.Set("Origin", "pas une origine") }, http.StatusForbidden},
		"sous-domaine voisin":   {func(r *http.Request) { r.Header.Set("Origin", "http://localhost:3000.evil.example") }, http.StatusForbidden},
		"requête intersite":     {func(r *http.Request) { r.Header.Del("Origin"); r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		"origine du frontend":   {func(r *http.Request) {}, http.StatusUnauthorized},
		"serveur à serveur":     {func(r *http.Request) { r.Header.Del("Origin") }, http.StatusUnauthorized},
		"même origine déclarée": {func(r *http.Request) { r.Header.Del("Origin"); r.Header.Set("Sec-Fetch-Site", "same-origin") }, http.StatusUnauthorized},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := anonymous.request(http.MethodPost, "/auth/login", body, test.mutate)
			if got.status != test.want {
				t.Fatalf("statut = %d, attendu %d (%s)", got.status, test.want, got.body)
			}
			if test.want == http.StatusForbidden && got.code() != auth.CodeOriginNotAllowed {
				t.Errorf("code = %q, attendu %q", got.code(), auth.CodeOriginNotAllowed)
			}
		})
	}
}

func TestAuthenticatedWritesRequireTheCSRFToken(t *testing.T) {
	e := newE2E(t, nil)
	victim := e.signUpAndLogin("victime@example.org")
	other := e.signUpAndLogin("autre@example.org")
	payload := map[string]string{"alias": "Faucon", "bio": "Il vole", "requestMessage": "Candidat"}

	tests := map[string]string{
		"sans jeton":                "",
		"jeton inventé":             "inventé",
		"jeton d'une autre session": other.csrf,
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			got := victim.request(http.MethodPost, "/heroes/requests", payload, func(r *http.Request) {
				r.Header.Del(auth.CSRFHeader)
				if token != "" {
					r.Header.Set(auth.CSRFHeader, token)
				}
			})
			if got.status != http.StatusForbidden || got.code() != auth.CodeCSRFInvalid {
				t.Fatalf("%d %s", got.status, got.body)
			}
		})
	}

	if got := victim.post("/heroes/requests", payload); got.status != http.StatusCreated {
		t.Fatalf("avec le bon jeton : %d %s", got.status, got.body)
	}
	if e.count("hero_profiles", "true") != 1 {
		t.Error("seule la requête valide doit avoir écrit en base")
	}
}

func TestRequestBodyIsValidatedStrictly(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_HTTP_MAX_BODY_BYTES": "512"})
	anonymous := e.newActor()

	t.Run("champ inconnu", func(t *testing.T) {
		body := registrationBody("a@b.co")
		body["role"] = "moderator" // tentative d'élévation de privilèges
		if got := anonymous.post("/auth/register", body); got.status != http.StatusBadRequest {
			t.Fatalf("%d %s", got.status, got.body)
		}
		if e.count("users", "true") != 0 {
			t.Fatal("aucun compte ne doit être créé")
		}
	})
	t.Run("type de contenu", func(t *testing.T) {
		got := anonymous.request(http.MethodPost, "/auth/login", map[string]any{"identifier": "a", "password": "b"},
			func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") })
		if got.status != http.StatusUnsupportedMediaType {
			t.Fatalf("%d %s", got.status, got.body)
		}
	})
	t.Run("corps trop volumineux", func(t *testing.T) {
		got := anonymous.post("/auth/login", map[string]any{"identifier": "a@b.co", "password": strings.Repeat("x", 2000)})
		if got.status != http.StatusRequestEntityTooLarge {
			t.Fatalf("%d %s", got.status, got.body)
		}
	})
	t.Run("JSON invalide", func(t *testing.T) {
		got := anonymous.request(http.MethodPost, "/auth/login", nil, func(r *http.Request) {
			r.Header.Set("Content-Type", "application/json")
			r.Body = http.NoBody
		})
		if got.status != http.StatusBadRequest {
			t.Fatalf("%d %s", got.status, got.body)
		}
	})
}

func TestRegistrationValidationReportsEveryField(t *testing.T) {
	e := newE2E(t, nil)
	anonymous := e.newActor()

	got := anonymous.post("/auth/register", map[string]any{
		"civility": "x", "firstName": "", "lastName": "", "birthDate": "demain", "email": "pas-un-email",
		"mobilePhone": "1", "password": "faible", "address": map[string]any{"street": "", "postalCode": "", "city": "", "country": ""},
	})
	if got.status != http.StatusUnprocessableEntity || got.code() != "validation_failed" {
		t.Fatalf("%d %s", got.status, got.body)
	}
	for _, field := range []string{"civility", "firstName", "lastName", "birthDate", "email", "mobilePhone", "password", "address.street", "address.city", "address.country", "address.postalCode"} {
		if got.fields()[field] == "" {
			t.Errorf("champ %q non signalé : %v", field, got.fields())
		}
	}
}

// ---- Mot de passe oublié -----------------------------------------------------------------------

func TestPasswordResetOverHTTP(t *testing.T) {
	e := newE2E(t, nil)
	user := e.signUpAndLogin("oubli@example.org")
	anonymous := e.newActor()

	// La réponse est la même pour un compte existant et inconnu.
	for _, email := range []string{"oubli@example.org", "fantome@example.org"} {
		if got := anonymous.post("/auth/password-reset", map[string]string{"email": email}); got.status != http.StatusAccepted {
			t.Fatalf("%s : %d %s", email, got.status, got.body)
		}
	}
	if messages := e.mailer.Messages(); len(messages) != 2 { // activation + un seul mail de réinitialisation
		t.Fatalf("mails = %d, attendu 2 : seul le compte existant reçoit un lien", len(messages))
	}

	token := e.lastToken()
	weak := anonymous.post("/auth/password-reset/confirm", map[string]string{"token": token, "newPassword": "faible"})
	if weak.status != http.StatusUnprocessableEntity || weak.fields()["newPassword"] == "" {
		t.Fatalf("mot de passe faible : %d %s", weak.status, weak.body)
	}
	if got := anonymous.post("/auth/password-reset/confirm", map[string]string{"token": token, "newPassword": otherPassword}); got.status != http.StatusNoContent {
		t.Fatalf("confirmation : %d %s", got.status, got.body)
	}

	if got := user.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("l'ancienne session doit être fermée : %d", got.status)
	}
	if got := anonymous.login("oubli@example.org", goodPassword); got.status != http.StatusUnauthorized {
		t.Errorf("ancien mot de passe : %d", got.status)
	}
	if got := anonymous.login("oubli@example.org", otherPassword); got.status != http.StatusOK {
		t.Errorf("nouveau mot de passe : %d %s", got.status, got.body)
	}
	if got := anonymous.post("/auth/password-reset/confirm", map[string]string{"token": token, "newPassword": goodPassword}); got.status != http.StatusBadRequest || got.code() != auth.CodeInvalidToken {
		t.Errorf("lien réutilisé : %d %s", got.status, got.body)
	}
}

func TestPasswordResetRequestsAreRateLimitedPerAddress(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_PASSWORD_RESET_PER_EMAIL": "2/1h"})
	e.signUp("spam@example.org")
	anonymous := e.newActor()
	before := len(e.mailer.Messages())

	for range 2 {
		if got := anonymous.post("/auth/password-reset", map[string]string{"email": "spam@example.org"}); got.status != http.StatusAccepted {
			t.Fatalf("%d", got.status)
		}
	}
	if got := anonymous.post("/auth/password-reset", map[string]string{"email": "Spam@Example.org"}); got.status != http.StatusTooManyRequests {
		t.Fatalf("la troisième demande doit être limitée : %d", got.status)
	}
	if sent := len(e.mailer.Messages()) - before; sent != 2 {
		t.Errorf("mails envoyés = %d, attendu 2 : le mail-bombing doit être borné", sent)
	}
}

// ---- Demande « Devenir super-héros » et modération ----------------------------------------------

func TestHeroRequestApprovalFlow(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("tony@stark.io")
	moderator := e.newModerator("fury@shield.example")
	request := map[string]string{"alias": "Iron Man", "bio": "Génie, milliardaire", "requestMessage": "J'ai une armure"}

	if got := citizen.get("/heroes/requests/me"); got.status != http.StatusNotFound || got.code() != "hero_request_not_found" {
		t.Errorf("avant toute demande : %d %s", got.status, got.body)
	}
	if got := citizen.post("/heroes/requests", request); got.status != http.StatusCreated {
		t.Fatalf("demande : %d %s", got.status, got.body)
	}
	if got := citizen.post("/heroes/requests", request); got.status != http.StatusConflict || got.code() != "hero_request_pending" {
		t.Errorf("doublon : %d %s", got.status, got.body)
	}
	// Tant que la demande est en attente, le citoyen n'a aucun pouvoir de héros.
	var pending struct{ Status string }
	citizen.get("/heroes/requests/me").decode(t, &pending)
	if pending.Status != "pending" {
		t.Errorf("état = %q", pending.Status)
	}
	if got := citizen.get("/admin/hero-requests"); got.status != http.StatusForbidden {
		t.Errorf("un citoyen ne voit pas les demandes : %d", got.status)
	}

	// Le modérateur voit la demande avec l'identité du demandeur (déchiffrée).
	list := moderator.get("/admin/hero-requests?status=pending")
	var page struct {
		Items []struct {
			UserID, Alias, Email, FirstName, LastName, Status string
		}
		Total int
	}
	list.decode(t, &page)
	if list.status != http.StatusOK || page.Total != 1 || page.Items[0].Alias != "Iron Man" ||
		page.Items[0].Email != "tony@stark.io" || page.Items[0].FirstName != "Steve" {
		t.Fatalf("liste : %d %s", list.status, list.body)
	}

	approve := moderator.post("/admin/hero-requests/"+page.Items[0].UserID+"/approve", nil)
	if approve.status != http.StatusNoContent {
		t.Fatalf("validation : %d %s", approve.status, approve.body)
	}
	if again := moderator.post("/admin/hero-requests/"+page.Items[0].UserID+"/approve", nil); again.status != http.StatusConflict || again.code() != "hero_request_not_pending" {
		t.Errorf("seconde validation : %d %s", again.status, again.body)
	}

	// Le rôle change immédiatement, sans reconnexion ; le pseudo devient le nom affiché.
	var me struct {
		User struct{ Role, DisplayName string }
	}
	citizen.get("/auth/me").decode(t, &me)
	if me.User.Role != "hero" || me.User.DisplayName != "Iron Man" {
		t.Errorf("après validation : %+v", me.User)
	}
	if got := citizen.post("/heroes/requests", request); got.status != http.StatusConflict || got.code() != "already_hero" {
		t.Errorf("un héros ne redemande pas : %d %s", got.status, got.body)
	}
	if got := moderator.post("/heroes/requests", request); got.status != http.StatusForbidden || got.code() != "hero_request_not_eligible" {
		t.Errorf("un modérateur ne peut pas demander à devenir héros : %d %s", got.status, got.body)
	}

	// Le héros se connecte aussi par son pseudo.
	byAlias := e.newActor()
	if got := byAlias.login("iron man", goodPassword); got.status != http.StatusOK {
		t.Errorf("connexion par pseudo : %d %s", got.status, got.body)
	}
	if e.count("audit_logs", "action IN ('hero.request','hero.approve')") != 2 {
		t.Error("la demande et la validation doivent être journalisées")
	}
}

func TestHeroRequestRejectionAndResubmission(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("candidat@example.org")
	moderator := e.newModerator("fury@shield.example")
	userID := citizen.userID()

	citizen.post("/heroes/requests", map[string]string{"alias": "Super Menteur", "requestMessage": "Je vole"})
	reject := moderator.post("/admin/hero-requests/"+userID+"/reject", map[string]string{"reason": "Preuves insuffisantes"})
	if reject.status != http.StatusNoContent {
		t.Fatalf("refus : %d %s", reject.status, reject.body)
	}

	var status struct{ Status, RejectionReason string }
	citizen.get("/heroes/requests/me").decode(t, &status)
	if status.Status != "rejected" || status.RejectionReason != "Preuves insuffisantes" {
		t.Errorf("état après refus = %+v", status)
	}
	var me struct{ User struct{ Role string } }
	citizen.get("/auth/me").decode(t, &me)
	if me.User.Role != "citizen" {
		t.Errorf("un refus ne donne aucun pouvoir : rôle = %q", me.User.Role)
	}

	// Une demande refusée peut être redéposée, et repart en attente.
	if got := citizen.post("/heroes/requests", map[string]string{"alias": "Vrai Héros", "requestMessage": "Voici mes preuves"}); got.status != http.StatusCreated {
		t.Fatalf("nouvelle demande : %d %s", got.status, got.body)
	}
	status = struct{ Status, RejectionReason string }{} // omitempty : un champ absent ne remet pas la valeur à zéro
	citizen.get("/heroes/requests/me").decode(t, &status)
	if status.Status != "pending" || status.RejectionReason != "" {
		t.Errorf("état après nouvelle demande = %+v", status)
	}
}

func TestHeroAliasIsUniqueIgnoringCase(t *testing.T) {
	e := newE2E(t, nil)
	first := e.signUpAndLogin("premier@example.org")
	second := e.signUpAndLogin("second@example.org")

	if got := first.post("/heroes/requests", map[string]string{"alias": "Hawkeye"}); got.status != http.StatusCreated {
		t.Fatalf("%d %s", got.status, got.body)
	}
	if got := second.post("/heroes/requests", map[string]string{"alias": "HAWKEYE"}); got.status != http.StatusConflict || got.code() != "alias_already_used" {
		t.Fatalf("%d %s", got.status, got.body)
	}
	if got := second.post("/heroes/requests", map[string]string{"alias": "x"}); got.status != http.StatusUnprocessableEntity || got.fields()["alias"] == "" {
		t.Fatalf("pseudo trop court : %d %s", got.status, got.body)
	}
}

func TestAdminRoutesRequireAModerator(t *testing.T) {
	e := newE2E(t, nil)
	anonymous := e.newActor()
	citizen := e.signUpAndLogin("simple@example.org")
	someone := "00000000-0000-7000-8000-000000000001"

	routes := []struct{ method, path string }{
		{http.MethodGet, "/admin/hero-requests"},
		{http.MethodPost, "/admin/hero-requests/" + someone + "/approve"},
		{http.MethodPost, "/admin/hero-requests/" + someone + "/reject"},
		{http.MethodGet, "/admin/users"},
		{http.MethodGet, "/admin/moderation/summary"},
		{http.MethodPost, "/admin/users/" + someone + "/deactivate"},
		{http.MethodPost, "/admin/users/" + someone + "/reactivate"},
		{http.MethodPost, "/admin/users/" + someone + "/verify"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			if got := anonymous.request(route.method, route.path, map[string]string{}); got.status != http.StatusUnauthorized {
				t.Errorf("sans session : %d, attendu 401", got.status)
			}
			if got := citizen.request(route.method, route.path, map[string]string{}); got.status != http.StatusForbidden {
				t.Errorf("citoyen : %d, attendu 403", got.status)
			}
		})
	}
}

func TestModeratorSummaryCountsPendingItems(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	type summary struct {
		Incidents, HeroApplications, Videos, Comments, EmailChanges, DraftArticles, DraftNewsletters int64
	}
	var before summary
	moderator.get("/admin/moderation/summary").decode(t, &before)
	if before != (summary{}) {
		t.Fatalf("base vide : %+v", before)
	}

	applicant := e.signUpAndLogin("candidat@example.org")
	if got := applicant.post("/heroes/requests", map[string]string{"alias": "Hawkeye"}); got.status != http.StatusCreated {
		t.Fatalf("%d %s", got.status, got.body)
	}
	var after summary
	moderator.get("/admin/moderation/summary").decode(t, &after)
	if after != (summary{HeroApplications: 1}) {
		t.Fatalf("après une demande de héros : %+v", after)
	}

	var draft struct{ ID string }
	moderator.post("/admin/newsletters", map[string]any{"subject": "À envoyer", "body": "<p>Brouillon</p>"}).decode(t, &draft)
	moderator.get("/admin/moderation/summary").decode(t, &after)
	if after != (summary{HeroApplications: 1, DraftNewsletters: 1}) {
		t.Fatalf("après un brouillon de newsletter : %+v", after)
	}
	moderator.request(http.MethodDelete, "/admin/newsletters/"+draft.ID, nil)
	moderator.get("/admin/moderation/summary").decode(t, &after)
	if after.DraftNewsletters != 0 {
		t.Fatalf("après suppression du brouillon : %+v", after)
	}
}

func TestModeratorManagesAccounts(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	victim := e.signUpAndLogin("banni@example.org")
	victimID := victim.userID()

	// Liste filtrée.
	var page struct {
		Items []struct{ ID, Email, Role, Status string }
		Total int
	}
	moderator.get("/admin/users?role=citizen&email=BANNI").decode(t, &page)
	if page.Total != 1 || page.Items[0].ID != victimID {
		t.Fatalf("liste filtrée = %+v", page)
	}
	if got := moderator.get("/admin/users?role=roi"); got.status != http.StatusUnprocessableEntity {
		t.Errorf("rôle invalide : %d", got.status)
	}
	if got := moderator.get("/admin/users?limit=5000"); got.status != http.StatusUnprocessableEntity {
		t.Errorf("limite démesurée : %d", got.status)
	}

	if got := moderator.post("/admin/users/"+victimID+"/verify", nil); got.status != http.StatusNoContent {
		t.Fatalf("vérification : %d %s", got.status, got.body)
	}

	// La désactivation coupe la session ouverte immédiatement et interdit toute reconnexion.
	if got := moderator.post("/admin/users/"+victimID+"/deactivate", nil); got.status != http.StatusNoContent {
		t.Fatalf("désactivation : %d %s", got.status, got.body)
	}
	if got := victim.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("la session d'un compte désactivé doit être coupée : %d", got.status)
	}
	if got := e.newActor().login("banni@example.org", goodPassword); got.status != http.StatusForbidden || got.code() != auth.CodeDeactivated {
		t.Errorf("reconnexion : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/users/"+victimID+"/deactivate", nil); got.status != http.StatusNoContent {
		t.Errorf("la désactivation est idempotente : %d", got.status)
	}

	if got := moderator.post("/admin/users/"+victimID+"/reactivate", nil); got.status != http.StatusNoContent {
		t.Fatalf("réactivation : %d %s", got.status, got.body)
	}
	if got := e.newActor().login("banni@example.org", goodPassword); got.status != http.StatusOK {
		t.Errorf("après réactivation : %d %s", got.status, got.body)
	}

	// Garde-fous.
	moderatorID := moderator.userID()
	if got := moderator.post("/admin/users/"+moderatorID+"/deactivate", nil); got.status != http.StatusConflict || got.code() != "self_action_forbidden" {
		t.Errorf("auto-désactivation : %d %s", got.status, got.body)
	}
	other := e.newModerator("coulson@shield.example")
	if got := moderator.post("/admin/users/"+other.userID()+"/deactivate", nil); got.status != http.StatusForbidden || got.code() != "protected_account" {
		t.Errorf("modérateur protégé : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/users/inconnu/deactivate", nil); got.status != http.StatusNotFound {
		t.Errorf("identifiant invalide : %d", got.status)
	}
	if got := moderator.post("/admin/users/00000000-0000-7000-8000-000000000099/deactivate", nil); got.status != http.StatusNotFound {
		t.Errorf("utilisateur inconnu : %d", got.status)
	}

	if e.count("audit_logs", "action = 'user.deactivate'") != 1 || e.count("audit_logs", "action = 'user.verify'") != 1 || e.count("audit_logs", "action = 'user.reactivate'") != 1 {
		t.Error("chaque action de modération effective doit être journalisée une seule fois (les rejeux idempotents n'écrivent rien)")
	}
}

func TestLogoutAllClosesEverySessionOverHTTP(t *testing.T) {
	e := newE2E(t, nil)
	first := e.signUpAndLogin("multi@example.org")
	second := e.newActor()
	second.login("multi@example.org", goodPassword)

	if got := first.post("/auth/logout-all", nil); got.status != http.StatusNoContent {
		t.Fatalf("%d %s", got.status, got.body)
	}
	for name, a := range map[string]*actor{"première": first, "seconde": second} {
		if got := a.get("/auth/me"); got.status != http.StatusUnauthorized {
			t.Errorf("la %s session doit être fermée : %d", name, got.status)
		}
	}
}

func TestExpiredSessionCookieIsClearedByTheServer(t *testing.T) {
	e := newE2E(t, nil)
	user := e.signUpAndLogin("expire@example.org")

	e.clock.Advance(e.cfg.Session.Member.IdleTimeout + time.Minute)
	got := user.get("/auth/me")
	if got.status != http.StatusUnauthorized {
		t.Fatalf("session inactive : %d", got.status)
	}
	if cleared := got.header.Get("Set-Cookie"); !strings.Contains(cleared, "Max-Age=0") && !strings.Contains(cleared, "Max-Age=-1") && !strings.Contains(cleared, "1970") {
		t.Errorf("le cookie périmé doit être effacé : %q", cleared)
	}
}

func TestStaleSessionCookieDoesNotBlockANewLogin(t *testing.T) {
	e := newE2E(t, nil)
	user := e.signUpAndLogin("revient@example.org")
	e.clock.Advance(e.cfg.Session.Member.IdleTimeout + time.Minute)

	// Le navigateur renvoie encore l'ancien cookie, et aucun jeton CSRF valide : la connexion
	// doit malgré tout aboutir.
	user.csrf = ""
	if got := user.login("revient@example.org", goodPassword); got.status != http.StatusOK {
		t.Fatalf("%d %s", got.status, got.body)
	}
	if got := user.get("/auth/me"); got.status != http.StatusOK {
		t.Errorf("la nouvelle session doit fonctionner : %d", got.status)
	}
}

// ---- Concurrence -------------------------------------------------------------------------------

func TestConcurrentRequestsAcrossSessionsAreIsolated(t *testing.T) {
	e := newE2E(t, nil)
	const users = 8
	actors := make([]*actor, users)
	for i := range users {
		actors[i] = e.signUpAndLogin(fmt.Sprintf("citoyen%d@example.org", i))
	}

	var wg sync.WaitGroup
	for i, a := range actors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				got := a.get("/auth/me")
				if got.status != http.StatusOK || !strings.Contains(string(got.body), fmt.Sprintf("citoyen%d@example.org", i)) {
					t.Errorf("acteur %d : %d %s", i, got.status, got.body)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentRegistrationOverHTTPCreatesOneAccount(t *testing.T) {
	e := newE2E(t, nil)

	const attempts = 10
	statuses := make(chan int, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- e.newActor().post("/auth/register", registrationBody("course@example.org")).status
		}()
	}
	wg.Wait()
	close(statuses)

	created, conflicts := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("statut inattendu : %d", status)
		}
	}
	if created != 1 || conflicts != attempts-1 {
		t.Fatalf("créés = %d, conflits = %d", created, conflicts)
	}
}
