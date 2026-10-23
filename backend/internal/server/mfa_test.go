package server_test

import (
	"encoding/base32"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/totp"
)

// mfaEnv impose la 2FA aux modérateurs, comme en production, et desserre la limite de saisie des codes :
// chaque test qui vérifie une limite la resserre explicitement.
var mfaEnv = map[string]string{
	"HEROS_MFA_MODERATOR_REQUIRED":   "true",
	"HEROS_RATE_MFA_VERIFY_PER_USER": "10000/1h",
}

var recoveryCodeFormat = regexp.MustCompile(`^[A-Z2-7]{4}(-[A-Z2-7]{4}){3}$`)

type mfaSession struct {
	MFA                    string
	CSRFToken              string
	ExpiresAt              time.Time
	RecoveryCodes          []string
	Secret                 string
	OtpauthURL             string
	Status                 string
	RecoveryCodesRemaining int
	User                   struct{ ID, Role, Email string }
}

// totpCode calcule le code que l'application d'authentification afficherait maintenant (horloge de l'application).
func (e *e2e) totpCode(secret string) string {
	e.t.Helper()
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		e.t.Fatalf("secret illisible %q : %v", secret, err)
	}
	return totp.Code(raw, totp.Step(e.clock.Now()))
}

// sessionToken retourne le jeton du cookie de session de l'acteur.
func (a *actor) sessionToken() string {
	a.e.t.Helper()
	base, _ := url.Parse(a.e.http.URL)
	for _, cookie := range a.client.Jar.Cookies(base) {
		if cookie.Name == a.e.cfg.Session.CookieName() {
			return cookie.Value
		}
	}
	return ""
}

func (a *actor) mfaStatus(t *testing.T) mfaSession {
	t.Helper()
	got := a.get("/auth/mfa")
	var status mfaSession
	got.decode(t, &status)
	if got.status != http.StatusOK {
		t.Fatalf("GET /auth/mfa : %d %s", got.status, got.body)
	}
	return status
}

// loginStatus connecte l'acteur et retourne l'état de la double authentification annoncé par la connexion.
func (a *actor) loginStatus(t *testing.T, identifier, password string) (string, response) {
	t.Helper()
	got := a.login(identifier, password)
	var body mfaSession
	if got.status == http.StatusOK {
		got.decode(t, &body)
	}
	return body.MFA, got
}

func (a *actor) setupMFA(t *testing.T) mfaSession {
	t.Helper()
	got := a.post("/auth/mfa/setup", nil)
	var setup mfaSession
	got.decode(t, &setup)
	if got.status != http.StatusOK || setup.Secret == "" {
		t.Fatalf("setup : %d %s", got.status, got.body)
	}
	return setup
}

// enableMFA confirme l'activation avec le code courant et adopte la nouvelle session (jeton CSRF compris).
func (a *actor) enableMFA(t *testing.T, secret string) mfaSession {
	t.Helper()
	got := a.post("/auth/mfa/enable", map[string]string{"code": a.e.totpCode(secret)})
	var enabled mfaSession
	got.decode(t, &enabled)
	if got.status != http.StatusOK || len(enabled.RecoveryCodes) == 0 {
		t.Fatalf("enable : %d %s", got.status, got.body)
	}
	a.csrf = enabled.CSRFToken
	return enabled
}

// enrolledModerator crée un modérateur qui a activé la 2FA, et retourne son secret et ses codes de secours.
func (e *e2e) enrolledModerator(email string) (moderator *actor, secret string, recoveryCodes []string) {
	e.t.Helper()
	moderator = e.newModerator(email)
	secret = moderator.setupMFA(e.t).Secret
	enabled := moderator.enableMFA(e.t, secret)
	e.clock.Advance(totp.Period * time.Second) // le code d'activation a servi : le suivant est celui du pas d'après
	return moderator, secret, enabled.RecoveryCodes
}

// ---- Activation -------------------------------------------------------------------------------

