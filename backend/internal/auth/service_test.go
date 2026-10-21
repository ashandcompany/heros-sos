package auth_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

var ctx = context.Background()

// ---- Inscription ----------------------------------------------------------------------------

func TestRegisterCreatesPendingCitizenAndSendsActivationMail(t *testing.T) {
	h := newHarness(t, nil)

	created, err := h.service.Register(ctx, registration("Steve@Avengers.example"))
	if err != nil {
		t.Fatal(err)
	}

	stored, err := h.users.FindByEmail(ctx, "steve@avengers.example")
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != created.ID || stored.Role != user.RoleCitizen || stored.Status != user.StatusPendingActivation {
		t.Errorf("compte = %+v", stored)
	}
	if stored.Phone != "0612345678" || stored.Address.Country != "US" {
		t.Errorf("les données doivent être normalisées : téléphone %q, pays %q", stored.Phone, stored.Address.Country)
	}
	if strings.Contains(stored.PasswordHash, goodPassword) || !strings.HasPrefix(stored.PasswordHash, "$argon2id$") {
		t.Errorf("le mot de passe doit être stocké haché : %q", stored.PasswordHash)
	}

	messages := h.mailer.Messages()
	if len(messages) != 1 || messages[0].To != "steve@avengers.example" {
		t.Fatalf("mails = %+v", messages)
	}
	if !strings.Contains(messages[0].Text, "http://localhost:3000/activation?token=") {
		t.Errorf("le mail doit contenir le lien d'activation :\n%s", messages[0].Text)
	}
	if h.count("user_tokens", "user_id = ? AND type = 'activation'", stored.ID) != 1 {
		t.Error("un jeton d'activation doit être enregistré")
	}
}

func TestRegisterReportsAllInvalidFieldsAndCreatesNothing(t *testing.T) {
	h := newHarness(t, nil)
	invalid := auth.RegisterInput{
		Civility: "empereur", FirstName: "", LastName: "4chan", BirthDate: "2999-01-01",
		Email: "pas-un-email", Phone: "123", Password: "faible",
		Address: user.Address{Street: "", PostalCode: "ABCDEFGHIJKLMN", City: "", Country: "FRA"},
	}

	_, err := h.service.Register(ctx, invalid)

	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("erreur = %v, attendu *validation.Error", err)
	}
	for _, field := range []string{
		"civility", "firstName", "lastName", "birthDate", "email", "mobilePhone", "password",
		"address.street", "address.city", "address.country", "address.postalCode",
	} {
		if validationErr.Fields[field] == "" {
			t.Errorf("le champ %s devrait être signalé : %v", field, validationErr.Fields)
		}
	}
	if h.count("users", "true") != 0 || len(h.mailer.Messages()) != 0 {
		t.Error("une inscription invalide ne doit ni créer de compte ni envoyer de mail")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("tony@stark.io")
	before := len(h.mailer.Messages())

	_, err := h.service.Register(ctx, registration("TONY@stark.io"))

	if !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("erreur = %v, attendu ErrEmailTaken", err)
	}
	if len(h.mailer.Messages()) != before {
		t.Error("aucun mail ne doit partir pour une inscription refusée")
	}
	if h.count("user_tokens", "true") != 1 {
		t.Error("l'inscription refusée ne doit laisser aucun jeton orphelin")
	}
}

