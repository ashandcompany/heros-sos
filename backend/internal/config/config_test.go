package config

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func fakeEnv(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// validEnv retourne le minimum requis pour obtenir une configuration de développement valide.
func validEnv() map[string]string {
	return map[string]string{
		envDatabaseURL:   "postgres://heros:heros@localhost:5432/herossos?sslmode=disable",
		envCryptoKeys:    "1:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, MasterKeyLength)),
		envCryptoActiveK: "1",
		envCSRFSecret:    base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, minCSRFSecret)),
	}
}

func productionEnv() map[string]string {
	env := validEnv()
	env[envEnvironment] = string(EnvironmentProduction)
	env[envDatabaseURL] = "postgres://heros:secret@db:5432/herossos?sslmode=require"
	env[envAllowedOrigins] = "https://heros-sos.example"
	env[envPublicBaseURL] = "https://heros-sos.example"
	env[envSMTPHost] = "smtp.example"
	env[envOMDbAPIKey] = "test-key"
	return env
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil {
		t.Fatalf("Load() a retourné une erreur : %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"environnement", cfg.Environment, EnvironmentDevelopment},
		{"niveau de log", cfg.LogLevel, slog.LevelInfo},
		{"adresse HTTP", cfg.HTTP.Addr, defaultHTTPAddr},
		{"taille max du corps", cfg.HTTP.MaxBodyBytes, int64(defaultMaxBodyBytes)},
		{"délai du healthcheck", cfg.HTTP.HealthcheckTimeout, defaultHealthcheckTimeout},
		{"session membre : inactivité", cfg.Session.Member.IdleTimeout, defaultMemberSessionIdle},
		{"session modérateur : durée max", cfg.Session.Moderator.MaxLifetime, defaultModeratorMax},
		{"cookie sécurisé", cfg.Session.CookieSecure, false},
		{"nom du cookie", cfg.Session.CookieName(), plainCookieName},
		{"argon2 mémoire", cfg.Security.Argon2.MemoryKiB, uint32(defaultArgon2MemoryKiB)},
		{"mot de passe min", cfg.Security.Password.MinLength, defaultPasswordMinLength},
		{"limite connexion par IP", cfg.Security.RateLimits.LoginPerIP, defaultRateLoginPerIP},
		{"driver mail", cfg.Mail.Driver, MailDriverLog},
		{"TLS SMTP", cfg.Mail.SMTP.TLSMode, TLSModeNone},
		{"migration auto", cfg.AutoMigrate, false},
		{"limite contact par IP", cfg.Security.RateLimits.ContactPerIP, defaultRateContactPerIP},
		{"limite newsletter par IP", cfg.Security.RateLimits.NewsletterPerIP, defaultRateNewsletterPerIP},
		{"limite newsletter par adresse", cfg.Security.RateLimits.NewsletterPerEmail, defaultRateNewsletterPerEmail},
		{"limite changement d'adresse", cfg.Security.RateLimits.EmailChangePerUser, defaultRateEmailChangePerUser},
		{"limite photo de profil", cfg.Security.RateLimits.AvatarPerUser, defaultRateAvatarPerUser},
		{"validité du lien newsletter", cfg.Security.Tokens.NewsletterConfirmTTL, defaultNewsletterConfirmTTL},
		{"destinataire du contact", cfg.Mail.ContactRecipient, defaultContactRecipient},
		{"dossier des photos", cfg.Avatar.Dir, defaultAvatarDir},
		{"taille max d'une photo", cfg.Avatar.MaxBytes, int64(defaultAvatarMaxBytes)},
		{"dimension max d'une photo", cfg.Avatar.MaxDimension, defaultAvatarMaxDimension},
		{"page de confirmation newsletter", cfg.App.NewsletterConfirmPath, defaultNewsletterConfirmPath},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s : obtenu %v, attendu %v", check.name, check.got, check.want)
		}
	}
}

