package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/server"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

const (
	pollEvery   = 25 * time.Millisecond
	startupWait = 10 * time.Second
	stopWait    = 15 * time.Second
)

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func waitFor(t *testing.T, timeout time.Duration, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(pollEvery)
	}
	t.Fatalf("délai dépassé en attendant : %s", what)
}

func isUp(url string) bool {
	response, err := http.Get(url)
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

// TestServerLifecycle démarre l'application comme le fait cmd/api : ouverture de la base,
// migrations automatiques, requêtes réelles, puis arrêt propre sur annulation du contexte.
func TestServerLifecycle(t *testing.T) {
	address := freeAddress(t)
	cfg := testutil.Config(t, map[string]string{
		"HEROS_DATABASE_URL":          testutil.DatabaseURL(t),
		"HEROS_AUTO_MIGRATE":          "true",
		"HEROS_HTTP_ADDR":             address,
		"HEROS_RATE_REGISTER_PER_IP":  "100/1h",
		"HEROS_HTTP_SHUTDOWN_TIMEOUT": "10s",
		"HEROS_MAIL_DRIVER":           "log",
		"HEROS_MAIL_WORKERS":          "2",
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

	// Une inscription réelle traverse toute la pile : base, chiffrement, file de mails.
	body, _ := json.Marshal(registrationBody("lifecycle@example.org"))
	request, _ := http.NewRequest(http.MethodPost, baseURL+"/auth/register", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("inscription : %d", response.StatusCode)
	}

	cancel() // équivalent d'un SIGTERM
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run doit se terminer sans erreur : %v", err)
		}
	case <-time.After(stopWait):
		t.Fatal("l'arrêt ne s'est pas terminé à temps")
	}

	if isUp(baseURL + "/health") {
		t.Error("le serveur ne doit plus répondre après l'arrêt")
	}
	if err := database.Ping(context.Background(), app.DB()); err == nil {
		t.Error("la connexion à la base doit être fermée après l'arrêt")
	}
	// Le mail d'activation, mis en file avant l'arrêt, a été envoyé (driver log) : la file est vidée.
	if !bytes.Contains(logs.Bytes(), []byte("mail (non envoyé, driver log)")) {
		t.Errorf("le mail en file n'a pas été traité avant l'arrêt :\n%s", logs.String())
	}
}

func TestRunFailsFastWhenThePortIsTaken(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = occupied.Close() }()

	cfg := testutil.Config(t, map[string]string{"HEROS_HTTP_ADDR": occupied.Addr().String()})
	app, err := server.NewWithDB(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), testutil.NewDB(t))
	if err != nil {
		t.Fatal(err)
	}

	stopped := make(chan error, 1)
	go func() { stopped <- app.Run(context.Background()) }()
	select {
	case err := <-stopped:
		if err == nil {
			t.Fatal("Run doit signaler que le port est déjà utilisé")
		}
	case <-time.After(stopWait):
		t.Fatal("Run ne doit pas rester bloqué quand le port est pris")
	}
}

func TestNewRejectsAnUnreachableDatabase(t *testing.T) {
	cfg := testutil.Config(t, map[string]string{
		"HEROS_DATABASE_URL":          "postgres://heros:motdepasse-secret@127.0.0.1:1/inexistante?sslmode=disable",
		"HEROS_DATABASE_PING_TIMEOUT": "500ms",
	})
	_, err := server.New(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("New doit échouer quand la base ne répond pas")
	}
	if bytes.Contains([]byte(err.Error()), []byte("motdepasse-secret")) {
		t.Fatalf("l'erreur ne doit jamais contenir le mot de passe de la base : %v", err)
	}
}

// TestJanitorPurgesStaleSessionsAndTokens vérifie que la tâche de fond tourne réellement dans
// Run, puis s'arrête avec lui.
func TestJanitorPurgesStaleSessionsAndTokens(t *testing.T) {
	db := testutil.NewDB(t)
	address := freeAddress(t)
	cfg := testutil.Config(t, map[string]string{
		"HEROS_HTTP_ADDR": address, "HEROS_CLEANUP_INTERVAL": "50ms", "HEROS_SESSION_RETENTION": "1h", "HEROS_TOKEN_RETENTION": "1h",
	})
	app, err := server.NewWithDB(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db)
	if err != nil {
		t.Fatal(err)
	}

	// Un compte, avec une session et un jeton expirés depuis longtemps, et une session valide.
	userID := uuid.New()
	for _, query := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, email, password_hash, civility, last_name, first_name, birth_date) VALUES (?, 'janitor@example.org', 'x', 'madame', '\x00', '\x00', '\x00')`, []any{userID}},
		{`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('\x01', ?, now() - interval '30 days')`, []any{userID}},
		{`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('\x02', ?, now() + interval '30 days')`, []any{userID}},
		{`INSERT INTO user_tokens (id, user_id, type, token_hash, expires_at) VALUES (?, ?, 'activation', '\x03', now() - interval '30 days')`, []any{uuid.New(), userID}},
	} {
		if err := db.Exec(query.sql, query.args...).Error; err != nil {
			t.Fatalf("préparation : %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- app.Run(ctx) }()
	defer cancel()

	count := func(table string) int64 {
		var total int64
		db.Table(table).Count(&total)
		return total
	}
	waitFor(t, startupWait, "les sessions et jetons périmés sont supprimés", func() bool {
		return count("sessions") == 1 && count("user_tokens") == 0
	})
	if count("sessions") != 1 {
		t.Fatal("la session encore valide doit être conservée")
	}

	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run : %v", err)
		}
	case <-time.After(stopWait):
		t.Fatal("la tâche de nettoyage doit s'arrêter avec le serveur")
	}
}
