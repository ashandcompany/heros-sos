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
