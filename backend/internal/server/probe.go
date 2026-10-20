package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

const (
	// loopbackHost remplace une adresse d'écoute « toutes interfaces » : la sonde interroge
	// toujours le processus depuis l'intérieur de son conteneur.
	loopbackHost = "127.0.0.1"
	readyPath    = "/ready"
	// maxProbeBody borne ce que la sonde lit d'une réponse, pour ne jamais être retenue par un
	// corps interminable.
	maxProbeBody = 1 << 10
)

// Probe interroge la sonde de disponibilité /ready du serveur local et échoue si elle ne répond
// pas 200 dans le délai configuré. Il alimente le contrôle de santé du conteneur
// (`api -healthcheck`) : une image `scratch` n'a ni shell ni curl, le binaire se sonde lui-même.
func Probe(ctx context.Context, cfg config.HTTP) error {
	target, err := probeURL(cfg.Addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.HealthcheckTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("server : sonde : %w", err)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("server : sonde injoignable : %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxProbeBody))

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server : la sonde répond %d", response.StatusCode)
	}
	return nil
}

// probeURL construit l'URL de la sonde à partir de l'adresse d'écoute (":8080", "0.0.0.0:8080",
// "127.0.0.1:9000"…).
func probeURL(listenAddr string) (string, error) {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "", fmt.Errorf("server : adresse d'écoute invalide %q : %w", listenAddr, err)
	}
	if ip, parseErr := netip.ParseAddr(host); host == "" || (parseErr == nil && ip.IsUnspecified()) {
		host = loopbackHost
	}
	return "http://" + net.JoinHostPort(host, port) + APIPrefix + readyPath, nil
}
