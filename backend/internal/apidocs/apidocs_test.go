package apidocs_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/apidocs"
)

const (
	apiPrefix = "/api/v1"
	spec      = "openapi: 3.0.3\ninfo:\n  title: test\n"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	apidocs.New([]byte(spec)).Register(func(method, path string, handler http.Handler) {
		mux.Handle(method+" "+apiPrefix+path, handler)
	})
	return mux
}

func get(t *testing.T, mux http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func TestPageIsServedWithItsOwnContentSecurityPolicy(t *testing.T) {
	response := get(t, newMux(), apiPrefix+apidocs.PagePath)

	if response.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q", contentType)
	}
	policy := response.Header().Get("Content-Security-Policy")
	for _, required := range []string{"default-src 'none'", "script-src 'self'", "connect-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(policy, required) {
			t.Errorf("la politique %q doit contenir %q", policy, required)
		}
	}
}

func TestPolicyAllowsNeitherInlineScriptsNorEvalNorOutsideHosts(t *testing.T) {
	policy := get(t, newMux(), apiPrefix+apidocs.PagePath).Header().Get("Content-Security-Policy")

	directives := map[string]string{}
	for _, directive := range strings.Split(policy, ";") {
		name, sources, _ := strings.Cut(strings.TrimSpace(directive), " ")
		directives[name] = sources
	}
	if strings.Contains(directives["script-src"], "unsafe") {
		t.Errorf("script-src = %q : aucun script en ligne ni eval", directives["script-src"])
	}
	for name, sources := range directives {
		if strings.Contains(sources, "*") || strings.Contains(sources, "http:") || strings.Contains(sources, "https:") {
			t.Errorf("%s = %q : aucune source extérieure", name, sources)
		}
	}
}

func TestOnlyThePageRelaxesThePolicy(t *testing.T) {
	// Les autres réponses gardent la politique de l'API, posée par le serveur : elles ne doivent pas
	// définir la leur.
	for _, path := range []string{apidocs.SpecPath, apidocs.InitializerPath, "/docs/assets/swagger-ui.css"} {
		response := get(t, newMux(), apiPrefix+path)
		if response.Code != http.StatusOK {
			t.Fatalf("%s : statut %d", path, response.Code)
		}
		if policy := response.Header().Get("Content-Security-Policy"); policy != "" {
			t.Errorf("%s définit sa propre politique : %q", path, policy)
		}
	}
}

func TestPageLoadsNothingFromOutsideAndRunsNoInlineScript(t *testing.T) {
	body := get(t, newMux(), apiPrefix+apidocs.PagePath).Body.String()

	if external := regexp.MustCompile(`(?:src|href)="(?:https?:)?//`).FindString(body); external != "" {
		t.Errorf("la page charge une ressource extérieure : %s", external)
	}
	// Un script est « en ligne » s'il n'a pas d'attribut src, ou s'il a un contenu malgré son src.
	for _, script := range regexp.MustCompile(`(?s)<script([^>]*)>(.*?)</script>`).FindAllStringSubmatch(body, -1) {
		if !strings.Contains(script[1], "src=") || strings.TrimSpace(script[2]) != "" {
			t.Errorf("la page contient un script en ligne : %s", script[0])
		}
	}
	if strings.Contains(body, "onclick=") || strings.Contains(body, "onload=") {
		t.Error("la page contient un gestionnaire d'événement en ligne")
	}
}

// Les adresses de index.html et de initializer.js sont relatives à la page : chacune doit mener à
// une route qui existe, derrière le préfixe de l'API comme derrière le reverse proxy.
func TestEveryAddressReferencedByThePageResolves(t *testing.T) {
	mux := newMux()
	pageURL, err := url.Parse("http://localhost" + apiPrefix + apidocs.PagePath)
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, mux, pageURL.Path).Body.String()
	initializer := get(t, mux, apiPrefix+apidocs.InitializerPath).Body.String()

	references := regexp.MustCompile(`(?:src|href)="([^"]+)"`).FindAllStringSubmatch(page, -1)
	references = append(references, regexp.MustCompile(`url:\s*"([^"]+)"`).FindAllStringSubmatch(initializer, -1)...)
	if len(references) < 4 {
		t.Fatalf("%d références trouvées, attendu au moins 4 (icône, style, bundle, initialiseur) + le contrat", len(references))
	}

	for _, reference := range references {
		target, err := url.Parse(reference[1])
		if err != nil {
			t.Fatal(err)
		}
		resolved := pageURL.ResolveReference(target)
		if response := get(t, mux, resolved.Path); response.Code != http.StatusOK {
			t.Errorf("%s (résolu en %s) : statut %d", reference[1], resolved.Path, response.Code)
		}
	}
}

