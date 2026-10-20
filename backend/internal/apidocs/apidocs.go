// Package apidocs sert la documentation interactive de l'API (Swagger UI) et son contrat OpenAPI.
//
// Tout est embarqué dans le binaire : la page n'appelle aucun service extérieur (ni CDN, ni
// validateur en ligne). Les fichiers de Swagger UI (assets/) sont ceux, non modifiés, de la
// distribution officielle : voir THIRD_PARTY.md. La documentation est réservée au développement,
// c'est au serveur de ne pas l'enregistrer en production.
package apidocs

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// Chemins des routes, relatifs à la racine de l'API. index.html et initializer.js y font référence
// par des adresses relatives : un test vérifie qu'elles se résolvent toutes.
const (
	PagePath        = "/docs"
	SpecPath        = "/docs/openapi.yaml"
	InitializerPath = "/docs/initializer.js"
	AssetsPath      = "/docs/assets/{" + AssetParam + "}"

	// AssetParam est le nom du paramètre de chemin de AssetsPath.
	AssetParam = "file"
)

// pageContentSecurityPolicy est la politique de la seule page HTML : plus souple que celle des
// réponses de l'API (« default-src 'none' »), mais sans scripts en ligne, sans eval, sans ressource
// extérieure. Swagger UI a besoin de styles en ligne et d'images « data: » (icônes SVG).
const pageContentSecurityPolicy = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

const (
	contentTypeHTML       = "text/html; charset=utf-8"
	contentTypeJavaScript = "text/javascript; charset=utf-8"
	contentTypeYAML       = "application/yaml"
)

//go:embed index.html
var indexHTML []byte

//go:embed initializer.js
var initializerJS []byte

//go:embed assets
var embeddedAssets embed.FS

// Docs sert la page de documentation, le contrat et les fichiers de Swagger UI.
type Docs struct {
	spec   []byte
	assets fs.FS
}

// New crée la documentation de l'API décrite par spec (le contenu de openapi.yaml).
func New(spec []byte) *Docs {
	// Le dossier assets existe toujours (go:embed échoue à la compilation sinon) : l'erreur de Sub
	// est impossible.
	assets, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic("apidocs : " + err.Error())
	}
	return &Docs{spec: spec, assets: assets}
}

// Register déclare les routes de la documentation avec handle (méthode, chemin relatif à la racine
// de l'API, handler) : le serveur et les tests utilisent ainsi exactement les mêmes routes.
func (d *Docs) Register(handle func(method, path string, handler http.Handler)) {
	handle(http.MethodGet, PagePath, http.HandlerFunc(d.Page))
	handle(http.MethodGet, SpecPath, http.HandlerFunc(d.Spec))
	handle(http.MethodGet, InitializerPath, http.HandlerFunc(d.Initializer))
	handle(http.MethodGet, AssetsPath, http.HandlerFunc(d.Asset))
}

// Page répond par la page HTML. C'est la seule réponse dont la politique de sécurité est assouplie.
func (d *Docs) Page(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Security-Policy", pageContentSecurityPolicy)
	write(w, contentTypeHTML, indexHTML)
}

// Spec répond par le contrat OpenAPI.
func (d *Docs) Spec(w http.ResponseWriter, _ *http.Request) {
	write(w, contentTypeYAML, d.spec)
}

// Initializer répond par le script qui démarre Swagger UI.
func (d *Docs) Initializer(w http.ResponseWriter, _ *http.Request) {
	write(w, contentTypeJavaScript, initializerJS)
}

// Asset répond par un fichier de Swagger UI. Le paramètre de chemin est un seul segment : il ne peut
// pas désigner un autre dossier, et seuls les fichiers embarqués existent.
func (d *Docs) Asset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue(AssetParam)
	if info, err := fs.Stat(d.assets, name); err != nil || info.IsDir() {
		httpx.WriteError(w, httpx.NotFound("Fichier introuvable."))
		return
	}
	http.ServeFileFS(w, r, d.assets, name)
}

func write(w http.ResponseWriter, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(body)
}
