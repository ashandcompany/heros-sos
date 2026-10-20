package server_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/server"
)

const probeTimeout = 500 * time.Millisecond

// readyServer répond à la sonde comme l'API : /api/v1/ready avec le statut donné.
func readyServer(t *testing.T, status int) (address string, port string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(server.APIPrefix+"/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return net.JoinHostPort(host, port), port
}

func probeConfig(addr string) config.HTTP {
	return config.HTTP{Addr: addr, HealthcheckTimeout: probeTimeout}
}

func TestProbeSucceedsWhenTheServerIsReady(t *testing.T) {
	address, _ := readyServer(t, http.StatusOK)
	if err := server.Probe(context.Background(), probeConfig(address)); err != nil {
		t.Fatalf("Probe : %v", err)
	}
}

func TestProbeUnderstandsEveryListenAddressForm(t *testing.T) {
	_, port := readyServer(t, http.StatusOK)

	// L'API écoute souvent sur « toutes les interfaces » : la sonde doit viser la boucle locale.
	for _, addr := range []string{":" + port, "0.0.0.0:" + port, "[::]:" + port, "127.0.0.1:" + port, "localhost:" + port} {
		if err := server.Probe(context.Background(), probeConfig(addr)); err != nil {
			t.Errorf("adresse %q : %v", addr, err)
		}
	}
}

func TestProbeFailsWhenTheServerIsNotReady(t *testing.T) {
	address, _ := readyServer(t, http.StatusServiceUnavailable)
	err := server.Probe(context.Background(), probeConfig(address))
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("erreur = %v, attendu un échec mentionnant 503", err)
	}
}

func TestProbeDoesNotFollowRedirects(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(server.APIPrefix+"/ready", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ailleurs", http.StatusFound)
	})
	mux.HandleFunc("/ailleurs", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := server.Probe(context.Background(), probeConfig(srv.Listener.Addr().String())); err == nil {
		t.Fatal("une redirection n'est pas une réponse « prêt »")
	}
}

func TestProbeFailsWhenNothingListens(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	if err := server.Probe(context.Background(), probeConfig(address)); err == nil {
		t.Fatal("aucun serveur n'écoute : la sonde doit échouer")
	}
}

func TestProbeGivesUpWithinTheConfiguredTimeout(t *testing.T) {
	release := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc(server.APIPrefix+"/ready", func(http.ResponseWriter, *http.Request) { <-release })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer close(release)

	started := time.Now()
	err := server.Probe(context.Background(), probeConfig(srv.Listener.Addr().String()))
	if err == nil {
		t.Fatal("un serveur qui ne répond pas doit faire échouer la sonde")
	}
	if elapsed := time.Since(started); elapsed > 5*probeTimeout {
		t.Fatalf("la sonde a ignoré son délai (%v pour %v configurés)", elapsed, probeTimeout)
	}
}

func TestProbeRejectsAMalformedListenAddress(t *testing.T) {
	for _, addr := range []string{"pas-une-adresse", "8080", ""} {
		if err := server.Probe(context.Background(), probeConfig(addr)); err == nil {
			t.Errorf("l'adresse %q doit être refusée", addr)
		}
	}
}

// TestProbeAgainstTheRealServer vérifie la sonde contre l'application complète : /ready doit
// répondre 200 quand la base est joignable.
func TestProbeAgainstTheRealServer(t *testing.T) {
	e := newE2E(t, nil)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(e.http.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := probeConfig("127.0.0.1:" + port)
	if err := server.Probe(context.Background(), cfg); err != nil {
		t.Fatalf("la sonde doit réussir contre l'API réelle : %v", err)
	}
}
