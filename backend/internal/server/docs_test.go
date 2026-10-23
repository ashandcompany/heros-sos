package server_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/ashandcompany/heros-sos/backend/internal/apidocs"
	"github.com/ashandcompany/heros-sos/backend/internal/server"
	"github.com/ashandcompany/heros-sos/backend/openapi"
)

// apiPolicy est la politique de sécurité de toutes les réponses de l'API (voir httpx.SecurityHeaders).
const apiPolicy = "default-src 'none'; frame-ancestors 'none'"

// plainResponse est une réponse déjà lue : son corps est fermé, il n'y a rien à libérer.
type plainResponse struct {
	status int
	header http.Header
	body   []byte
}

// getPlain lit une URL sans passer par le contrat OpenAPI : la documentation n'en fait pas partie.
func (e *e2e) getPlain(path string) plainResponse {
	e.t.Helper()
	response, err := http.Get(e.http.URL + path)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		e.t.Fatal(err)
	}
	return plainResponse{status: response.StatusCode, header: response.Header, body: body}
}

func TestDocsAreServedByTheRealApplication(t *testing.T) {
	e := newE2E(t, nil)

	page := e.getPlain(server.APIPrefix + apidocs.PagePath)
	if page.status != http.StatusOK || !strings.Contains(string(page.body), "swagger-ui") {
		t.Fatalf("page : statut %d, corps %.80q", page.status, page.body)
	}
	if policy := page.header.Get("Content-Security-Policy"); !strings.Contains(policy, "script-src 'self'") {
		t.Errorf("la page doit définir sa propre politique, reçu %q", policy)
	}

	bundle := e.getPlain(server.APIPrefix + "/docs/assets/swagger-ui-bundle.js")
	if bundle.status != http.StatusOK {
		t.Errorf("bundle : statut %d", bundle.status)
	}
}

func TestServedContractIsTheOneTheTestsEnforce(t *testing.T) {
	e := newE2E(t, nil)

	response := e.getPlain(server.APIPrefix + apidocs.SpecPath)
	if response.status != http.StatusOK || !bytes.Equal(response.body, openapi.Spec) {
		t.Fatalf("statut %d : le contrat servi n'est pas openapi.yaml", response.status)
	}
	if _, err := openapi3.NewLoader().LoadFromData(response.body); err != nil {
		t.Fatalf("le contrat servi n'est pas un OpenAPI valide : %v", err)
	}
}

func TestDocsDoNotChangeThePolicyOfTheAPI(t *testing.T) {
	e := newE2E(t, nil)

	for _, path := range []string{"/health", "/auth/me", "/inconnue", apidocs.SpecPath, apidocs.InitializerPath} {
		response := e.getPlain(server.APIPrefix + path)
		if policy := response.header.Get("Content-Security-Policy"); policy != apiPolicy {
			t.Errorf("%s : politique %q, attendu celle de l'API %q", path, policy, apiPolicy)
		}
	}
}

func TestDocsAreNotPartOfTheContract(t *testing.T) {
	e := newE2E(t, nil)

	for _, route := range e.server.Routes() {
		if strings.HasPrefix(route.Path, "/docs") {
			t.Errorf("%s %s est dans les routes du contrat : elle serait exigée dans openapi.yaml", route.Method, route.Path)
		}
	}
}

func TestDocsRefuseWrites(t *testing.T) {
	e := newE2E(t, nil)

	response, err := http.Post(e.http.URL+server.APIPrefix+apidocs.PagePath, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST sur la page : statut %d, attendu 405", response.StatusCode)
	}
}
