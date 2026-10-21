// Commande api : serveur HTTP de l'API Heros-SOS.
//
//	go run ./cmd/api               démarre le serveur
//	go run ./cmd/api -migrate      applique les migrations SQL puis quitte
//	go run ./cmd/api -healthcheck  interroge /ready (code 0 si prêt) : contrôle de santé du conteneur
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run() error {
	migrateOnly := flag.Bool("migrate", false, "applique les migrations SQL puis quitte")
	healthcheck := flag.Bool("healthcheck", false, "interroge /ready puis quitte (code 0 si le serveur est prêt)")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}

	// Une image `scratch` n'a ni shell ni curl : le binaire se sonde lui-même.
	if *healthcheck {
		return server.Probe(context.Background(), cfg.HTTP)
	}
	logger := newLogger(cfg)

	if *migrateOnly {
		if err := database.Migrate(cfg.Database.URL); err != nil {
			return err
		}
		logger.Info("migrations appliquées")
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := server.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

// newLogger écrit en JSON en production (exploitable par un agrégateur de logs) et en texte
// lisible sinon.
func newLogger(cfg config.Config) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
