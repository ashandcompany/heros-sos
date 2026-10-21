package httpx

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const forwardedForHeader = "X-Forwarded-For"

type clientIPKey struct{}

// ClientIP détermine l'adresse IP du client. L'en-tête X-Forwarded-For n'est cru que si la
// connexion vient d'un proxy de confiance : on le parcourt alors de droite à gauche et on
// retient la première adresse qui n'est pas elle-même un proxy de confiance. Un client ne
// peut donc pas se faire passer pour une autre adresse en forgeant l'en-tête.
func ClientIP(r *http.Request, trustedProxies []netip.Prefix) netip.Addr {
	remote := remoteAddr(r.RemoteAddr)
	if !remote.IsValid() || !isTrusted(remote, trustedProxies) {
		return remote
	}

	forwarded := r.Header.Values(forwardedForHeader)
	var hops []string
	for _, value := range forwarded {
		for _, hop := range strings.Split(value, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(hops[i])
		if err != nil {
			return remote // en-tête mal formé : on ne lui fait pas confiance
		}
		addr = addr.Unmap()
		if !isTrusted(addr, trustedProxies) {
			return addr
		}
	}
	return remote
}

func remoteAddr(hostPort string) netip.Addr {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}

func isTrusted(addr netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// WithClientIP place l'adresse du client dans le contexte de la requête.
func WithClientIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), clientIPKey{}, ClientIP(r, trustedProxies))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClientIPFrom retourne l'adresse du client déterminée par WithClientIP (invalide sinon).
func ClientIPFrom(ctx context.Context) netip.Addr {
	addr, _ := ctx.Value(clientIPKey{}).(netip.Addr)
	return addr
}

// unknownClientKey remplace l'adresse d'un client dont l'IP est inconnue (connexion sans adresse
// exploitable) : ces requêtes partagent alors une même limite, la plus prudente.
const unknownClientKey = "unknown"

// ClientKey identifie le client d'une requête pour la limitation de débit : son adresse IP.
func ClientKey(r *http.Request) string {
	if ip := ClientIPFrom(r.Context()); ip.IsValid() {
		return "ip:" + ip.String()
	}
	return "ip:" + unknownClientKey
}
