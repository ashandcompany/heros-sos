package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/apidocs"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

// Ce test est interne au paquet : un serveur de production exige une configuration complète (TLS,
// SMTP…) et une base, alors qu'ici seul l'enregistrement des routes compte.
func TestDocsExistOnlyOutsideProduction(t *testing.T) {
	for environment, wantStatus := range map[config.Environment]int{
		config.EnvironmentDevelopment: http.StatusOK,
		config.EnvironmentTest:        http.StatusOK,
		config.EnvironmentProduction:  http.StatusNotFound,
	} {
		router := newRouter()
		registerDocsRoutes(router, &components{
			cfg:    config.Config{Environment: environment},
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})

		for _, path := range []string{apidocs.PagePath, apidocs.SpecPath, apidocs.InitializerPath, "/docs/assets/swagger-ui.css"} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, APIPrefix+path, nil))
			if recorder.Code != wantStatus {
				t.Errorf("%s %s : statut %d, attendu %d", environment, path, recorder.Code, wantStatus)
			}
		}
	}
}
