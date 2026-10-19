// Commande createmoderator : crée un compte modérateur actif.
//
// Le mot de passe est lu dans la variable HEROS_MODERATOR_PASSWORD, jamais dans un argument
// (les arguments restent visibles dans la liste des processus et l'historique du shell).
//
//	HEROS_MODERATOR_PASSWORD='…' go run ./cmd/createmoderator \
//	    -email modo@example.org -first-name Nick -last-name Fury -birth-date 1950-12-04
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
	"github.com/ashandcompany/heros-sos/backend/internal/bootstrap"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const passwordEnv = "HEROS_MODERATOR_PASSWORD"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var input bootstrap.ModeratorInput
	flag.StringVar(&input.Email, "email", "", "adresse email du modérateur")
	flag.StringVar(&input.FirstName, "first-name", "", "prénom")
	flag.StringVar(&input.LastName, "last-name", "", "nom")
	flag.StringVar(&input.BirthDate, "birth-date", "", "date de naissance (AAAA-MM-JJ)")
	flag.Parse()
	input.Password = os.Getenv(passwordEnv)

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
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
	moderator, err := bootstrap.CreateModerator(ctx, bootstrap.ModeratorDeps{
		Users:    user.NewRepository(db, sealer),
		Hasher:   auth.NewPasswordHasher(cfg.Security.Argon2),
		Clock:    clock.System{},
		Password: cfg.Security.Password,
	}, input)

	var invalid *validation.Error
	switch {
	case errors.As(err, &invalid):
		return fmt.Errorf("%w (le mot de passe se passe par %s)", invalid, passwordEnv)
	case errors.Is(err, user.ErrEmailTaken):
		return fmt.Errorf("un compte existe déjà pour %s", input.Email)
	case err != nil:
		return err
	}
	fmt.Printf("modérateur créé : %s (%s)\n", moderator.Email, moderator.ID)
	return nil
}