func TestSpecIsServedAsIs(t *testing.T) {
	response := get(t, newMux(), apiPrefix+apidocs.SpecPath)

	if response.Code != http.StatusOK || response.Body.String() != spec {
		t.Fatalf("statut %d, corps %q", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "yaml") {
		t.Errorf("Content-Type = %q", contentType)
	}
}

func TestAssetsHaveTheirContentType(t *testing.T) {
	for file, want := range map[string]string{
		"swagger-ui-bundle.js": "javascript",
		"swagger-ui.css":       "text/css",
		"favicon-32x32.png":    "image/png",
		"LICENSE":              "text/plain",
	} {
		response := get(t, newMux(), apiPrefix+"/docs/assets/"+file)
		if response.Code != http.StatusOK {
			t.Errorf("%s : statut %d", file, response.Code)
			continue
		}
		if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, want) {
			t.Errorf("%s : Content-Type = %q, attendu %q", file, contentType, want)
		}
	}
}

func TestUnknownOrOutOfFolderAssetsAreRefused(t *testing.T) {
	for _, file := range []string{
		"inconnu.js",
		"index.html",       // dans le paquet, pas dans assets/
		"..%2Fapidocs.go",  // « ../apidocs.go » : un seul segment, qui sort du dossier
		"..%2F..%2Fgo.mod", // idem, plus haut
		"%2e%2e",           // « .. »
	} {
		response := get(t, newMux(), apiPrefix+"/docs/assets/"+file)
		if response.Code == http.StatusOK {
			t.Errorf("%s : servi (%d octets)", file, response.Body.Len())
		}
	}

	response := get(t, newMux(), apiPrefix+"/docs/assets/inconnu.js")
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"not_found"`) {
		t.Errorf("fichier inconnu : statut %d, corps %q (attendu l'erreur JSON de l'API)", response.Code, response.Body.String())
	}
}

func TestOnlyReadingIsAllowed(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		newMux().ServeHTTP(recorder, httptest.NewRequest(method, apiPrefix+apidocs.PagePath, nil))
		if recorder.Code == http.StatusOK {
			t.Errorf("%s est accepté sur la page", method)
		}
	}
}

// Les fichiers de Swagger UI sont ceux de la distribution officielle, sans modification (voir
// THIRD_PARTY.md). Ce test échoue si l'un d'eux est modifié par erreur ; pour mettre à jour la
// version, remplacer les fichiers et ces empreintes ensemble.
func TestSwaggerUIFilesAreUnmodified(t *testing.T) {
	expected := map[string]string{
		"swagger-ui-bundle.js":             "3f6abf70480db7e07914df4c98a77385f22ea585eaf457a3e6415ba0eae2d167",
		"swagger-ui.css":                   "c3cfe8248c4b0b89673600fb829924eafbac6de4bc4b2b06f5f5243483074d0a",
		"favicon-32x32.png":                "3ed612f41e050ca5e7000cad6f1cbe7e7da39f65fca99c02e99e6591056e5837",
		"LICENSE":                          "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30",
		"NOTICE":                           "0d20d1adef18aee3f40dd258172155521ce702ac445cb5f7b7d60ed32dad2fb2",
		"swagger-ui-bundle.js.LICENSE.txt": "e5a4792feaa5332dfba37f1d7adf84308b93b4f2d8da0569a9f48a3cb61c969b",
	}
	for file, want := range expected {
		response := get(t, newMux(), apiPrefix+"/docs/assets/"+file)
		if response.Code != http.StatusOK {
			t.Errorf("%s : statut %d", file, response.Code)
			continue
		}
		sum := sha256.Sum256(response.Body.Bytes())
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s : sha256 %s, attendu %s (fichier modifié ?)", file, got, want)
		}
	}
}
