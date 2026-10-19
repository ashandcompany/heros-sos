package database

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pilote pgx5://
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/ashandcompany/heros-sos/backend/migrations"
)

const migrateScheme = "pgx5"

// Migrate applique toutes les migrations en attente. golang-migrate prend un verrou
// consultatif PostgreSQL : deux instances démarrées en même temps ne se marchent pas dessus.
func Migrate(databaseURL string) error {
	return run(databaseURL, (*migrate.Migrate).Up)
}

// Rollback annule toutes les migrations. Réservé aux tests et au développement.
func Rollback(databaseURL string) error {
	return run(databaseURL, (*migrate.Migrate).Down)
}

func run(databaseURL string, step func(*migrate.Migrate) error) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("database : lecture des migrations : %w", err)
	}

	target, err := migrateURL(databaseURL)
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, target)
	if err != nil {
		return fmt.Errorf("database : initialisation des migrations : %w", errorWithoutDetail(err))
	}
	defer func() { _, _ = migrator.Close() }()

	if err := step(migrator); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database : migration : %w", errorWithoutDetail(err))
	}
	return nil
}

// migrateURL convertit une URL postgres:// en pgx5://, schéma du pilote de migration.
func migrateURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return "", errors.New("database : URL postgres:// attendue")
	}
	parsed.Scheme = migrateScheme
	return parsed.String(), nil
}