func TestConcurrentRegistrationsOfSameEmailCreateOneAccountAndOneMail(t *testing.T) {
	h := newHarness(t, nil)

	const attempts = 12
	var wg sync.WaitGroup
	var succeeded, taken atomic.Int32
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch _, err := h.service.Register(ctx, registration("course@example.org")); {
			case err == nil:
				succeeded.Add(1)
			case errors.Is(err, user.ErrEmailTaken):
				taken.Add(1)
			default:
				t.Errorf("erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded.Load() != 1 || taken.Load() != attempts-1 {
		t.Fatalf("inscriptions réussies = %d, refusées = %d", succeeded.Load(), taken.Load())
	}
	if h.count("users", "true") != 1 || h.count("user_tokens", "true") != 1 || len(h.mailer.Messages()) != 1 {
		t.Error("un seul compte, un seul jeton et un seul mail attendus")
	}
}

func TestRegisterSucceedsEvenIfMailCannotBeQueued(t *testing.T) {
	h := newHarness(t, nil)
	h.mailer.FailWith(mail.ErrQueueFull)

	if _, err := h.service.Register(ctx, registration("file-pleine@example.org")); err != nil {
		t.Fatalf("l'inscription ne doit pas échouer à cause du mail : %v", err)
	}
	if h.count("users", "email = 'file-pleine@example.org'") != 1 {
		t.Fatal("le compte doit exister")
	}

	// Une fois le mail rétabli, l'utilisateur peut redemander son lien.
	h.mailer.FailWith(nil)
	if err := h.service.ResendActivation(ctx, "file-pleine@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := h.service.Activate(ctx, h.lastToken()); err != nil {
		t.Fatalf("le lien renvoyé doit fonctionner : %v", err)
	}
}

// ---- Activation -----------------------------------------------------------------------------

func TestActivateOpensTheAccountOnlyOnce(t *testing.T) {
	h := newHarness(t, nil)
	u, _ := h.service.Register(ctx, registration("activation@example.org"))
	token := h.lastToken()

	if err := h.service.Activate(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err := h.service.Activate(ctx, token); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("second usage : %v, attendu ErrTokenInvalid", err)
	}

	got, _ := h.users.FindByID(ctx, u.ID)
	if got.Status != user.StatusActive || got.ActivatedAt == nil {
		t.Errorf("compte = %+v", got)
	}
}

func TestActivateRejectsBadTokens(t *testing.T) {
	h := newHarness(t, nil)
	h.service.Register(ctx, registration("jeton@example.org"))
	valid := h.lastToken()

	h.clock.Advance(25 * time.Hour) // au-delà des 24 h de validité
	tests := map[string]string{
		"expiré":      valid,
		"mal formé":   "pas-un-jeton",
		"vide":        "",
		"inconnu":     strings.Repeat("A", 43),
		"trop long":   strings.Repeat("A", 5000),
		"avec espace": valid + " ",
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if err := h.service.Activate(ctx, token); !errors.Is(err, auth.ErrTokenInvalid) {
				t.Fatalf("erreur = %v, attendu ErrTokenInvalid", err)
			}
		})
	}
}

func TestActivateIsAtomicWithTheTokenConsumption(t *testing.T) {
	h := newHarness(t, nil)
	u, _ := h.service.Register(ctx, registration("atomique@example.org"))
	token := h.lastToken()

	// Le compte est désactivé avant que l'utilisateur clique sur le lien.
	if _, err := h.users.Deactivate(ctx, u.ID, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := h.service.Activate(ctx, token); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("un compte désactivé ne s'active pas : %v", err)
	}
	if h.count("user_tokens", "used_at IS NOT NULL") != 0 {
		t.Fatal("l'activation a échoué : le jeton ne doit pas être consommé (transaction annulée)")
	}

	// Un modérateur rétablit le compte : le même lien fonctionne alors.
	if _, err := h.users.Reactivate(ctx, u.ID, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := h.service.Activate(ctx, token); err != nil {
		t.Fatalf("le lien non consommé doit encore fonctionner : %v", err)
	}
}

func TestConcurrentActivationsWithSameTokenSucceedOnce(t *testing.T) {
	h := newHarness(t, nil)
	h.service.Register(ctx, registration("double-clic@example.org"))
	token := h.lastToken()

	const clicks = 10
	var wg sync.WaitGroup
	var ok atomic.Int32
	for range clicks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := h.service.Activate(ctx, token); {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, auth.ErrTokenInvalid):
			default:
				t.Errorf("erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 1 {
		t.Fatalf("activations réussies = %d, attendu exactement 1", ok.Load())
	}
}

func TestResendActivation(t *testing.T) {
	h := newHarness(t, nil)
	h.service.Register(ctx, registration("renvoi@example.org"))
	first := h.lastToken()

	if err := h.service.ResendActivation(ctx, "RENVOI@example.org"); err != nil {
		t.Fatal(err)
	}
	second := h.lastToken()
	if first == second || len(h.mailer.Messages()) != 2 {
		t.Fatalf("un nouveau lien doit être envoyé (mails : %d)", len(h.mailer.Messages()))
	}
	if err := h.service.Activate(ctx, first); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Errorf("l'ancien lien doit être invalidé : %v", err)
	}
	if err := h.service.Activate(ctx, second); err != nil {
		t.Errorf("le nouveau lien doit fonctionner : %v", err)
	}

	before := len(h.mailer.Messages())
	for _, email := range []string{"renvoi@example.org", "inconnu@example.org"} {
		if err := h.service.ResendActivation(ctx, email); err != nil {
			t.Errorf("%s : %v", email, err)
		}
	}
	if len(h.mailer.Messages()) != before {
		t.Error("aucun mail pour un compte déjà actif ou inexistant")
	}
	if err := h.service.ResendActivation(ctx, "pas-un-email"); err == nil {
		t.Error("une adresse mal formée doit être signalée")
	}
}