func TestLoadOverridesFromEnvironment(t *testing.T) {
	env := validEnv()
	env[envHTTPAddr] = ":9090"
	env[envSessionMemberIdle] = "30m"
	env[envRateLoginPerIP] = "7/2m"
	env[envLogLevel] = "debug"
	env[envTrustedProxies] = "10.0.0.0/8, 192.168.1.5"
	env[envAllowedOrigins] = "https://A.example/, http://localhost:3000"
	env[envAutoMigrate] = "true"
	env[envContactRecipient] = "bureau@heros-sos.example"
	env[envAvatarDir] = "/var/lib/photos"
	env[envAvatarMaxBytes] = "1048576"
	env[envAvatarMaxDimension] = "512"
	env[envRateContactPerIP] = "2/30m"
	env[envNewsletterConfirmTTL] = "72h"
	env[envNewsletterConfirmPath] = "/newsletter/ok"

	cfg, err := Load(fakeEnv(env))
	if err != nil {
		t.Fatalf("Load() a retourné une erreur : %v", err)
	}

	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("adresse = %q", cfg.HTTP.Addr)
	}
	if cfg.Session.Member.IdleTimeout != 30*time.Minute {
		t.Errorf("inactivité = %v", cfg.Session.Member.IdleTimeout)
	}
	if got := cfg.Security.RateLimits.LoginPerIP; got.Max != 7 || got.Window != 2*time.Minute {
		t.Errorf("limite = %+v", got)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("niveau de log = %v", cfg.LogLevel)
	}
	if len(cfg.HTTP.TrustedProxies) != 2 || cfg.HTTP.TrustedProxies[1].Bits() != 32 {
		t.Errorf("proxys de confiance = %v", cfg.HTTP.TrustedProxies)
	}
	if !cfg.AutoMigrate {
		t.Error("la migration automatique devrait être active")
	}
	if cfg.Mail.ContactRecipient != "bureau@heros-sos.example" {
		t.Errorf("destinataire du contact = %q", cfg.Mail.ContactRecipient)
	}
	if got := cfg.Avatar; got.Dir != "/var/lib/photos" || got.MaxBytes != 1<<20 || got.MaxDimension != 512 {
		t.Errorf("photos = %+v", got)
	}
	if got := cfg.Security.RateLimits.ContactPerIP; got.Max != 2 || got.Window != 30*time.Minute {
		t.Errorf("limite du contact = %+v", got)
	}
	if cfg.Security.Tokens.NewsletterConfirmTTL != 72*time.Hour || cfg.App.NewsletterConfirmPath != "/newsletter/ok" {
		t.Errorf("confirmation newsletter = %v, %q", cfg.Security.Tokens.NewsletterConfirmTTL, cfg.App.NewsletterConfirmPath)
	}
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	env := map[string]string{
		envEnvironment:      "staging",
		envHTTPReadTimeout:  "vite",
		envRateLoginPerIP:   "beaucoup",
		envArgon2Time:       "-1",
		envCookieSecure:     "peut-être",
		envCryptoKeys:       "abc",
		envTrustedProxies:   "pas-une-ip",
		envPublicBaseURL:    "/relatif",
		envMailDriver:       "pigeon",
		envCSRFSecret:       "%%%",
		envSessionMemberMax: "1h",
	}
	_, err := Load(fakeEnv(env))
	if err == nil {
		t.Fatal("Load() devrait échouer")
	}

	for _, name := range []string{
		envEnvironment, envHTTPReadTimeout, envRateLoginPerIP, envArgon2Time, envCookieSecure,
		envCryptoKeys, envTrustedProxies, envPublicBaseURL, envMailDriver, envCSRFSecret,
		envDatabaseURL,
	} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("l'erreur ne mentionne pas %s :\n%v", name, err)
		}
	}
}

func TestLoadRejectsInconsistentValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(env map[string]string)
		wantErr string
	}{
		{"clé trop courte", func(env map[string]string) {
			env[envCryptoKeys] = "1:" + base64.StdEncoding.EncodeToString([]byte("trop-courte"))
		}, "octets"},
		{"clé active inconnue", func(env map[string]string) { env[envCryptoActiveK] = "2" }, "n'existe pas"},
		{"secret CSRF trop court", func(env map[string]string) {
			env[envCSRFSecret] = base64.StdEncoding.EncodeToString([]byte("court"))
		}, envCSRFSecret},
		{"inactivité supérieure à la durée max", func(env map[string]string) {
			env[envSessionMemberIdle] = "10h"
			env[envSessionMemberMax] = "1h"
		}, "inactivité maximale"},
		{"mot de passe min supérieur au max", func(env map[string]string) {
			env[envPasswordMinLength] = "50"
			env[envPasswordMaxLength] = "20"
		}, envPasswordMinLength},
		{"origine invalide", func(env map[string]string) { env[envAllowedOrigins] = "https://a.example/chemin" }, "origine invalide"},
		{"URL de base de données", func(env map[string]string) { env[envDatabaseURL] = "mysql://x" }, "postgres://"},
		{"destinataire du contact invalide", func(env map[string]string) { env[envContactRecipient] = "pas-une-adresse" }, envContactRecipient},
		{"destinataire du contact avec un nom", func(env map[string]string) { env[envContactRecipient] = "Bureau <b@heros-sos.example>" }, envContactRecipient},
		{"taille de photo nulle", func(env map[string]string) { env[envAvatarMaxBytes] = "0" }, envAvatarMaxBytes},
		{"dimension de photo invalide", func(env map[string]string) { env[envAvatarMaxDimension] = "grande" }, envAvatarMaxDimension},
		{"limite du contact invalide", func(env map[string]string) { env[envRateContactPerIP] = "beaucoup" }, envRateContactPerIP},
		{"validité du lien newsletter invalide", func(env map[string]string) { env[envNewsletterConfirmTTL] = "-1h" }, envNewsletterConfirmTTL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := validEnv()
			test.mutate(env)
			_, err := Load(fakeEnv(env))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("erreur = %v, attendu un message contenant %q", err, test.wantErr)
			}
		})
	}
}

func TestProductionIsStricter(t *testing.T) {
	if _, err := Load(fakeEnv(productionEnv())); err != nil {
		t.Fatalf("une configuration de production correcte doit passer : %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(env map[string]string)
		wantErr string
	}{
		{"cookie non sécurisé", func(env map[string]string) { env[envCookieSecure] = "false" }, envCookieSecure},
		{"base sans TLS", func(env map[string]string) {
			env[envDatabaseURL] = "postgres://u:p@db/herossos?sslmode=disable"
		}, "sslmode=disable"},
		{"driver mail log", func(env map[string]string) { env[envMailDriver] = "log" }, envMailDriver},
		{"SMTP sans TLS", func(env map[string]string) { env[envSMTPTLSMode] = "none" }, envSMTPTLSMode},
		{"origines absentes", func(env map[string]string) { delete(env, envAllowedOrigins) }, envAllowedOrigins},
		{"origine en http", func(env map[string]string) { env[envAllowedOrigins] = "http://heros-sos.example" }, "https"},
		{"URL publique en http", func(env map[string]string) { env[envPublicBaseURL] = "http://heros-sos.example" }, envPublicBaseURL},
		{"argon2 trop faible", func(env map[string]string) { env[envArgon2MemoryKiB] = "1024" }, "argon2id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := productionEnv()
			test.mutate(env)
			_, err := Load(fakeEnv(env))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("erreur = %v, attendu un message contenant %q", err, test.wantErr)
			}
		})
	}
}

