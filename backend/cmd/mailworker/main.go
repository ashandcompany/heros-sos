// Commande mailworker : envoie les mails que l'API dépose dans RabbitMQ.
//
//	go run ./cmd/mailworker               consomme la file « mail.send » jusqu'à SIGINT ou SIGTERM
//	go run ./cmd/mailworker -healthcheck  vérifie qu'un worker est attaché à la file (code 0 si oui)
//
// Il lit les mêmes variables d'environnement que l'API (HEROS_MAIL_*, HEROS_SMTP_*, HEROS_RABBITMQ_URL) et exige
// HEROS_MAIL_QUEUE=rabbitmq. Voir backend/stack/rabbitmq.md.
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
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
)

// logService identifie ce programme dans les journaux JSON (voir cmd/api).
const logService = "mailworker"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run() error {
	healthcheck := flag.Bool("healthcheck", false, "vérifie qu'un worker est attaché à la file puis quitte (code 0 si oui)")
	flag.Parse()

	cfg, err := config.MailWorkerFromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}

	// Une image `scratch` n'a ni shell ni outil de diagnostic : le binaire se sonde lui-même.
	if *healthcheck {
		return mail.CheckWorker(cfg.Mail.RabbitMQ)
	}
	logger := newLogger(cfg)

	transport, err := mail.NewTransport(cfg.Mail, logger)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("worker de mails démarré", "driver", cfg.Mail.Driver, "workers", cfg.Mail.Workers)
	if err := mail.NewWorker(transport, cfg.Mail, logger).Run(ctx); err != nil {
		return err
	}
	logger.Info("worker de mails arrêté")
	return nil
}

// newLogger écrit en JSON ou en texte lisible, selon HEROS_LOG_FORMAT, comme l'API.
func newLogger(cfg config.MailWorker) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == config.LogFormatJSON {
		return slog.New(slog.NewJSONHandler(os.Stdout, options)).With("service", logService)
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
