// Commande seed : remplit une base de développement avec des données de démonstration tirées de
// l'univers Marvel (modérateurs, super-héros, citoyens, vilains, incidents, interventions, avis,
// films, articles et abonnés à la newsletter).
//
// Tous les comptes partagent le mot de passe lu dans HEROS_SEED_PASSWORD (Avengers2026! par
// défaut). La commande refuse de tourner en production, et sur une base déjà remplie sans -reset.
//
//	go run ./cmd/seed          # base vide (migrations appliquées)
//	go run ./cmd/seed -reset   # vide d'abord toutes les tables de données
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/film"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const (
	passwordEnv     = "HEROS_SEED_PASSWORD"
	defaultPassword = "Avengers2026!"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	reset := flag.Bool("reset", false, "vide toutes les tables de données avant de les remplir")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}
	if cfg.IsProduction() {
		return errors.New("seed refusé en production : ces données sont fictives et le mot de passe est public")
	}

	password := os.Getenv(passwordEnv)
	if password == "" {
		password = defaultPassword
	}
	if code := validation.Password(password, validation.PasswordPolicy{
		MinLength: cfg.Security.Password.MinLength, MaxLength: cfg.Security.Password.MaxLength,
	}); code != "" {
		return fmt.Errorf("mot de passe %s refusé par la politique de mots de passe (%s)", passwordEnv, code)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	sealer, err := crypto.NewSealer(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		return err
	}
	// Un seul hachage pour tous les comptes : argon2id est volontairement lent.
	hash, err := auth.NewPasswordHasher(cfg.Security.Argon2).Hash(ctx, password)
	if err != nil {
		return fmt.Errorf("hachage du mot de passe : %w", err)
	}

	s := &seeder{
		db:           db,
		users:        user.NewRepository(db, sealer),
		passwordHash: hash,
	}
	// Affiches et résumés depuis OMDb quand une clé est configurée ; sinon, les textes du seed.
	if cfg.OMDb.APIKey != "" {
		s.catalog = film.NewOMDbClient(cfg.OMDb.BaseURL, cfg.OMDb.APIKey, cfg.OMDb.Timeout)
	}

	stats, err := s.run(ctx, *reset)
	if err != nil {
		return err
	}
	stats.print(os.Stdout, password)
	return nil
}