func TestProductionDefaults(t *testing.T) {
	cfg, err := Load(fakeEnv(productionEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Session.CookieSecure || cfg.Session.CookieName() != secureCookieName {
		t.Errorf("le cookie doit être sécurisé et préfixé __Host- en production : %q", cfg.Session.CookieName())
	}
	if cfg.Mail.Driver != MailDriverSMTP || cfg.Mail.SMTP.TLSMode != TLSModeStartTLS {
		t.Errorf("mail = %+v", cfg.Mail)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"https://Heros-SOS.example", "https://heros-sos.example", true},
		{"https://heros-sos.example/", "https://heros-sos.example", true},
		{"http://localhost:3000", "http://localhost:3000", true},
		{"https://heros-sos.example/chemin", "", false},
		{"https://heros-sos.example?x=1", "", false},
		{"ftp://heros-sos.example", "", false},
		{"heros-sos.example", "", false},
		{"", "", false},
	}
	for _, test := range tests {
		got, ok := NormalizeOrigin(test.input)
		if got != test.want || ok != test.ok {
			t.Errorf("NormalizeOrigin(%q) = (%q, %v), attendu (%q, %v)", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestAppLinks(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.App.ActivationLink("a b/c"), "http://localhost:3000/activation?token=a+b%2Fc"; got != want {
		t.Errorf("ActivationLink = %q, attendu %q", got, want)
	}
	if got, want := cfg.App.PasswordResetLink("xyz"), "http://localhost:3000/reinitialisation-mot-de-passe?token=xyz"; got != want {
		t.Errorf("PasswordResetLink = %q, attendu %q", got, want)
	}
	if got, want := cfg.App.NewsletterConfirmLink("a b/c"), "http://localhost:3000/newsletter/confirmation?token=a+b%2Fc"; got != want {
		t.Errorf("NewsletterConfirmLink = %q, attendu %q", got, want)
	}
	if got, want := cfg.App.HeroReviewLink("intervention id", "hero/id"), "http://localhost:3000/profil/interventions?heroId=hero%2Fid&interventionId=intervention+id"; got != want {
		t.Errorf("HeroReviewLink = %q, attendu %q", got, want)
	}
}

// TestEnvExampleStaysInSync garantit que .env.example documente exactement les variables lues
// par le code : une variable ajoutée sans documentation, ou documentée mais supprimée, échoue.
func TestEnvExampleStaysInSync(t *testing.T) {
	content, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatalf("lecture de .env.example : %v", err)
	}

	documented := make(map[string]bool)
	for _, match := range regexp.MustCompile(`(?m)^#?\s*(HEROS_[A-Z0-9_]+)=`).FindAllStringSubmatch(string(content), -1) {
		// Les variantes NOM_FILE sont décrites une fois pour toutes, pas variable par variable.
		if !strings.HasSuffix(match[1], fileSuffix) {
			documented[match[1]] = true
		}
	}

	known := make(map[string]bool)
	for _, name := range EnvVarNames() {
		known[name] = true
		if !documented[name] {
			t.Errorf("%s est lue par le code mais absente de .env.example", name)
		}
	}
	for name := range documented {
		if !known[name] {
			t.Errorf("%s est dans .env.example mais n'est lue par aucun code", name)
		}
	}
}

// ---- Secrets fournis par des fichiers ---------------------------------------------------------

func writeSecret(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSecretsCanComeFromFiles(t *testing.T) {
	direct := validEnv()
	env := map[string]string{
		// Un saut de ligne final, comme en laisse un `echo` ou un éditeur, est ignoré.
		envDatabaseURL + fileSuffix: writeSecret(t, direct[envDatabaseURL]+"\n"),
		envCryptoKeys + fileSuffix:  writeSecret(t, direct[envCryptoKeys]+"\r\n"),
		envCSRFSecret + fileSuffix:  writeSecret(t, direct[envCSRFSecret]+"\n"),
		envCryptoActiveK:            direct[envCryptoActiveK],
	}

	cfg, err := Load(fakeEnv(env))
	if err != nil {
		t.Fatalf("Load() : %v", err)
	}
	if cfg.Database.URL != direct[envDatabaseURL] {
		t.Errorf("URL de base lue depuis le fichier = %q", cfg.Database.URL)
	}
	if len(cfg.Crypto.Keys[1]) != MasterKeyLength || len(cfg.Security.CSRFSecret) != minCSRFSecret {
		t.Errorf("clés lues depuis les fichiers : %d et %d octets", len(cfg.Crypto.Keys[1]), len(cfg.Security.CSRFSecret))
	}
}

func TestSecretFileProblemsAreReportedWithoutLeakingContent(t *testing.T) {
	const secretContent = "contenu-ultra-secret"

	tests := []struct {
		name    string
		mutate  func(env map[string]string)
		wantErr string
	}{
		{"valeur directe et fichier", func(env map[string]string) {
			env[envDatabaseURL+fileSuffix] = writeSecret(t, secretContent)
		}, "tous deux définis"},
		{"fichier absent", func(env map[string]string) {
			delete(env, envDatabaseURL)
			env[envDatabaseURL+fileSuffix] = filepath.Join(t.TempDir(), "absent")
		}, "lecture impossible"},
		{"fichier vide", func(env map[string]string) {
			delete(env, envDatabaseURL)
			env[envDatabaseURL+fileSuffix] = writeSecret(t, "  \n")
		}, envDatabaseURL + " est obligatoire"},
		{"fichier de clés invalide", func(env map[string]string) {
			delete(env, envCryptoKeys)
			env[envCryptoKeys+fileSuffix] = writeSecret(t, secretContent)
		}, envCryptoKeys},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := validEnv()
			test.mutate(env)
			_, err := Load(fakeEnv(env))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("erreur = %v, attendu un message contenant %q", err, test.wantErr)
			}
			if strings.Contains(err.Error(), secretContent) {
				t.Fatalf("l'erreur ne doit jamais citer le contenu d'un secret : %v", err)
			}
		})
	}
}

func TestProductionRulesStillApplyToSecretsFromFiles(t *testing.T) {
	env := productionEnv()
	delete(env, envDatabaseURL)
	env[envDatabaseURL+fileSuffix] = writeSecret(t, "postgres://u:p@db/herossos?sslmode=disable")

	if _, err := Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), "sslmode=disable") {
		t.Fatalf("la validation de production doit s'appliquer à une valeur lue depuis un fichier : %v", err)
	}
}