// ---- Connexion ------------------------------------------------------------------------------

func TestLoginSuccessOpensSession(t *testing.T) {
	h := newHarness(t, nil)
	registered := h.registerAndActivate("steve@avengers.example")

	result, err := h.login("Steve@Avengers.example", goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != registered.ID || result.Session.Token == "" {
		t.Fatalf("résultat = %+v", result)
	}
	if want := h.clock.Now().Add(h.cfg.Session.Member.MaxLifetime); !result.Session.ExpiresAt.Equal(want) {
		t.Errorf("expiration = %v, attendu %v", result.Session.ExpiresAt, want)
	}

	principal, err := h.sessions.Authenticate(ctx, result.Session.Token)
	if err != nil || principal.UserID != registered.ID || principal.Role != user.RoleCitizen {
		t.Fatalf("Authenticate = (%+v, %v)", principal, err)
	}

	got, _ := h.users.FindByID(ctx, registered.ID)
	if got.LastLoginAt == nil || got.FailedLoginCount != 0 {
		t.Errorf("compte après connexion = %+v", got)
	}

	var stored struct{ DeviceLabel string }
	h.db.Raw(`SELECT device_label FROM sessions`).Scan(&stored)
	if stored.DeviceLabel != "Chrome · macOS" {
		t.Errorf("appareil = %q : le User-Agent brut ne doit pas être conservé", stored.DeviceLabel)
	}
}

func TestLoginGivesTheSameAnswerForUnknownAccountAndWrongPassword(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("existe@example.org")

	for name, attempt := range map[string]struct{ identifier, password string }{
		"mauvais mot de passe": {"existe@example.org", otherPassword},
		"compte inconnu":       {"inconnu@example.org", goodPassword},
		"pseudo inconnu":       {"Pseudo Inconnu", goodPassword},
		"identifiant invalide": {"a b@@c", goodPassword},
		"identifiant vide":     {"", goodPassword},
		"mot de passe vide":    {"existe@example.org", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := h.login(attempt.identifier, attempt.password); !errors.Is(err, auth.ErrInvalidCredentials) {
				t.Fatalf("erreur = %v, attendu ErrInvalidCredentials", err)
			}
		})
	}
}

func TestLoginRequiresActivationButOnlyRevealsItWithTheRightPassword(t *testing.T) {
	h := newHarness(t, nil)
	h.service.Register(ctx, registration("inactif@example.org"))

	if _, err := h.login("inactif@example.org", goodPassword); !errors.Is(err, auth.ErrAccountNotActivated) {
		t.Errorf("bon mot de passe : %v, attendu ErrAccountNotActivated", err)
	}
	if _, err := h.login("inactif@example.org", otherPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("mauvais mot de passe : %v, l'état du compte ne doit pas fuiter", err)
	}
}