func TestModeratorMustActivateTwoFactorBeforeAnythingElse(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")

	// Obligatoire : la session du modérateur est limitée, celle d'un citoyen ne l'est pas.
	status, _ := e.newActor().loginStatus(t, "fury@shield.example", goodPassword)
	if status != "enrollment_required" {
		t.Errorf("modérateur sans 2FA : mfa = %q", status)
	}
	if status, _ := e.newActor().loginStatus(t, "steve@avengers.example", goodPassword); status != "not_enrolled" {
		t.Errorf("citoyen : mfa = %q", status)
	}

	// Une session limitée ne donne accès à rien d'autre que la 2FA, son état et la déconnexion.
	if got := moderator.get("/auth/me"); got.status != http.StatusOK {
		t.Errorf("/auth/me : %d %s", got.status, got.body)
	} else {
		var me mfaSession
		got.decode(t, &me)
		if me.MFA != "enrollment_required" {
			t.Errorf("/auth/me : mfa = %q", me.MFA)
		}
	}
	if status := moderator.mfaStatus(t); status.Status != "enrollment_required" || status.RecoveryCodesRemaining != 0 {
		t.Errorf("GET /auth/mfa = %+v", status)
	}
	for _, path := range []string{"/admin/users", "/admin/email-change-requests", "/admin/hero-requests", "/me"} {
		if got := moderator.get(path); got.status != http.StatusForbidden || got.code() != "mfa_required" {
			t.Errorf("GET %s avec une session limitée : %d %s", path, got.status, got.body)
		}
	}
	if got := moderator.post("/admin/users/"+citizen.userID()+"/deactivate", nil); got.status != http.StatusForbidden || got.code() != "mfa_required" {
		t.Errorf("écriture avec une session limitée : %d %s", got.status, got.body)
	}

	if got := e.newActor().post("/auth/mfa/setup", nil); got.status != http.StatusUnauthorized {
		t.Errorf("setup sans session : %d", got.status)
	}
}

// La 2FA est obligatoire pour les modérateurs mais ouverte à tous : un citoyen et un super-héros peuvent l'activer, et
// leur connexion demande ensuite un code.
func TestCitizensAndHeroesCanActivateTwoFactor(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator, _, _ := e.enrolledModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	hero := e.approvedHero(moderator, "tony@avengers.example", "Iron Test")

	for name, account := range map[string]struct {
		actor *actor
		email string
	}{
		"citoyen":     {citizen, "steve@avengers.example"},
		"super-héros": {hero, "tony@avengers.example"},
	} {
		// Facultative : la session reste complète tant que la 2FA n'est pas activée.
		if status := account.actor.mfaStatus(t); status.Status != "not_enrolled" {
			t.Fatalf("%s avant activation : %+v", name, status)
		}

		secret := account.actor.setupMFA(t).Secret
		enabled := account.actor.enableMFA(t, secret)
		if len(enabled.RecoveryCodes) == 0 || !recoveryCodeFormat.MatchString(enabled.RecoveryCodes[0]) {
			t.Errorf("%s : codes de secours = %v", name, enabled.RecoveryCodes)
		}
		if status := account.actor.mfaStatus(t); status.Status != "verified" || status.RecoveryCodesRemaining != len(enabled.RecoveryCodes) {
			t.Errorf("%s après activation : %+v", name, status)
		}
		e.clock.Advance(totp.Period * time.Second) // le code d'activation a servi

		// Une 2FA déjà activée ne se remplace pas par l'API, pas même par son propriétaire.
		if got := account.actor.post("/auth/mfa/setup", nil); got.status != http.StatusConflict || got.code() != "mfa_already_enabled" {
			t.Errorf("%s : second setup %d %s", name, got.status, got.body)
		}

		// Les codes de secours se renouvellent contre le code courant.
		got := account.actor.post("/auth/mfa/recovery-codes", map[string]string{"code": e.totpCode(secret)})
		if got.status != http.StatusOK {
			t.Errorf("%s : codes de secours %d %s", name, got.status, got.body)
		}
		e.clock.Advance(totp.Period * time.Second)

		// La connexion suivante est limitée jusqu'à la saisie du code.
		browser := e.newActor()
		status, _ := browser.loginStatus(t, account.email, goodPassword)
		if status != "verification_required" {
			t.Errorf("%s : connexion = %q, attendu verification_required", name, status)
		}
		if got := browser.get("/me"); got.status != http.StatusForbidden || got.code() != "mfa_required" {
			t.Errorf("%s : session limitée sur /me : %d %s", name, got.status, got.body)
		}
		if got := browser.post("/auth/mfa/verify", map[string]string{"code": e.totpCode(secret)}); got.status != http.StatusOK {
			t.Errorf("%s : validation du code %d %s", name, got.status, got.body)
		}
		if got := browser.get("/auth/me"); got.status != http.StatusOK {
			t.Errorf("%s : /auth/me après validation : %d %s", name, got.status, got.body)
		}
		e.clock.Advance(totp.Period * time.Second)
	}
}

