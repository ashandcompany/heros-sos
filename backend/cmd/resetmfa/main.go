// Commande resetmfa : supprime la double authentification d'un compte (appareil perdu, codes de secours
// épuisés) et ferme toutes ses sessions. C'est la seule façon de le faire : l'API ne le permet pas, pour
// qu'un compte piraté ne puisse pas retirer sa propre protection.
//
//	go run ./cmd/resetmfa -email modo@example.org
//
// Le compte réactive sa 2FA à sa prochaine connexion (obligatoire pour un modérateur en production). À
// n'utiliser qu'après avoir vérifié l'identité de la personne : réinitialiser la 2FA d'un compte, c'est
// réduire sa protection au mot de passe jusque-là.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/bootstrap"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var email string
	flag.StringVar(&email, "email", "", "adresse email du compte")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}
	logger := slog.New(slog.DiscardHandler)
	db, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	sealer, err := crypto.NewSealer(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		return err
	}
	clk := clock.System{}
	account, err := bootstrap.ResetMFA(ctx, bootstrap.ResetMFADeps{
		Users:    user.NewRepository(db, sealer),
		MFA:      auth.NewMFAStore(db, sealer),
		Sessions: auth.NewSessionService(auth.NewSessionStore(db), clk, cfg.Session, cfg.Security.MFA, logger),
		Audit:    audit.NewStore(db, clk),
		Tx:       database.NewTransactor(db),
		Clock:    clk,
	}, email)

	var invalid *validation.Error
	switch {
	case errors.As(err, &invalid):
		return fmt.Errorf("%w (indiquez l'adresse du compte avec -email)", invalid)
	case errors.Is(err, user.ErrNotFound):
		return fmt.Errorf("aucun compte ne correspond à %s", email)
	case err != nil:
		return err
	}
	fmt.Printf("double authentification réinitialisée : %s (%s), sessions fermées\n", account.Email, account.ID)
	return nil
}
