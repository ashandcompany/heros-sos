package server_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

const specPath = "../../openapi/openapi.yaml"

// contract est le contrat OpenAPI chargé une seule fois pour tous les tests du paquet.
type contract struct {
	doc    *openapi3.T
	router routers.Router
}

var (
	loadOnce    sync.Once
	loadedSpec  *contract
	loadSpecErr error
)

func loadContract(t testing.TB) *contract {
	t.Helper()
	loadOnce.Do(func() {
		doc, err := openapi3.NewLoader().LoadFromFile(specPath)
		if err != nil {
			loadSpecErr = err
			return
		}
		if err := doc.Validate(context.Background()); err != nil {
			loadSpecErr = err
			return
		}
		router, err := legacy.NewRouter(doc)
		if err != nil {
			loadSpecErr = err
			return
		}
		loadedSpec = &contract{doc: doc, router: router}
	})
	if loadSpecErr != nil {
		t.Fatalf("le contrat OpenAPI est invalide : %v", loadSpecErr)
	}
	return loadedSpec
}

// verify vérifie qu'un échange HTTP respecte le contrat : la route et le statut sont documentés,
// la réponse a la forme décrite, et — quand la requête a été acceptée — elle respecte aussi le
// schéma de la requête.
func (c *contract) verify(t testing.TB, request *http.Request, requestBody []byte, status int, header http.Header, body []byte) {
	t.Helper()

	// Le contrat décrit un serveur relatif (/api/v1) : on lui présente le chemin seul, sans
	// l'hôte du serveur de test.
	request = request.Clone(context.Background())
	request.URL = &url.URL{Path: request.URL.Path, RawQuery: request.URL.RawQuery}
	request.Host = ""
	route, pathParams, err := c.router.FindRoute(request)
	if err != nil {
		t.Errorf("%s %s n'est pas documentée dans openapi.yaml : %v", request.Method, request.URL.Path, err)
		return
	}
	options := &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, IncludeResponseStatus: true}
	input := &openapi3filter.RequestValidationInput{Request: request, PathParams: pathParams, Route: route, Options: options}

	if status < http.StatusMultipleChoices && len(requestBody) > 0 {
		clone := request.Clone(context.Background())
		clone.Body = io.NopCloser(bytes.NewReader(requestBody))
		accepted := &openapi3filter.RequestValidationInput{Request: clone, PathParams: pathParams, Route: route, Options: options}
		if err := openapi3filter.ValidateRequest(context.Background(), accepted); err != nil {
			t.Errorf("%s %s : la requête acceptée ne respecte pas le contrat : %v", request.Method, request.URL.Path, err)
		}
	}

	err = openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: input, Status: status, Header: header,
		Body: io.NopCloser(bytes.NewReader(body)), Options: options,
	})
	if err != nil {
		t.Errorf("%s %s → %d : la réponse ne respecte pas le contrat : %v\ncorps : %s", request.Method, request.URL.Path, status, err, body)
	}
}

func TestOpenAPIDocumentIsValid(t *testing.T) {
	spec := loadContract(t)
	if spec.doc.Info.Title == "" || len(spec.doc.Paths.Map()) == 0 {
		t.Fatal("le contrat est vide")
	}
}

// TestRegisteredRoutesMatchTheContract garantit que l'API et sa documentation ne divergent jamais :
// chaque route enregistrée est décrite, et chaque opération décrite existe.
func TestRegisteredRoutesMatchTheContract(t *testing.T) {
	spec := loadContract(t)
	e := newE2E(t, nil)

	registered := make(map[string]bool)
	for _, route := range e.server.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	documented := make(map[string]bool)
	for path, item := range spec.doc.Paths.Map() {
		for method := range item.Operations() {
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	missingFromContract, missingFromCode := difference(registered, documented), difference(documented, registered)
	if len(missingFromContract) > 0 {
		t.Errorf("routes enregistrées mais absentes de openapi.yaml : %v", missingFromContract)
	}
	if len(missingFromCode) > 0 {
		t.Errorf("opérations décrites dans openapi.yaml mais non enregistrées : %v", missingFromCode)
	}
	if len(registered) == 0 {
		t.Fatal("aucune route enregistrée")
	}
}

func TestEveryOperationDeclaresAnIdentifierAndTag(t *testing.T) {
	spec := loadContract(t)
	seen := make(map[string]string)
	for path, item := range spec.doc.Paths.Map() {
		for method, operation := range item.Operations() {
			where := strings.ToUpper(method) + " " + path
			if operation.OperationID == "" || len(operation.Tags) == 0 || operation.Summary == "" {
				t.Errorf("%s : operationId, tags et summary sont obligatoires", where)
			}
			if previous, duplicate := seen[operation.OperationID]; duplicate {
				t.Errorf("operationId %q en double : %s et %s", operation.OperationID, previous, where)
			}
			seen[operation.OperationID] = where
		}
	}
}

func difference(a, b map[string]bool) []string {
	var only []string
	for key := range a {
		if !b[key] {
			only = append(only, key)
		}
	}
	sort.Strings(only)
	return only
}
