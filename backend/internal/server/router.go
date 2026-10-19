package server

import (
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// APIPrefix préfixe toutes les routes de l'API (valeur prévue par le README du frontend).
const APIPrefix = "/api/v1"

// Route décrit une route enregistrée. Path est relatif à APIPrefix.
type Route struct {
	Method string
	Path   string
}

// Router enregistre les routes de l'API et en garde la liste, pour vérifier qu'elles
// correspondent au contrat OpenAPI. Chaque domaine y ajoute ses routes dans sa propre fonction
// d'enregistrement (voir routes.go).
type Router struct {
	mux    *http.ServeMux
	routes []Route
}

func newRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// Handle enregistre un handler pour une méthode et un chemin relatif à APIPrefix. Les
// paramètres de chemin s'écrivent {nom} et se lisent avec r.PathValue("nom").
func (r *Router) Handle(method, path string, handler http.Handler) {
	r.mux.Handle(method+" "+APIPrefix+path, handler)
	r.routes = append(r.routes, Route{Method: method, Path: path})
}

// HandleFunc est la variante de Handle pour une fonction.
func (r *Router) HandleFunc(method, path string, handler http.HandlerFunc) {
	r.Handle(method, path, handler)
}

// Routes retourne les routes enregistrées.
func (r *Router) Routes() []Route {
	return append([]Route(nil), r.routes...)
}

// ServeHTTP répond aux requêtes. Les routes inconnues et les méthodes non permises produisent des
// erreurs JSON au format de l'API, à la place des pages texte de la bibliothèque standard.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if _, pattern := r.mux.Handler(req); pattern != "" {
		// ServeHTTP du mux (et non le handler retourné par Handler) : c'est lui qui renseigne
		// les paramètres de chemin lus par r.PathValue.
		r.mux.ServeHTTP(w, req)
		return
	}

	// Aucune route ne correspond : le mux sait distinguer « chemin inconnu » (404) de « chemin
	// connu avec une autre méthode » (405, avec l'en-tête Allow). On lui laisse décider sur une
	// réponse jetable, puis on répond nous-mêmes en JSON.
	probe := &probeWriter{header: make(http.Header)}
	r.mux.ServeHTTP(probe, req)
	if probe.status == http.StatusMethodNotAllowed {
		w.Header().Set("Allow", probe.header.Get("Allow"))
		httpx.WriteError(w, httpx.NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Méthode non autorisée pour cette route."))
		return
	}
	httpx.WriteError(w, httpx.NotFound("Route introuvable."))
}

// probeWriter capture le statut et les en-têtes d'une réponse sans rien envoyer.
type probeWriter struct {
	header http.Header
	status int
}

func (p *probeWriter) Header() http.Header         { return p.header }
func (p *probeWriter) Write(b []byte) (int, error) { return len(b), nil }
func (p *probeWriter) WriteHeader(status int)      { p.status = status }
