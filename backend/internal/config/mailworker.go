package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

// MailWorker est la configuration du worker de mails (commande mailworker). De tout ce que lit l'API, il ne
// lui faut que l'environnement, les journaux et les mails : il ne reçoit donc ni l'accès à la base ni les clés
// de chiffrement, que son travail (relire un mail dans RabbitMQ, le remettre au SMTP) n'exige pas.
type MailWorker struct {
	Environment Environment
	LogLevel    slog.Level
	LogFormat   LogFormat
	Mail        Mail
}

// MailWorkerFromEnv construit la configuration du worker depuis les variables d'environnement du processus.
func MailWorkerFromEnv() (MailWorker, error) {
	return LoadMailWorker(os.Getenv)
}

// LoadMailWorker construit la configuration du worker à partir de getenv. Les variables sont les mêmes que
// celles de l'API : un seul fichier d'environnement décrit les deux processus. Toutes les erreurs de lecture
// et de validation sont retournées ensemble.
func LoadMailWorker(getenv func(string) string) (MailWorker, error) {
	l := &loader{getenv: getenv}
	environment := l.environment()
	production := environment == EnvironmentProduction
	cfg := MailWorker{
		Environment: environment,
		LogLevel:    l.logLevel(),
		LogFormat:   l.logFormat(production),
		Mail:        l.mail(production),
	}

	errs := l.errs
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	Config{Environment: environment, Mail: cfg.Mail}.validateMail(fail, production)
	if cfg.Mail.Queue != MailQueueRabbitMQ {
		fail("%s doit valoir %s : avec la file en mémoire, l'API envoie elle-même les mails et il n'y a rien à consommer",
			envMailQueue, MailQueueRabbitMQ)
	}
	return cfg, errors.Join(errs...)
}
