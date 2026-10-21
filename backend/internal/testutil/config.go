package testutil

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// Valeurs de test : volontairement faibles et publiques, elles ne protègent aucune donnée.
const (
	testDatabaseURL = "postgres://heros:heros@localhost:5432/herossos_test?sslmode=disable"
	testKeyID       = "1"
	testKeyFill     = 0x42
	testCSRFFill    = 0x24
	testCSRFSize    = 32
)

// Config retourne une configuration de test valide : hachage argon2id très léger pour
// accélérer les tests, secrets fixes, mails journalisés. Les variables fournies s'ajoutent
// à celles de base ou les remplacent.
func Config(t testing.TB, overrides map[string]string) config.Config {
	t.Helper()

	env := map[string]string{
		"HEROS_ENV":                  string(config.EnvironmentTest),
		"HEROS_DATABASE_URL":         testDatabaseURL,
		"HEROS_CRYPTO_KEYS":          testKeyID + ":" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{testKeyFill}, config.MasterKeyLength)),
		"HEROS_CRYPTO_ACTIVE_KEY_ID": testKeyID,
		"HEROS_CSRF_SECRET":          base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{testCSRFFill}, testCSRFSize)),
		"HEROS_ARGON2_TIME":          "1",
		"HEROS_ARGON2_MEMORY_KIB":    "64",
		"HEROS_ARGON2_THREADS":       "1",
		"HEROS_ALLOWED_ORIGINS":      "http://localhost:3000",
	}
	for name, value := range overrides {
		env[name] = value
	}

	cfg, err := config.Load(func(name string) string { return env[name] })
	if err != nil {
		t.Fatalf("configuration de test invalide : %v", err)
	}
	return cfg
}
