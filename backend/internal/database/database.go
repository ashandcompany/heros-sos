// Package database ouvre la connexion PostgreSQL (GORM), applique les migrations et fournit
// les transactions utilisées par les services.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// Codes d'erreur SQLSTATE traités par l'application.
const uniqueViolationCode = "23505"

// Open ouvre le pool de connexions et vérifie que la base répond. L'URL de connexion n'est
// jamais incluse dans les erreurs ni dans les logs : elle contient le mot de passe.
func Open(ctx context.Context, cfg config.Database, logger *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		Logger:  newLogger(logger, cfg.SlowQueryThreshold),
		NowFunc: func() time.Time { return time.Now().UTC() },
		// Les écritures simples sont déjà atomiques ; les transactions sont explicites
		// (voir Transactor), ce qui évite un BEGIN/COMMIT inutile par requête.
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database : ouverture de la connexion : %w", errorWithoutDetail(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database : accès au pool : %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database : la base ne répond pas : %w", errorWithoutDetail(err))
	}
	return db, nil
}

// Ping vérifie que la base répond dans le délai du contexte.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close ferme le pool de connexions.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// IsUniqueViolation indique si err est une violation de la contrainte d'unicité nommée.
// Un nom vide accepte n'importe quelle contrainte d'unicité.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationCode {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

// errorWithoutDetail retire d'une erreur PostgreSQL le champ Detail, qui peut contenir des
// valeurs de colonnes (adresses email, par exemple) : seul le code SQLSTATE est conservé.
func errorWithoutDetail(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("erreur PostgreSQL %s (%s)", pgErr.Code, pgErr.ConstraintName)
	}
	return err
}

// newLogger adapte slog à GORM sans jamais journaliser le SQL exécuté, qui contient les
// valeurs des paramètres (données personnelles, empreintes de jetons).
func newLogger(logger *slog.Logger, slowThreshold time.Duration) gormlogger.Interface {
	return &slogAdapter{logger: logger, slowThreshold: slowThreshold}
}

type slogAdapter struct {
	logger        *slog.Logger
	slowThreshold time.Duration
}

func (a *slogAdapter) LogMode(gormlogger.LogLevel) gormlogger.Interface { return a }

func (a *slogAdapter) Info(ctx context.Context, msg string, _ ...any) {
	a.logger.InfoContext(ctx, "gorm : "+msg)
}

func (a *slogAdapter) Warn(ctx context.Context, msg string, _ ...any) {
	a.logger.WarnContext(ctx, "gorm : "+msg)
}

func (a *slogAdapter) Error(ctx context.Context, msg string, _ ...any) {
	a.logger.ErrorContext(ctx, "gorm : "+msg)
}

func (a *slogAdapter) Trace(ctx context.Context, begin time.Time, _ func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !IsUniqueViolation(err, ""):
		a.logger.WarnContext(ctx, "requête en erreur", "elapsed", elapsed, "error", errorWithoutDetail(err))
	case elapsed > a.slowThreshold:
		a.logger.WarnContext(ctx, "requête lente", "elapsed", elapsed)
	}
}