// La 2FA des modérateurs est obligatoire avant toute mise en ligne : en production elle est exigée par défaut
// et ne peut pas être désactivée ; en développement elle est facultative, pour une démonstration.
func TestModeratorTwoFactorIsRequiredInProductionAndCannotBeDisabled(t *testing.T) {
	development, err := Load(fakeEnv(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if development.Security.MFA.ModeratorRequired || development.Security.MFA.Issuer != defaultMFAIssuer ||
		development.Security.MFA.RecoveryCodes != defaultMFARecoveryCodes {
		t.Errorf("développement : %+v", development.Security.MFA)
	}
	optional := validEnv()
	optional[envMFAModeratorRequired] = "true"
	if cfg, err := Load(fakeEnv(optional)); err != nil || !cfg.Security.MFA.ModeratorRequired {
		t.Errorf("exigée en développement : %+v %v", cfg.Security.MFA, err)
	}

	production, err := Load(fakeEnv(productionEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if !production.Security.MFA.ModeratorRequired {
		t.Error("la 2FA des modérateurs doit être exigée par défaut en production")
	}

	env := productionEnv()
	env[envMFAModeratorRequired] = "false"
	_, err = Load(fakeEnv(env))
	if err == nil || !strings.Contains(err.Error(), envMFAModeratorRequired) {
		t.Fatalf("désactiver la 2FA en production doit être refusé : %v", err)
	}

	env[envMFAModeratorRequired] = "peut-être"
	if _, err = Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), "booléen attendu") {
		t.Errorf("valeur invalide : %v", err)
	}
}

func TestLinkPathsAndTTLsOfTheAccountFlows(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Tokens.EmailChangeConfirmTTL != defaultEmailChangeConfirmTTL {
		t.Errorf("durée du lien de confirmation d'adresse = %v", cfg.Security.Tokens.EmailChangeConfirmTTL)
	}
	if got, want := cfg.App.EmailChangeConfirmLink("a b"), "http://localhost:3000/changement-email/confirmation?token=a+b"; got != want {
		t.Errorf("EmailChangeConfirmLink = %q, attendu %q", got, want)
	}
	if got, want := cfg.App.NewsletterUnsubscribeLink("x/y"), "http://localhost:3000/newsletter/desinscription?token=x%2Fy"; got != want {
		t.Errorf("NewsletterUnsubscribeLink = %q, attendu %q", got, want)
	}

	env := validEnv()
	env[envEmailChangeConfirmPath] = "/confirmer"
	env[envNewsletterUnsubscribePath] = "/stop"
	env[envEmailChangeConfirmTTL] = "1h"
	cfg, err = Load(fakeEnv(env))
	if err != nil || cfg.App.EmailChangeConfirmPath != "/confirmer" || cfg.App.NewsletterUnsubscribePath != "/stop" ||
		cfg.Security.Tokens.EmailChangeConfirmTTL != time.Hour {
		t.Errorf("surcharges : %+v %v", cfg.App, err)
	}
}

// Les paramètres d'argon2 sont convertis vers des types plus étroits (uint32, uint8) : une valeur hors limites
// doit être refusée, jamais tronquée en silence (300 threads deviendraient 44).
func TestArgon2SettingsAreBoundedToTheirType(t *testing.T) {
	tests := map[string]struct {
		name, value string
	}{
		"threads au-delà d'un octet":    {envArgon2Threads, "300"},
		"mémoire au-delà de 32 bits":    {envArgon2MemoryKiB, "4294967296"},
		"itérations au-delà de 32 bits": {envArgon2Time, "5000000000"},
		"threads nuls":                  {envArgon2Threads, "0"},
	}
	for label, tc := range tests {
		t.Run(label, func(t *testing.T) {
			env := validEnv()
			env[tc.name] = tc.value
			_, err := Load(fakeEnv(env))
			if err == nil || !strings.Contains(err.Error(), tc.name) {
				t.Errorf("%s=%s devrait être refusé, reçu %v", tc.name, tc.value, err)
			}
		})
	}

	env := validEnv()
	env[envArgon2Threads] = "255"
	cfg, err := Load(fakeEnv(env))
	if err != nil || cfg.Security.Argon2.Threads != 255 {
		t.Errorf("255 threads (la borne) doivent passer : %d, %v", cfg.Security.Argon2.Threads, err)
	}
}

func TestCleanupTimeoutHasADefaultAndMustBePositive(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil || cfg.Jobs.CleanupTimeout != defaultCleanupTimeout {
		t.Errorf("durée par défaut = %v, %v", cfg.Jobs.CleanupTimeout, err)
	}
	env := validEnv()
	env[envCleanupTimeout] = "0s"
	if _, err := Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), envCleanupTimeout) {
		t.Errorf("un délai nul doit être refusé : %v", err)
	}
}