func TestActivatingTwoFactorGivesRecoveryCodesAndARotatedSession(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator := e.newModerator("fury@shield.example")
	other := e.newActor() // une autre session du même compte, ouverte avant l'activation
	if status, got := other.loginStatus(t, "fury@shield.example", goodPassword); got.status != http.StatusOK || status != "enrollment_required" {
		t.Fatalf("seconde session : %d %s", got.status, got.body)
	}
	oldToken := moderator.sessionToken()
	oldCSRF := moderator.csrf

	// Sans activation commencée, on ne peut rien confirmer.
	if got := moderator.post("/auth/mfa/enable", map[string]string{"code": "123456"}); got.status != http.StatusConflict || got.code() != "mfa_not_set_up" {
		t.Errorf("enable sans setup : %d %s", got.status, got.body)
	}

	first := moderator.setupMFA(t)
	if !strings.HasPrefix(first.OtpauthURL, "otpauth://totp/") || !strings.Contains(first.OtpauthURL, "secret="+first.Secret) ||
		!strings.Contains(first.OtpauthURL, "issuer=Heros-SOS") || !strings.Contains(first.OtpauthURL, "Heros-SOS:fury@shield.example") {
		t.Errorf("adresse otpauth = %q", first.OtpauthURL)
	}
	if len(first.Secret) != 32 {
		t.Errorf("secret = %q", first.Secret)
	}
	// Recommencer remplace le secret d'une activation non confirmée.
	second := moderator.setupMFA(t)
	if second.Secret == first.Secret {
		t.Error("un nouveau setup doit tirer un nouveau secret")
	}

	// Code vide, faux ou de l'ancien secret : refusé, sans activer.
	for name, code := range map[string]string{"vide": "", "faux": "000000", "ancien secret": e.totpCode(first.Secret), "texte": "abcdef"} {
		got := moderator.post("/auth/mfa/enable", map[string]string{"code": code})
		want := "invalid"
		if code == "" {
			want = "required"
		}
		if got.status != http.StatusUnprocessableEntity || got.fields()["code"] != want {
			t.Errorf("code %s : %d %s", name, got.status, got.body)
		}
	}
	if moderator.mfaStatus(t).Status != "enrollment_required" {
		t.Fatal("un code refusé ne doit pas activer la 2FA")
	}

	// Le bon code active la 2FA, remet les codes de secours et ouvre une session complète neuve.
	enabled := moderator.enableMFA(t, second.Secret)
	if enabled.MFA != "verified" || enabled.User.Role != "moderator" || enabled.User.Email != "fury@shield.example" {
		t.Errorf("activation = %+v", enabled)
	}
	if len(enabled.RecoveryCodes) != e.cfg.Security.MFA.RecoveryCodes {
		t.Fatalf("%d codes de secours, attendu %d", len(enabled.RecoveryCodes), e.cfg.Security.MFA.RecoveryCodes)
	}
	distinct := map[string]bool{}
	for _, code := range enabled.RecoveryCodes {
		distinct[code] = true
		if !recoveryCodeFormat.MatchString(code) {
			t.Errorf("code de secours mal formé : %q", code)
		}
	}
	if len(distinct) != len(enabled.RecoveryCodes) {
		t.Error("les codes de secours doivent être tous différents")
	}
	if enabled.CSRFToken == oldCSRF || moderator.sessionToken() == oldToken || moderator.sessionToken() == "" {
		t.Error("l'activation doit donner un nouveau jeton de session et un nouveau jeton CSRF")
	}

	// L'ancien jeton ne vaut plus rien (anti-fixation), et les autres sessions du compte sont fermées.
	stale := e.newActor()
	stale.client.Jar.SetCookies(mustURL(e.http.URL), []*http.Cookie{{Name: e.cfg.Session.CookieName(), Value: oldToken, Path: "/"}})
	if got := stale.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("l'ancien jeton : %d %s", got.status, got.body)
	}
	if got := other.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("l'autre session du compte doit être fermée : %d %s", got.status, got.body)
	}

	// La nouvelle session est complète.
	if got := moderator.get("/admin/users"); got.status != http.StatusOK {
		t.Errorf("session complète : %d %s", got.status, got.body)
	}
	if status := moderator.mfaStatus(t); status.Status != "verified" || status.RecoveryCodesRemaining != len(enabled.RecoveryCodes) {
		t.Errorf("GET /auth/mfa = %+v", status)
	}
	// Une fois activée, la 2FA ne se réinitialise pas par l'API : un compte piraté ne remplace pas sa protection.
	if got := moderator.post("/auth/mfa/setup", nil); got.status != http.StatusConflict || got.code() != "mfa_already_enabled" {
		t.Errorf("setup après activation : %d %s", got.status, got.body)
	}
	if got := moderator.post("/auth/mfa/enable", map[string]string{"code": e.totpCode(second.Secret)}); got.status != http.StatusConflict || got.code() != "mfa_already_enabled" {
		t.Errorf("enable après activation : %d %s", got.status, got.body)
	}
}

func mustURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}

// ---- Connexion --------------------------------------------------------------------------------

func TestLoginWithTwoFactorNeedsTheCodeAndRotatesTheSession(t *testing.T) {
	e := newE2E(t, mfaEnv)
	_, secret, _ := e.enrolledModerator("fury@shield.example")

	browser := e.newActor()
	status, got := browser.loginStatus(t, "fury@shield.example", goodPassword)
	if got.status != http.StatusOK || status != "verification_required" {
		t.Fatalf("connexion : %d %s", got.status, got.body)
	}
	pendingToken := browser.sessionToken()

	// Le mot de passe seul n'ouvre rien.
	for _, path := range []string{"/admin/users", "/me"} {
		if got := browser.get(path); got.status != http.StatusForbidden || got.code() != "mfa_required" {
			t.Errorf("GET %s avant le code : %d %s", path, got.status, got.body)
		}
	}
	if status := browser.mfaStatus(t); status.Status != "verification_required" || status.RecoveryCodesRemaining == 0 {
		t.Errorf("GET /auth/mfa = %+v", status)
	}

	// Aucun code, ou un code faux : refusé.
	if got := browser.post("/auth/mfa/verify", map[string]string{}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "required" {
		t.Errorf("sans code : %d %s", got.status, got.body)
	}
	if got := browser.post("/auth/mfa/verify", map[string]string{"code": "000000"}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "invalid" {
		t.Errorf("code faux : %d %s", got.status, got.body)
	}

	verified := browser.post("/auth/mfa/verify", map[string]string{"code": e.totpCode(secret)})
	var session mfaSession
	verified.decode(t, &session)
	if verified.status != http.StatusOK || session.MFA != "verified" || session.CSRFToken == "" {
		t.Fatalf("validation : %d %s", verified.status, verified.body)
	}
	browser.csrf = session.CSRFToken
	if browser.sessionToken() == pendingToken {
		t.Error("la validation doit donner un nouveau jeton de session")
	}
	if got := browser.get("/admin/users"); got.status != http.StatusOK {
		t.Errorf("après la validation : %d %s", got.status, got.body)
	}

	// L'ancien jeton (limité) est révoqué : il ne peut pas être rejoué.
	replay := e.newActor()
	replay.client.Jar.SetCookies(mustURL(e.http.URL), []*http.Cookie{{Name: e.cfg.Session.CookieName(), Value: pendingToken, Path: "/"}})
	if got := replay.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("jeton d'avant la validation : %d", got.status)
	}
}

// Un code TOTP ne sert qu'une fois : le code observé par-dessus l'épaule ne peut pas être rejoué.
func TestATotpCodeCannotBeReplayed(t *testing.T) {
	e := newE2E(t, mfaEnv)
	_, secret, _ := e.enrolledModerator("fury@shield.example")

	code := e.totpCode(secret)
	first, second := e.newActor(), e.newActor()
	first.login("fury@shield.example", goodPassword)
	second.login("fury@shield.example", goodPassword)

	if got := first.post("/auth/mfa/verify", map[string]string{"code": code}); got.status != http.StatusOK {
		t.Fatalf("premier usage : %d %s", got.status, got.body)
	}
	if got := second.post("/auth/mfa/verify", map[string]string{"code": code}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "invalid" {
		t.Errorf("rejeu du même code : %d %s", got.status, got.body)
	}
	// Le code du pas suivant, lui, est bon : on ne bloque pas le prochain code légitime.
	e.clock.Advance(totp.Period * time.Second)
	if got := second.post("/auth/mfa/verify", map[string]string{"code": e.totpCode(secret)}); got.status != http.StatusOK {
		t.Errorf("code suivant : %d %s", got.status, got.body)
	}
}

func TestTwoSimultaneousVerificationsWithTheSameCodeSucceedOnce(t *testing.T) {
	// Chaque rejeu compte comme un échec de connexion : le seuil de verrouillage est relevé pour ne pas
	// verrouiller le compte au milieu de la course.
	e := newE2E(t, map[string]string{
		"HEROS_MFA_MODERATOR_REQUIRED": "true", "HEROS_RATE_MFA_VERIFY_PER_USER": "10000/1h", "HEROS_LOCKOUT_MAX_FAILURES": "100",
	})
	_, secret, _ := e.enrolledModerator("fury@shield.example")
	code := e.totpCode(secret)

	const sessions = 6
	browsers := make([]*actor, sessions)
	for i := range browsers {
		browsers[i] = e.newActor()
		if got := browsers[i].login("fury@shield.example", goodPassword); got.status != http.StatusOK {
			t.Fatalf("connexion %d : %d %s", i, got.status, got.body)
		}
	}
	statuses := make(chan int, sessions)
	var wg sync.WaitGroup
	for _, browser := range browsers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- browser.post("/auth/mfa/verify", map[string]string{"code": code}).status
		}()
	}
	wg.Wait()
	close(statuses)

	succeeded, refused := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			succeeded++
		case http.StatusUnprocessableEntity:
			refused++
		default:
			t.Errorf("statut inattendu : %d", status)
		}
	}
	if succeeded != 1 || refused != sessions-1 {
		t.Fatalf("validations = %d, refus = %d : un code ne peut servir qu'une fois, même simultanément", succeeded, refused)
	}
}

