package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// probeRoutes doit reprendre les motifs que le routeur note vraiment : une faute de frappe rendrait le filtre
// sans effet, sans qu'aucun test ne le voie.
func TestProbeRoutesMatchWhatTheRouterRecords(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	router := newRouter()
	registerHealthRoutes(router, &components{})
	handler := httpx.AccessLog(logger, probeRoutes...)(router)

	// Seule la sonde live répond sans base : la sonde ready partage la même construction de motif.
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, APIPrefix+livePath, nil))

	if output := buffer.String(); !strings.Contains(output, "level=DEBUG") {
		t.Errorf("la sonde live doit être journalisée en DEBUG :\n%s", output)
	}
	want := http.MethodGet + " " + APIPrefix + readyPath
	found := false
	for _, pattern := range probeRoutes {
		found = found || pattern == want
	}
	if !found {
		t.Errorf("probeRoutes = %q, doit contenir %q", probeRoutes, want)
	}
}
