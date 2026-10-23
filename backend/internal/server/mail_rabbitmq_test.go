package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/server"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

// TestServerStaysUpWhenRabbitMQIsDown : la file de mails RabbitMQ est un confort, pas une dépendance du
// service. Avec un courtier injoignable, l'API démarre, répond, accepte une inscription (le mail d'activation
// est refusé et journalisé, l'utilisateur peut le redemander) et s'arrête proprement sans attendre le courtier.
func TestServerStaysUpWhenRabbitMQIsDown(t *testing.T) {
	address := freeAddress(t)
	cfg := testutil.Config(t, map[string]string{
		"HEROS_DATABASE_URL":             testutil.DatabaseURL(t),
		"HEROS_AUTO_MIGRATE":             "true",
		"HEROS_HTTP_ADDR":                address,
		"HEROS_HTTP_SHUTDOWN_TIMEOUT":    "5s",
		"HEROS_MAIL_QUEUE":               "rabbitmq",
		"HEROS_RABBITMQ_URL":             "amqp://heros:heros@" + freeAddress(t) + "/",
		"HEROS_RABBITMQ_RECONNECT_DELAY": "50ms",
	})
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app, err := server.New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("New : %v", err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- app.Run(ctx) }()

	baseURL := "http://" + address + server.APIPrefix
	waitFor(t, startupWait, "le serveur répond", func() bool { return isUp(baseURL + "/ready") })

	body, _ := json.Marshal(registrationBody("sans-courtier@example.org"))
	request, _ := http.NewRequest(http.MethodPost, baseURL+"/auth/register", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("inscription sans courtier : %d, attendu %d", response.StatusCode, http.StatusCreated)
	}

	started := time.Now()
	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run doit se terminer sans erreur : %v", err)
		}
	case <-time.After(stopWait):
		t.Fatal("l'arrêt ne s'est pas terminé à temps")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Errorf("l'arrêt a pris %v : il ne doit pas attendre un courtier absent", elapsed)
	}

	for _, want := range []string{"RabbitMQ indisponible", "envoi du mail d'activation impossible"} {
		if !bytes.Contains(logs.Bytes(), []byte(want)) {
			t.Errorf("journal sans %q :\n%s", want, logs.String())
		}
	}
	if bytes.Contains(logs.Bytes(), []byte("heros:heros@")) {
		t.Errorf("le journal contient l'URL du courtier, donc son mot de passe :\n%s", logs.String())
	}
}