func TestRecoveryCodesWorkOnceAndAreNormalized(t *testing.T) {
	e := newE2E(t, mfaEnv)
	_, _, codes := e.enrolledModerator("fury@shield.example")

	// La saisie est tolérante : minuscules, sans tirets.
	typed := strings.ToLower(strings.ReplaceAll(codes[0], "-", ""))
	first := e.newActor()
	first.login("fury@shield.example", goodPassword)
	got := first.post("/auth/mfa/verify", map[string]string{"recoveryCode": typed})
	var session mfaSession
	got.decode(t, &session)
	if got.status != http.StatusOK || session.MFA != "verified" {
		t.Fatalf("code de secours : %d %s", got.status, got.body)
	}
	first.csrf = session.CSRFToken
	if status := first.mfaStatus(t); status.RecoveryCodesRemaining != len(codes)-1 {
		t.Errorf("il doit rester %d codes : %+v", len(codes)-1, status)
	}

	// À usage unique.
	second := e.newActor()
	second.login("fury@shield.example", goodPassword)
	if got := second.post("/auth/mfa/verify", map[string]string{"recoveryCode": codes[0]}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "invalid" {
		t.Errorf("code de secours réutilisé : %d %s", got.status, got.body)
	}
	if got := second.post("/auth/mfa/verify", map[string]string{"recoveryCode": "AAAA-AAAA-AAAA-AAAA"}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("code de secours inventé : %d %s", got.status, got.body)
	}
	// Les autres codes restent valables.
	if got := second.post("/auth/mfa/verify", map[string]string{"recoveryCode": codes[1]}); got.status != http.StatusOK {
		t.Errorf("autre code de secours : %d %s", got.status, got.body)
	}
}

func TestRegeneratingRecoveryCodesNeedsACompleteSessionAndTheCurrentCode(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator, secret, oldCodes := e.enrolledModerator("fury@shield.example")

	if got := moderator.post("/auth/mfa/recovery-codes", map[string]string{"code": "000000"}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "invalid" {
		t.Errorf("code faux : %d %s", got.status, got.body)
	}
	if got := moderator.post("/auth/mfa/recovery-codes", map[string]string{}); got.status != http.StatusUnprocessableEntity || got.fields()["code"] != "required" {
		t.Errorf("sans code : %d %s", got.status, got.body)
	}
	got := moderator.post("/auth/mfa/recovery-codes", map[string]string{"code": e.totpCode(secret)})
	var regenerated mfaSession
	got.decode(t, &regenerated)
	if got.status != http.StatusOK || len(regenerated.RecoveryCodes) != len(oldCodes) {
		t.Fatalf("régénération : %d %s", got.status, got.body)
	}
	// Les anciens codes ne valent plus rien, les nouveaux oui.
	browser := e.newActor()
	browser.login("fury@shield.example", goodPassword)
	if got := browser.post("/auth/mfa/verify", map[string]string{"recoveryCode": oldCodes[0]}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("ancien code de secours : %d %s", got.status, got.body)
	}
	if got := browser.post("/auth/mfa/verify", map[string]string{"recoveryCode": regenerated.RecoveryCodes[0]}); got.status != http.StatusOK {
		t.Errorf("nouveau code de secours : %d %s", got.status, got.body)
	}

	// Une session limitée ne peut pas en réclamer (la route exige une session complète).
	pending := e.newActor()
	pending.login("fury@shield.example", goodPassword)
	e.clock.Advance(totp.Period * time.Second)
	if got := pending.post("/auth/mfa/recovery-codes", map[string]string{"code": e.totpCode(secret)}); got.status != http.StatusForbidden || got.code() != "mfa_required" {
		t.Errorf("session limitée : %d %s", got.status, got.body)
	}
	// Un compte qui n'a pas activé la 2FA n'a pas de codes de secours à renouveler.
	citizen := e.signUpAndLogin("steve@avengers.example")
	if got := citizen.post("/auth/mfa/recovery-codes", map[string]string{"code": "123456"}); got.status != http.StatusConflict || got.code() != "mfa_not_set_up" {
		t.Errorf("citoyen sans 2FA : %d %s", got.status, got.body)
	}
}

// ---- Force brute ------------------------------------------------------------------------------

// Les codes faux comptent comme des échecs de connexion, et le mot de passe seul ne remet pas le compteur à
// zéro : sinon, celui qui connaît le mot de passe alternerait connexion et devinettes sans jamais être verrouillé.
func TestWrongCodesLockTheAccountAndPasswordAloneDoesNotResetTheCount(t *testing.T) {
	e := newE2E(t, mfaEnv)
	_, secret, _ := e.enrolledModerator("fury@shield.example")
	maxFailures := e.cfg.Security.Lockout.MaxFailures

	attacker := e.newActor()
	attacker.login("fury@shield.example", goodPassword)
	for attempt := 1; attempt < maxFailures; attempt++ {
		// À chaque série, une nouvelle connexion par mot de passe : elle ne doit pas effacer les échecs.
		if attempt%2 == 0 {
			attacker = e.newActor()
			if got := attacker.login("fury@shield.example", goodPassword); got.status != http.StatusOK {
				t.Fatalf("reconnexion %d : %d %s", attempt, got.status, got.body)
			}
		}
		if got := attacker.post("/auth/mfa/verify", map[string]string{"code": "000000"}); got.status != http.StatusUnprocessableEntity {
			t.Fatalf("tentative %d : %d %s", attempt, got.status, got.body)
		}
	}
	locking := attacker.post("/auth/mfa/verify", map[string]string{"code": "000000"})
	if locking.status != http.StatusTooManyRequests || locking.code() != "account_locked" || locking.header.Get("Retry-After") == "" {
		t.Fatalf("la %de erreur doit verrouiller le compte : %d %s", maxFailures, locking.status, locking.body)
	}

	// Ses sessions sont fermées, et même le bon mot de passe est refusé tant que le verrou dure.
	if got := attacker.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("la session de l'attaquant doit être fermée : %d", got.status)
	}
	if got := e.newActor().login("fury@shield.example", goodPassword); got.status != http.StatusTooManyRequests || got.code() != "account_locked" {
		t.Errorf("connexion pendant le verrou : %d %s", got.status, got.body)
	}
	if e.count("audit_logs", "action = 'auth.mfa_failed'") != int64(maxFailures) {
		t.Errorf("chaque échec doit être journalisé : %d", e.count("audit_logs", "action = 'auth.mfa_failed'"))
	}

	// Le verrou levé, le bon code rétablit tout.
	e.clock.Advance(e.cfg.Security.Lockout.BaseDelay + time.Minute)
	legit := e.newActor()
	if status, got := legit.loginStatus(t, "fury@shield.example", goodPassword); got.status != http.StatusOK || status != "verification_required" {
		t.Fatalf("connexion après le verrou : %d %s", got.status, got.body)
	}
	if got := legit.post("/auth/mfa/verify", map[string]string{"code": e.totpCode(secret)}); got.status != http.StatusOK {
		t.Fatalf("bon code après le verrou : %d %s", got.status, got.body)
	}
	if e.count("users", "email = 'fury@shield.example' AND failed_login_count = 0") != 1 {
		t.Error("un code juste remet le compteur d'échecs à zéro")
	}
}

func TestVerificationAttemptsAreRateLimitedPerUser(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_MFA_MODERATOR_REQUIRED": "true", "HEROS_RATE_MFA_VERIFY_PER_USER": "3/1h"})
	moderator := e.newModerator("fury@shield.example")
	secret := moderator.setupMFA(t).Secret // 1 appel de la limite

	// setup a consommé une tentative : il en reste deux pour la confirmation, puis la limite tombe.
	for i := range 2 {
		if got := moderator.post("/auth/mfa/enable", map[string]string{"code": "000000"}); got.status != http.StatusUnprocessableEntity {
			t.Fatalf("tentative %d : %d %s", i+1, got.status, got.body)
		}
	}
	limited := moderator.post("/auth/mfa/enable", map[string]string{"code": e.totpCode(secret)})
	if limited.status != http.StatusTooManyRequests || limited.code() != "rate_limited" || limited.header.Get("Retry-After") == "" {
		t.Errorf("après la limite, même le bon code est refusé : %d %s", limited.status, limited.body)
	}
	if moderator.mfaStatus(t).Status != "enrollment_required" {
		t.Error("une tentative limitée ne doit rien activer")
	}
	e.clock.Advance(time.Hour + time.Second)
	// La session (15 min d'inactivité pour un modérateur) a expiré avec l'horloge : on se reconnecte.
	if got := moderator.login("fury@shield.example", goodPassword); got.status != http.StatusOK {
		t.Fatalf("reconnexion : %d %s", got.status, got.body)
	}
}

// ---- Autres garanties -------------------------------------------------------------------------

func TestTwoFactorIsOptionalOutsideProductionAndEnforcedOnceActivated(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_MFA_VERIFY_PER_USER": "10000/1h"}) // 2FA non obligatoire (démonstration)
	moderator := e.newModerator("fury@shield.example")

	if status, _ := e.newActor().loginStatus(t, "fury@shield.example", goodPassword); status != "not_enrolled" {
		t.Errorf("sans obligation : mfa = %q", status)
	}
	if got := moderator.get("/admin/users"); got.status != http.StatusOK {
		t.Errorf("session complète sans 2FA : %d %s", got.status, got.body)
	}

	// L'activation volontaire est possible, et elle s'impose ensuite.
	secret := moderator.setupMFA(t).Secret
	moderator.enableMFA(t, secret)
	if status, _ := e.newActor().loginStatus(t, "fury@shield.example", goodPassword); status != "verification_required" {
		t.Errorf("après activation volontaire : mfa = %q", status)
	}
}

// Une 2FA réinitialisée par l'exploitation prend effet aussitôt, sans attendre l'expiration des sessions :
// une session qui avait passé le code redevient limitée, et le modérateur doit réactiver la 2FA.
func TestResettingTwoFactorLimitsEveryExistingSessionAtOnce(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator, _, _ := e.enrolledModerator("fury@shield.example")
	if got := moderator.get("/admin/users"); got.status != http.StatusOK {
		t.Fatalf("avant la réinitialisation : %d %s", got.status, got.body)
	}

	if err := e.db.Exec(`DELETE FROM user_mfa WHERE user_id = ?`, moderator.userID()).Error; err != nil {
		t.Fatal(err)
	}
	if got := moderator.get("/admin/users"); got.status != http.StatusForbidden || got.code() != "mfa_required" {
		t.Errorf("après la réinitialisation : %d %s", got.status, got.body)
	}
	if status := moderator.mfaStatus(t); status.Status != "enrollment_required" {
		t.Errorf("GET /auth/mfa = %+v", status)
	}
	// Il peut réactiver une 2FA neuve.
	moderator.enableMFA(t, moderator.setupMFA(t).Secret)
	if got := moderator.get("/admin/users"); got.status != http.StatusOK {
		t.Errorf("après la réactivation : %d %s", got.status, got.body)
	}
}

func TestTwoFactorSecretsAreEncryptedAndNeverAudited(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator := e.newModerator("fury@shield.example")
	setup := moderator.setupMFA(t)
	enabled := moderator.enableMFA(t, setup.Secret)
	attacker := e.newActor()
	attacker.login("fury@shield.example", goodPassword)
	attacker.post("/auth/mfa/verify", map[string]string{"code": "000000"})
	attacker.post("/auth/mfa/verify", map[string]string{"recoveryCode": enabled.RecoveryCodes[0]})

	// Le secret est chiffré en base : ni le base32 ni les octets bruts n'y figurent.
	var stored struct{ TotpSecret []byte }
	if err := e.db.Raw(`SELECT totp_secret FROM user_mfa`).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	raw, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(setup.Secret)
	if len(stored.TotpSecret) <= totp.SecretLength || strings.Contains(string(stored.TotpSecret), string(raw)) || strings.Contains(string(stored.TotpSecret), setup.Secret) {
		t.Errorf("le secret doit être chiffré en base (%d octets)", len(stored.TotpSecret))
	}
	// Les codes de secours ne sont gardés que sous forme d'empreinte.
	var hashes []struct{ CodeHash []byte }
	if err := e.db.Raw(`SELECT code_hash FROM mfa_recovery_codes`).Scan(&hashes).Error; err != nil {
		t.Fatal(err)
	}
	if len(hashes) != len(enabled.RecoveryCodes) {
		t.Fatalf("%d empreintes, attendu %d", len(hashes), len(enabled.RecoveryCodes))
	}
	for _, hash := range hashes {
		for _, code := range enabled.RecoveryCodes {
			if strings.Contains(string(hash.CodeHash), strings.ReplaceAll(code, "-", "")) {
				t.Errorf("un code de secours est stocké en clair : %q", code)
			}
		}
	}

	// Le journal dit qui a fait quoi, jamais avec un code ni un secret.
	var entries []struct{ Action, Details string }
	if err := e.db.Raw(`SELECT action, details::text AS details FROM audit_logs WHERE action LIKE 'auth.mfa_%' ORDER BY created_at`).Scan(&entries).Error; err != nil {
		t.Fatal(err)
	}
	actions := map[string]int{}
	for _, entry := range entries {
		actions[entry.Action]++
		for _, secret := range append([]string{setup.Secret, "000000"}, enabled.RecoveryCodes...) {
			if strings.Contains(entry.Details, secret) {
				t.Errorf("%s : le journal contient un secret : %s", entry.Action, entry.Details)
			}
		}
	}
	if actions["auth.mfa_enabled"] != 1 || actions["auth.mfa_failed"] != 1 || actions["auth.mfa_recovery_used"] != 1 {
		t.Errorf("actions journalisées = %v", actions)
	}
}

func TestVerifyWithoutTwoFactorIsRefused(t *testing.T) {
	e := newE2E(t, mfaEnv)
	citizen := e.signUpAndLogin("steve@avengers.example")
	if got := citizen.post("/auth/mfa/verify", map[string]string{"code": "123456"}); got.status != http.StatusConflict || got.code() != "mfa_not_set_up" {
		t.Errorf("citoyen sans 2FA : %d %s", got.status, got.body)
	}
	if got := citizen.get("/auth/mfa"); got.status != http.StatusOK {
		t.Errorf("GET /auth/mfa : %d", got.status)
	} else {
		var status mfaSession
		got.decode(t, &status)
		if status.Status != "not_enrolled" || status.RecoveryCodesRemaining != 0 {
			t.Errorf("état d'un citoyen = %+v", status)
		}
	}
}

func TestLimitedSessionsStillLogOut(t *testing.T) {
	e := newE2E(t, mfaEnv)
	moderator := e.newModerator("fury@shield.example")
	if got := moderator.post("/auth/logout", nil); got.status != http.StatusNoContent {
		t.Errorf("déconnexion d'une session limitée : %d %s", got.status, got.body)
	}
	if got := moderator.get("/auth/me"); got.status != http.StatusUnauthorized {
		t.Errorf("après la déconnexion : %d", got.status)
	}
	again := e.newActor()
	again.login("fury@shield.example", goodPassword)
	if got := again.post("/auth/logout-all", nil); got.status != http.StatusNoContent {
		t.Errorf("déconnexion partout d'une session limitée : %d %s", got.status, got.body)
	}
}
