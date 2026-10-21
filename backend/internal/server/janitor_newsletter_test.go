package server_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/server"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

// TestJanitorPurgesUnconfirmedNewsletterRequests : l'adresse d'une personne qui n'a jamais confirmé son
// abonnement n'est pas conservée au-delà de la durée de rétention des jetons. Les abonnements confirmés
// ou retirés, eux, ne sont jamais touchés.
func TestJanitorPurgesUnconfirmedNewsletterRequests(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.Config(t, map[string]string{
		"HEROS_HTTP_ADDR": freeAddress(t), "HEROS_CLEANUP_INTERVAL": "50ms", "HEROS_TOKEN_RETENTION": "1h",
	})
	app, err := server.NewWithDB(cfg, slog.New(slog.DiscardHandler), db)
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range []struct{ email, status, createdAt, confirmedAt string }{
		{"abandonnee@example.org", "pending", "now() - interval '2 hours'", "NULL"},
		{"recente@example.org", "pending", "now() - interval '10 minutes'", "NULL"},
		{"confirmee@example.org", "subscribed", "now() - interval '30 days'", "now() - interval '30 days'"},
		{"retiree@example.org", "unsubscribed", "now() - interval '30 days'", "now() - interval '40 days'"},
	} {
		query := `INSERT INTO newsletter_subscriptions (id, email, status, source, created_at, confirmed_at) VALUES (?, ?, ?, 'footer', ` +
			row.createdAt + `, ` + row.confirmedAt + `)`
		if err := db.Exec(query, uuid.New(), row.email, row.status).Error; err != nil {
			t.Fatalf("préparation : %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- app.Run(ctx) }()
	defer cancel()

	remaining := func() []string {
		var emails []string
		db.Raw(`SELECT email FROM newsletter_subscriptions ORDER BY email`).Scan(&emails)
		return emails
	}
	waitFor(t, startupWait, "la demande abandonnée est supprimée", func() bool { return len(remaining()) == 3 })
	// Laisse passer quelques passages de plus : rien d'autre ne doit disparaître.
	time.Sleep(300 * time.Millisecond)
	if got := remaining(); len(got) != 3 || got[0] != "confirmee@example.org" || got[1] != "recente@example.org" || got[2] != "retiree@example.org" {
		t.Errorf("lignes restantes = %v", got)
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