func TestLoginRejectsDeactivatedAccountsOnlyWithTheRightPassword(t *testing.T) {
	h := newHarness(t, nil)
	u := h.registerAndActivate("banni@example.org")
	h.users.Deactivate(ctx, u.ID, h.clock.Now())

	if _, err := h.login("banni@example.org", goodPassword); !errors.Is(err, auth.ErrAccountDeactivated) {
		t.Errorf("bon mot de passe : %v", err)
	}
	if _, err := h.login("banni@example.org", otherPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("mauvais mot de passe : %v", err)
	}
}

func TestLoginLocksAccountAfterRepeatedFailures(t *testing.T) {
	h := newHarness(t, map[string]string{
		"HEROS_LOCKOUT_MAX_FAILURES": "3", "HEROS_LOCKOUT_BASE_DELAY": "1m", "HEROS_LOCKOUT_MAX_DELAY": "1h",
	})
	h.registerAndActivate("verrou@example.org")

	for range 3 {
		if _, err := h.login("verrou@example.org", otherPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("échec attendu : %v", err)
		}
	}

	// Même avec le bon mot de passe, le compte reste fermé pendant le verrouillage.
	_, err := h.login("verrou@example.org", goodPassword)
	var locked *auth.LockedError
	if !errors.Is(err, auth.ErrAccountLocked) || !errors.As(err, &locked) {
		t.Fatalf("erreur = %v, attendu un LockedError", err)
	}
	if want := h.clock.Now().Add(time.Minute); !locked.Until.Equal(want) {
		t.Errorf("verrouillé jusqu'à %v, attendu %v", locked.Until, want)
	}

	h.clock.Advance(61 * time.Second)
	if _, err := h.login("verrou@example.org", goodPassword); err != nil {
		t.Fatalf("après le délai, la connexion doit fonctionner : %v", err)
	}
	got, _ := h.users.FindByEmail(ctx, "verrou@example.org")
	if got.FailedLoginCount != 0 || got.LockedUntil != nil {
		t.Errorf("une connexion réussie doit lever le verrouillage : %+v", got)
	}
}