func TestLogFormatDefaultsToJSONInProductionAndTextElsewhere(t *testing.T) {
	development, err := Load(fakeEnv(validEnv()))
	if err != nil || development.LogFormat != LogFormatText {
		t.Errorf("développement : %q, %v, attendu text", development.LogFormat, err)
	}
	production, err := Load(fakeEnv(productionEnv()))
	if err != nil || production.LogFormat != LogFormatJSON {
		t.Errorf("production : %q, %v, attendu json", production.LogFormat, err)
	}

	env := validEnv()
	env[envLogFormat] = "json"
	if cfg, err := Load(fakeEnv(env)); err != nil || cfg.LogFormat != LogFormatJSON {
		t.Errorf("json demandé en développement : %q, %v", cfg.LogFormat, err)
	}
	env[envLogFormat] = "xml"
	if _, err := Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), envLogFormat) {
		t.Errorf("un format inconnu doit être refusé : %v", err)
	}
}

func TestDashboardStatsTTLHasADefaultAndMayBeZero(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil || cfg.Dashboard.StatsTTL != defaultDashboardStatsTTL {
		t.Fatalf("défaut : %v, %v", cfg.Dashboard.StatsTTL, err)
	}
	for value, want := range map[string]time.Duration{"0": 0, "5m": 5 * time.Minute} {
		env := validEnv()
		env[envDashboardStatsTTL] = value
		cfg, err := Load(fakeEnv(env))
		if err != nil || cfg.Dashboard.StatsTTL != want {
			t.Errorf("%s = %q : %v, %v", envDashboardStatsTTL, value, cfg.Dashboard.StatsTTL, err)
		}
	}
	for _, value := range []string{"-1s", "bientôt"} {
		env := validEnv()
		env[envDashboardStatsTTL] = value
		if _, err := Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), envDashboardStatsTTL) {
			t.Errorf("%s = %q doit être refusé : %v", envDashboardStatsTTL, value, err)
		}
	}
}

func TestEventSettingsHaveDefaultsAndMustBePositive(t *testing.T) {
	cfg, err := Load(fakeEnv(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := Events{
		PollInterval: defaultEventsPollInterval, BatchSize: defaultEventsBatchSize, Heartbeat: defaultEventsHeartbeat,
		MaxStreamsPerUser: defaultEventsMaxStreamsPerUser, StreamMaxAge: defaultEventsStreamMaxAge, StreamBuffer: defaultEventsStreamBuffer,
	}
	if cfg.Events != want {
		t.Errorf("réglages = %+v, attendu %+v", cfg.Events, want)
	}
	if cfg.Jobs.CommandRetention != defaultCommandRetention {
		t.Errorf("conservation des commandes = %v", cfg.Jobs.CommandRetention)
	}
	for _, name := range []string{envEventsPollInterval, envEventsBatchSize, envEventsHeartbeat, envEventsMaxStreamsPerUser, envEventsStreamMaxAge, envEventsStreamBuffer, envCommandRetention} {
		env := validEnv()
		env[name] = "0"
		if _, err := Load(fakeEnv(env)); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s = 0 doit être refusé : %v", name, err)
		}
	}
}
