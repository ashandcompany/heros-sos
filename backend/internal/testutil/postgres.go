// Package testutil regroupe les aides partagées par les tests : base PostgreSQL isolée,
// configuration de test. Il n'est importé que par des fichiers *_test.go.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	// EnvTestDatabaseURL désigne la base PostgreSQL utilisée par les tests d'intégration,
	// par exemple postgres://heros@127.0.0.1:5432/herossos_test?sslmode=disable.
	EnvTestDatabaseURL = "HEROS_TEST_DATABASE_URL"

	extensionLockKey = 7_000_001
	schemaPrefix     = "t_"
	schemaRandomSize = 8
	poolSize         = 4
	pingTimeout      = 5 * time.Second
	slowQuery        = 5 * time.Second

	deadlockCode      = "40P01"
	dropSchemaRetries = 5
	dropSchemaBackoff = 200 * time.Millisecond
)

// NewDB retourne une base PostgreSQL vierge, migrée, et isolée dans son propre schéma : les
// tests peuvent tourner en parallèle sans se perturber. Le schéma est supprimé à la fin.
// Sans HEROS_TEST_DATABASE_URL, le test est ignoré (échec en intégration continue, où
// ignorer silencieusement ces tests masquerait des régressions).
func NewDB(t testing.TB) *gorm.DB {
	t.Helper()

	baseURL := os.Getenv(EnvTestDatabaseURL)
	if baseURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s doit être défini en intégration continue", EnvTestDatabaseURL)
		}
		t.Skipf("%s non défini : test d'intégration ignoré", EnvTestDatabaseURL)
	}

	admin := open(t, baseURL)
	ensureExtension(t, admin)

	schema := schemaPrefix + randomSuffix(t)
	if err := admin.Exec(`CREATE SCHEMA ` + schema).Error; err != nil {
		t.Fatalf("création du schéma de test : %v", err)
	}
	t.Cleanup(func() {
		if err := dropSchema(admin, schema); err != nil {
			t.Errorf("suppression du schéma de test : %v", err)
		}
		_ = database.Close(admin)
	})

	schemaURL := withSearchPath(t, baseURL, schema)
	if err := database.Migrate(schemaURL); err != nil {
		t.Fatalf("migrations : %v", err)
	}

	db := open(t, schemaURL)
	t.Cleanup(func() { _ = database.Close(db) })
	return db
}

// DatabaseURL retourne l'URL d'un schéma isolé vierge, sans y appliquer les migrations. Elle
// sert à tester les migrations elles-mêmes.
func DatabaseURL(t testing.TB) string {
	t.Helper()

	baseURL := os.Getenv(EnvTestDatabaseURL)
	if baseURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s doit être défini en intégration continue", EnvTestDatabaseURL)
		}
		t.Skipf("%s non défini : test d'intégration ignoré", EnvTestDatabaseURL)
	}

	admin := open(t, baseURL)
	ensureExtension(t, admin)
	schema := schemaPrefix + randomSuffix(t)
	if err := admin.Exec(`CREATE SCHEMA ` + schema).Error; err != nil {
		t.Fatalf("création du schéma de test : %v", err)
	}
	t.Cleanup(func() {
		_ = dropSchema(admin, schema)
		_ = database.Close(admin)
	})
	return withSearchPath(t, baseURL, schema)
}

// dropSchema supprime un schéma de test. Une requête interrompue par l'arrêt du serveur testé
// (janitor, relais d'événements…) peut encore tourner côté PostgreSQL après que le client a rendu
// la main : elle prend ses verrous dans un autre ordre que DROP SCHEMA … CASCADE, et PostgreSQL
// sacrifie l'une des deux. Le DROP est alors rejoué une fois l'autre requête terminée.
func dropSchema(db *gorm.DB, schema string) error {
	var err error
	for attempt := range dropSchemaRetries {
		err = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`).Error
		var pgErr *pgconn.PgError
		if err == nil || !errors.As(err, &pgErr) || pgErr.Code != deadlockCode {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * dropSchemaBackoff)
	}
	return err
}

func open(t testing.TB, databaseURL string) *gorm.DB {
	t.Helper()
	db, err := database.Open(context.Background(), config.Database{
		URL:                databaseURL,
		MaxOpenConns:       poolSize,
		MaxIdleConns:       poolSize,
		ConnMaxLifetime:    time.Minute,
		ConnMaxIdleTime:    time.Minute,
		PingTimeout:        pingTimeout,
		SlowQueryThreshold: slowQuery,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("connexion à la base de test : %v", err)
	}
	return db
}

// ensureExtension crée citext une seule fois, même quand plusieurs binaires de test
// démarrent en même temps (CREATE EXTENSION concurrent échoue sans verrou).
func ensureExtension(t testing.TB, db *gorm.DB) {
	t.Helper()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(?)`, extensionLockKey).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE EXTENSION IF NOT EXISTS citext`).Error
	})
	if err != nil {
		t.Fatalf("extension citext : %v", err)
	}
}

func withSearchPath(t testing.TB, databaseURL, schema string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("URL de test invalide : %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func randomSuffix(t testing.TB) string {
	t.Helper()
	buffer := make([]byte, schemaRandomSize)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("tirage aléatoire : %v", err)
	}
	return hex.EncodeToString(buffer)
}