func TestLoginByAliasIsReservedToHeroes(t *testing.T) {
	h := newHarness(t, nil)
	hero := h.createUser("tony@stark.io", user.RoleHero, user.StatusActive)
	citizen := h.createUser("pepper@stark.io", user.RoleCitizen, user.StatusActive)
	for _, profile := range []struct {
		id    interface{}
		alias string
	}{{hero.ID, "Iron Man"}, {citizen.ID, "Faux Héros"}} {
		if err := h.db.Exec(`INSERT INTO hero_profiles (user_id, alias) VALUES (?, ?)`, profile.id, profile.alias).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := h.login("iron man", goodPassword)
	if err != nil || result.User.ID != hero.ID || result.User.DisplayName() != "Iron Man" {
		t.Fatalf("connexion par pseudo : (%+v, %v)", result, err)
	}
	if _, err := h.login("Faux Héros", goodPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("un citoyen ne se connecte pas par pseudo : %v", err)
	}
}

func TestLoginUpgradesWeakPasswordHashes(t *testing.T) {
	h := newHarness(t, nil)
	u := h.createUser("ancien@example.org", user.RoleCitizen, user.StatusActive)

	// L'empreinte stockée a été calculée avec des paramètres plus faibles que ceux d'aujourd'hui.
	weaker := h.cfg.Security.Argon2
	weaker.MemoryKiB = 32
	oldHash, _ := auth.NewPasswordHasher(weaker).Hash(ctx, goodPassword)
	if err := h.users.UpdatePasswordHash(ctx, u.ID, oldHash, h.clock.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := h.login("ancien@example.org", goodPassword); err != nil {
		t.Fatal(err)
	}
	got, _ := h.users.FindByID(ctx, u.ID)
	if got.PasswordHash == oldHash {
		t.Fatal("l'empreinte doit être recalculée avec les paramètres actuels")
	}
	if check, err := h.hasher.Verify(ctx, got.PasswordHash, goodPassword); err != nil || !check.Match || check.NeedsRehash {
		t.Fatalf("nouvelle empreinte = %+v, %v", check, err)
	}
}

// ---- Sessions -------------------------------------------------------------------------------

func TestSessionExpiresAfterInactivity(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("inactivite@example.org")
	result, _ := h.login("inactivite@example.org", goodPassword)

	h.clock.Advance(h.cfg.Session.Member.IdleTimeout - time.Minute)
	if _, err := h.sessions.Authenticate(ctx, result.Session.Token); err != nil {
		t.Fatalf("juste avant l'expiration : %v", err)
	}
	// L'activité précédente a repoussé l'échéance : la session vit encore presque aussi longtemps.
	h.clock.Advance(h.cfg.Session.Member.IdleTimeout - time.Minute)
	if _, err := h.sessions.Authenticate(ctx, result.Session.Token); err != nil {
		t.Fatalf("l'activité doit prolonger la session : %v", err)
	}

	h.clock.Advance(h.cfg.Session.Member.IdleTimeout + time.Minute)
	if _, err := h.sessions.Authenticate(ctx, result.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Fatalf("après l'inactivité : %v", err)
	}
}

func TestSessionExpiresAtMaxLifetimeEvenWhenActive(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("duree-max@example.org")
	result, _ := h.login("duree-max@example.org", goodPassword)

	for elapsed := time.Duration(0); elapsed < h.cfg.Session.Member.MaxLifetime-time.Hour; elapsed += time.Hour {
		h.clock.Advance(time.Hour)
		if _, err := h.sessions.Authenticate(ctx, result.Session.Token); err != nil {
			t.Fatalf("session active à %v : %v", elapsed, err)
		}
	}
	h.clock.Advance(2 * time.Hour)
	if _, err := h.sessions.Authenticate(ctx, result.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Fatalf("la durée maximale doit couper la session même active : %v", err)
	}
}

func TestRememberMeAndModeratorPolicies(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("souvenir@example.org")
	moderator := h.createUser("moderateur@example.org", user.RoleModerator, user.StatusActive)

	remembered, err := h.service.Login(ctx, auth.LoginInput{Identifier: "souvenir@example.org", Password: goodPassword, RememberMe: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := h.clock.Now().Add(h.cfg.Session.RememberMe.MaxLifetime); !remembered.Session.ExpiresAt.Equal(want) {
		t.Errorf("« se souvenir de moi » : expiration %v, attendu %v", remembered.Session.ExpiresAt, want)
	}

	// Un modérateur ne bénéficie jamais de « se souvenir de moi ».
	staff, err := h.service.Login(ctx, auth.LoginInput{Identifier: moderator.Email, Password: goodPassword, RememberMe: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := h.clock.Now().Add(h.cfg.Session.Moderator.MaxLifetime); !staff.Session.ExpiresAt.Equal(want) {
		t.Errorf("modérateur : expiration %v, attendu %v", staff.Session.ExpiresAt, want)
	}
	h.clock.Advance(h.cfg.Session.Moderator.IdleTimeout + time.Minute)
	if _, err := h.sessions.Authenticate(ctx, staff.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Errorf("la session d'un modérateur expire vite : %v", err)
	}
	if _, err := h.sessions.Authenticate(ctx, remembered.Session.Token); err != nil {
		t.Errorf("la session mémorisée reste valide : %v", err)
	}
}

func TestSessionStopsWorkingWhenAccountIsDeactivated(t *testing.T) {
	h := newHarness(t, nil)
	u := h.registerAndActivate("desactive@example.org")
	result, _ := h.login("desactive@example.org", goodPassword)

	if _, err := h.users.Deactivate(ctx, u.ID, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := h.sessions.Authenticate(ctx, result.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Fatalf("la désactivation doit couper les sessions immédiatement : %v", err)
	}
}

func TestSessionPicksUpRoleChangesImmediately(t *testing.T) {
	h := newHarness(t, nil)
	u := h.registerAndActivate("promu@example.org")
	result, _ := h.login("promu@example.org", goodPassword)

	if err := h.users.SetRole(ctx, u.ID, user.RoleHero, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	principal, err := h.sessions.Authenticate(ctx, result.Session.Token)
	if err != nil || principal.Role != user.RoleHero {
		t.Fatalf("rôle = %v (%v) : il doit être relu à chaque requête", principal.Role, err)
	}
}

func TestLogoutIsIdempotentAndLogoutAllClosesEverySession(t *testing.T) {
	h := newHarness(t, nil)
	u := h.registerAndActivate("deconnexion@example.org")
	first, _ := h.login("deconnexion@example.org", goodPassword)
	second, _ := h.login("deconnexion@example.org", goodPassword)
	third, _ := h.login("deconnexion@example.org", goodPassword)

	if err := h.service.Logout(ctx, first.Session.Token); err != nil {
		t.Fatal(err)
	}
	if err := h.service.Logout(ctx, first.Session.Token); err != nil {
		t.Fatalf("la déconnexion doit être idempotente : %v", err)
	}
	if err := h.service.Logout(ctx, "pas-un-jeton"); err != nil {
		t.Fatalf("un jeton absurde ne doit pas provoquer d'erreur : %v", err)
	}
	if _, err := h.sessions.Authenticate(ctx, first.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Error("la session déconnectée doit être invalide")
	}
	if _, err := h.sessions.Authenticate(ctx, second.Session.Token); err != nil {
		t.Error("les autres sessions ne doivent pas être affectées")
	}

	if err := h.service.LogoutAll(ctx, u.ID, testIP); err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"deuxième": second.Session.Token, "troisième": third.Session.Token} {
		if _, err := h.sessions.Authenticate(ctx, token); !errors.Is(err, auth.ErrSessionInvalid) {
			t.Errorf("la %s session doit être fermée", name)
		}
	}
	if h.count("audit_logs", "action = 'auth.logout_all'") != 1 {
		t.Error("« se déconnecter partout » doit être journalisé")
	}
}

func TestSessionTokensAreIndependentAndNeverStoredInClear(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("jetons@example.org")
	first, _ := h.login("jetons@example.org", goodPassword)
	second, _ := h.login("jetons@example.org", goodPassword)

	if first.Session.Token == second.Session.Token {
		t.Fatal("chaque connexion doit produire un nouveau jeton (anti-fixation)")
	}
	var rows []struct{ TokenHash []byte }
	h.db.Raw(`SELECT token_hash FROM sessions`).Scan(&rows)
	for _, row := range rows {
		if strings.Contains(string(row.TokenHash), first.Session.Token) || len(row.TokenHash) != 32 {
			t.Fatal("seule l'empreinte SHA-256 du jeton doit être stockée")
		}
	}
}

func TestSessionTouchIsThrottled(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("touch@example.org")
	result, _ := h.login("touch@example.org", goodPassword)

	lastSeen := func() time.Time {
		var row struct{ LastSeenAt time.Time }
		h.db.Raw(`SELECT last_seen_at FROM sessions`).Scan(&row)
		return row.LastSeenAt
	}

	created := lastSeen()
	h.clock.Advance(h.cfg.Session.TouchPeriod / 2)
	h.sessions.Authenticate(ctx, result.Session.Token)
	if !lastSeen().Equal(created) {
		t.Error("moins d'une période : aucune écriture attendue")
	}

	h.clock.Advance(h.cfg.Session.TouchPeriod)
	h.sessions.Authenticate(ctx, result.Session.Token)
	if !lastSeen().After(created) {
		t.Error("après une période : last_seen_at doit être mis à jour")
	}
}

func TestSessionRejectsMalformedAndUnknownTokens(t *testing.T) {
	h := newHarness(t, nil)
	for _, token := range []string{"", "x", strings.Repeat("A", 43), strings.Repeat("A", 4096), "../../etc/passwd"} {
		if _, err := h.sessions.Authenticate(ctx, token); !errors.Is(err, auth.ErrSessionInvalid) {
			t.Errorf("Authenticate(%.15q) = %v, attendu ErrSessionInvalid", token, err)
		}
	}
}

func TestCleanupRemovesOnlyStaleSessionsAndTokens(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("menage@example.org")
	old, _ := h.login("menage@example.org", goodPassword)
	h.service.Logout(ctx, old.Session.Token)

	h.clock.Advance(10 * 24 * time.Hour)
	fresh, _ := h.login("menage@example.org", goodPassword)

	sessions, tokens, err := h.service.Cleanup(ctx, 7*24*time.Hour, 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if sessions != 1 || tokens != 1 {
		t.Errorf("supprimés : %d sessions, %d jetons ; attendu 1 et 1", sessions, tokens)
	}
	if _, err := h.sessions.Authenticate(ctx, fresh.Session.Token); err != nil {
		t.Errorf("la session récente doit survivre au nettoyage : %v", err)
	}
}

// ---- Mot de passe oublié --------------------------------------------------------------------

func TestPasswordResetFullFlow(t *testing.T) {
	h := newHarness(t, nil)
	u := h.registerAndActivate("oubli@example.org")
	session, _ := h.login("oubli@example.org", goodPassword)
	h.mailer.FailWith(nil)

	if err := h.service.RequestPasswordReset(ctx, "Oubli@Example.org"); err != nil {
		t.Fatal(err)
	}
	message, _ := h.mailer.Last()
	if !strings.Contains(message.Subject, "Réinitialisation") {
		t.Fatalf("mail = %+v", message)
	}
	token := h.lastToken()

	if err := h.service.ConfirmPasswordReset(ctx, token, otherPassword, testIP); err != nil {
		t.Fatal(err)
	}

	if _, err := h.login("oubli@example.org", goodPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("l'ancien mot de passe ne doit plus fonctionner : %v", err)
	}
	if _, err := h.login("oubli@example.org", otherPassword); err != nil {
		t.Errorf("le nouveau mot de passe doit fonctionner : %v", err)
	}
	if _, err := h.sessions.Authenticate(ctx, session.Session.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Error("les sessions existantes doivent être fermées après un changement de mot de passe")
	}
	if err := h.service.ConfirmPasswordReset(ctx, token, goodPassword, testIP); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Errorf("le lien ne sert qu'une fois : %v", err)
	}
	if h.count("audit_logs", "action = 'auth.password_reset' AND actor_id = ?", u.ID) != 1 {
		t.Error("le changement de mot de passe doit être journalisé")
	}
}

func TestPasswordResetRequestNeverRevealsAccounts(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("actif@example.org")
	h.service.Register(ctx, registration("en-attente@example.org"))
	deactivated := h.registerAndActivate("banni@example.org")
	h.users.Deactivate(ctx, deactivated.ID, h.clock.Now())

	before := len(h.mailer.Messages())
	for _, email := range []string{"inconnu@example.org", "en-attente@example.org", "banni@example.org"} {
		if err := h.service.RequestPasswordReset(ctx, email); err != nil {
			t.Errorf("%s : %v : la réponse doit être identique pour tous les comptes", email, err)
		}
	}
	if len(h.mailer.Messages()) != before {
		t.Error("aucun mail de réinitialisation pour un compte inconnu, inactif ou désactivé")
	}

	var validationErr *validation.Error
	if err := h.service.RequestPasswordReset(ctx, "pas-un-email"); !errors.As(err, &validationErr) {
		t.Errorf("une adresse mal formée est signalée : %v", err)
	}
}

func TestNewPasswordResetRequestInvalidatesThePreviousLink(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("deux-demandes@example.org")

	h.service.RequestPasswordReset(ctx, "deux-demandes@example.org")
	first := h.lastToken()
	h.service.RequestPasswordReset(ctx, "deux-demandes@example.org")
	second := h.lastToken()

	if err := h.service.ConfirmPasswordReset(ctx, first, otherPassword, testIP); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Errorf("l'ancien lien doit être invalidé : %v", err)
	}
	if err := h.service.ConfirmPasswordReset(ctx, second, otherPassword, testIP); err != nil {
		t.Errorf("le dernier lien doit fonctionner : %v", err)
	}
}

func TestPasswordResetWithWeakPasswordKeepsTheLinkUsable(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("faible@example.org")
	h.service.RequestPasswordReset(ctx, "faible@example.org")
	token := h.lastToken()

	var validationErr *validation.Error
	if err := h.service.ConfirmPasswordReset(ctx, token, "faible", testIP); !errors.As(err, &validationErr) || validationErr.Fields["newPassword"] == "" {
		t.Fatalf("erreur = %v, attendu une erreur de validation sur newPassword", err)
	}
	if err := h.service.ConfirmPasswordReset(ctx, token, otherPassword, testIP); err != nil {
		t.Fatalf("une saisie faible ne doit pas brûler le lien : %v", err)
	}
}

func TestPasswordResetRejectsExpiredAndForgedTokens(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("expire@example.org")
	h.service.RequestPasswordReset(ctx, "expire@example.org")
	token := h.lastToken()
	activationToken := strings.Repeat("B", 43)

	h.clock.Advance(h.cfg.Security.Tokens.PasswordResetTTL + time.Minute)
	for name, candidate := range map[string]string{"expiré": token, "forgé": activationToken, "malformé": "x"} {
		if err := h.service.ConfirmPasswordReset(ctx, candidate, otherPassword, testIP); !errors.Is(err, auth.ErrTokenInvalid) {
			t.Errorf("%s : %v", name, err)
		}
	}
}

func TestPasswordResetCannotBeDoneWithAnActivationToken(t *testing.T) {
	h := newHarness(t, nil)
	h.service.Register(ctx, registration("croise@example.org"))
	activationToken := h.lastToken()

	if err := h.service.ConfirmPasswordReset(ctx, activationToken, otherPassword, testIP); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("un jeton d'activation ne doit pas servir à changer un mot de passe : %v", err)
	}
	if err := h.service.Activate(ctx, activationToken); err != nil {
		t.Fatalf("et il doit rester utilisable pour l'activation : %v", err)
	}
}

func TestPasswordResetLiftsLockout(t *testing.T) {
	h := newHarness(t, map[string]string{"HEROS_LOCKOUT_MAX_FAILURES": "2"})
	h.registerAndActivate("verrouille@example.org")
	for range 2 {
		h.login("verrouille@example.org", otherPassword)
	}
	if _, err := h.login("verrouille@example.org", goodPassword); !errors.Is(err, auth.ErrAccountLocked) {
		t.Fatalf("le compte devrait être verrouillé : %v", err)
	}

	h.service.RequestPasswordReset(ctx, "verrouille@example.org")
	if err := h.service.ConfirmPasswordReset(ctx, h.lastToken(), otherPassword, testIP); err != nil {
		t.Fatal(err)
	}
	if _, err := h.login("verrouille@example.org", otherPassword); err != nil {
		t.Fatalf("la récupération du compte doit lever le verrouillage : %v", err)
	}
}

func TestConcurrentPasswordResetsWithSameTokenSucceedOnce(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("course-reset@example.org")
	h.service.RequestPasswordReset(ctx, "course-reset@example.org")
	token := h.lastToken()

	var wg sync.WaitGroup
	var ok atomic.Int32
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.service.ConfirmPasswordReset(ctx, token, otherPassword, testIP); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("changements réussis = %d, attendu exactement 1", ok.Load())
	}
}

func TestConcurrentLoginsOfSameUserGetIndependentSessions(t *testing.T) {
	h := newHarness(t, nil)
	h.registerAndActivate("multi@example.org")

	const logins = 8
	tokens := make(chan string, logins)
	var wg sync.WaitGroup
	for range logins {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := h.login("multi@example.org", goodPassword)
			if err != nil {
				t.Error(err)
				return
			}
			tokens <- result.Session.Token
		}()
	}
	wg.Wait()
	close(tokens)

	seen := make(map[string]bool)
	for token := range tokens {
		if seen[token] {
			t.Fatal("deux connexions ont reçu le même jeton")
		}
		seen[token] = true
		if _, err := h.sessions.Authenticate(ctx, token); err != nil {
			t.Errorf("session invalide : %v", err)
		}
	}
	if len(seen) != logins {
		t.Fatalf("sessions = %d, attendu %d", len(seen), logins)
	}
}
