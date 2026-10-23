package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func bufferLogger() (*slog.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer
	return slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})), &buffer
}

func prefixes(t *testing.T, values ...string) []netip.Prefix {
	t.Helper()
	result := make([]netip.Prefix, len(values))
	for i, value := range values {
		result[i] = netip.MustParsePrefix(value)
	}
	return result
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		forwarded []string
		trusted   []string
		want      string
	}{
		{"sans proxy", "203.0.113.7:4444", nil, nil, "203.0.113.7"},
		{"en-tête ignoré si le proxy n'est pas de confiance", "203.0.113.7:4444", []string{"198.51.100.1"}, nil, "203.0.113.7"},
		{"un proxy de confiance", "10.0.0.2:80", []string{"198.51.100.1"}, []string{"10.0.0.0/8"}, "198.51.100.1"},
		{"plusieurs sauts, le client est le plus à droite non fiable", "10.0.0.2:80",
			[]string{"198.51.100.1, 192.0.2.9, 10.0.0.5"}, []string{"10.0.0.0/8"}, "192.0.2.9"},
		{"adresse forgée à gauche par le client", "10.0.0.2:80",
			[]string{"1.1.1.1, 198.51.100.1"}, []string{"10.0.0.0/8"}, "198.51.100.1"},
		{"plusieurs en-têtes", "10.0.0.2:80", []string{"1.1.1.1", "198.51.100.1"}, []string{"10.0.0.0/8"}, "198.51.100.1"},
		{"en-tête mal formé", "10.0.0.2:80", []string{"pas-une-ip"}, []string{"10.0.0.0/8"}, "10.0.0.2"},
		{"proxy sans en-tête", "10.0.0.2:80", nil, []string{"10.0.0.0/8"}, "10.0.0.2"},
		{"IPv4 encapsulée dans IPv6", "[::ffff:203.0.113.7]:80", nil, nil, "203.0.113.7"},
		{"IPv6", "[2001:db8::1]:443", nil, nil, "2001:db8::1"},
		{"adresse sans port", "203.0.113.7", nil, nil, "203.0.113.7"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = test.remote
			for _, value := range test.forwarded {
				request.Header.Add("X-Forwarded-For", value)
			}
			if got := ClientIP(request, prefixes(t, test.trusted...)); got.String() != test.want {
				t.Errorf("ClientIP = %s, attendu %s", got, test.want)
			}
		})
	}
}

func TestWithClientIPStoresAddressInContext(t *testing.T) {
	var got netip.Addr
	handler := WithClientIP(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = ClientIPFrom(r.Context())
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.7:1"
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if got.String() != "203.0.113.7" {
		t.Errorf("adresse = %s", got)
	}
	if ClientIPFrom(context.Background()).IsValid() {
		t.Error("sans middleware, l'adresse doit être invalide")
	}
}

type payload struct {
	Name string `json:"name"`
}

func decode(t *testing.T, contentType, body string, maxBytes int64) (payload, error) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	var dst payload
	err := DecodeJSON(httptest.NewRecorder(), request, maxBytes, &dst)
	return dst, err
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int // 0 = succès
	}{
		{"valide", "application/json", `{"name":"Steve"}`, 0},
		{"avec charset", "application/json; charset=utf-8", `{"name":"Steve"}`, 0},
		{"mauvais type de contenu", "text/plain", `{"name":"Steve"}`, http.StatusUnsupportedMediaType},
		{"sans type de contenu", "", `{"name":"Steve"}`, http.StatusUnsupportedMediaType},
		{"champ inconnu", "application/json", `{"name":"Steve","admin":true}`, http.StatusBadRequest},
		{"JSON invalide", "application/json", `{"name":`, http.StatusBadRequest},
		{"mauvais type de valeur", "application/json", `{"name":42}`, http.StatusBadRequest},
		{"corps vide", "application/json", ``, http.StatusBadRequest},
		{"deux objets", "application/json", `{"name":"a"}{"name":"b"}`, http.StatusBadRequest},
		{"données après l'objet", "application/json", `{"name":"a"} x`, http.StatusBadRequest},
		{"trop volumineux", "application/json", `{"name":"` + strings.Repeat("a", 100) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decode(t, test.contentType, test.body, 64)
			if test.wantStatus == 0 {
				if err != nil || got.Name != "Steve" {
					t.Fatalf("(%+v, %v)", got, err)
				}
				return
			}
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Status != test.wantStatus {
				t.Fatalf("erreur = %v, attendu le statut %d", err, test.wantStatus)
			}
		})
	}
}

func TestDecodeJSONDoesNotEchoInput(t *testing.T) {
	_, err := decode(t, "application/json", `{"name": "secret-valeur-saisie" "oups"}`, 1024)
	if err == nil || strings.Contains(err.Error(), "secret-valeur-saisie") {
		t.Fatalf("le message ne doit pas citer la saisie : %v", err)
	}
}

func TestFailMapsErrors(t *testing.T) {
	logger, logs := bufferLogger()

	var validationErr error
	collector := validation.Collector{}
	collector.Add("email", validation.CodeInvalid)
	validationErr = collector.Err()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"erreur HTTP", Forbidden("non"), http.StatusForbidden, CodeForbidden},
		{"erreur HTTP enveloppée", errors.Join(errors.New("contexte"), NotFound("absent")), http.StatusNotFound, CodeNotFound},
		{"validation", validationErr, http.StatusUnprocessableEntity, CodeValidationFailed},
		{"erreur inconnue", errors.New("connexion à postgres://u:motdepasse@db refusée"), http.StatusInternalServerError, CodeInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			Fail(recorder, httptest.NewRequest(http.MethodGet, "/", nil), logger, test.err)

			if recorder.Code != test.wantStatus {
				t.Errorf("statut = %d, attendu %d", recorder.Code, test.wantStatus)
			}
			var body errorBody
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("corps non JSON : %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Errorf("code = %q, attendu %q", body.Error.Code, test.wantCode)
			}
			if strings.Contains(recorder.Body.String(), "motdepasse") {
				t.Error("un détail interne a fuité vers le client")
			}
		})
	}

	if !strings.Contains(logs.String(), "motdepasse") {
		t.Error("l'erreur interne doit être journalisée côté serveur")
	}
}

func TestFailExposesValidationFields(t *testing.T) {
	collector := validation.Collector{}
	collector.Add("password", validation.CodeWeak)
	recorder := httptest.NewRecorder()
	Fail(recorder, httptest.NewRequest(http.MethodGet, "/", nil), discardLogger(), collector.Err())

	var body errorBody
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	if body.Error.Fields["password"] != validation.CodeWeak {
		t.Errorf("champs = %v", body.Error.Fields)
	}
}

func TestFailIgnoresAbandonedRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()

	Fail(recorder, request, discardLogger(), context.Canceled)

	if recorder.Body.Len() != 0 {
		t.Errorf("rien ne doit être écrit pour un client parti, reçu %q", recorder.Body.String())
	}
}

func TestRateLimitedSetsRetryAfter(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, RateLimited(90*time.Second+400*time.Millisecond))

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf("statut = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "90" {
		t.Errorf("Retry-After = %q, attendu 90", got)
	}
}

func TestRequestID(t *testing.T) {
	var seen string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))

	t.Run("généré", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if seen == "" || recorder.Header().Get("X-Request-Id") != seen {
			t.Errorf("identifiant = %q, en-tête = %q", seen, recorder.Header().Get("X-Request-Id"))
		}
	})
	t.Run("fourni par le client", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("X-Request-Id", "client-req-12345")
		handler.ServeHTTP(httptest.NewRecorder(), request)
		if seen != "client-req-12345" {
			t.Errorf("identifiant = %q", seen)
		}
	})
	t.Run("identifiant dangereux remplacé", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("X-Request-Id", "faux\nlevel=ERROR msg=injection")
		handler.ServeHTTP(httptest.NewRecorder(), request)
		if strings.ContainsAny(seen, " \n=") || seen == "" {
			t.Errorf("identifiant dangereux conservé : %q", seen)
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, hsts := range []bool{false, true} {
		recorder := httptest.NewRecorder()
		SecurityHeaders(hsts)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		header := recorder.Header()
		for name, want := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Cache-Control":          "no-store",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := header.Get(name); got != want {
				t.Errorf("hsts=%v : %s = %q, attendu %q", hsts, name, got, want)
			}
		}
		if got := header.Get("Strict-Transport-Security") != ""; got != hsts {
			t.Errorf("HSTS présent = %v, attendu %v", got, hsts)
		}
	}
}

func TestRecoverTurnsPanicIntoJSON500(t *testing.T) {
	logger, logs := bufferLogger()
	handler := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boum interne")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("statut = %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "boum") {
		t.Error("le message de la panique ne doit pas atteindre le client")
	}
	if !strings.Contains(logs.String(), "boum interne") || !strings.Contains(logs.String(), "stack=") {
		t.Errorf("la panique et sa pile doivent être journalisées :\n%s", logs.String())
	}
}

func TestRecoverDoesNotWriteTwice(t *testing.T) {
	handler := Recover(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("après l'en-tête")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusAccepted || recorder.Body.Len() != 0 {
		t.Errorf("(%d, %q) : la réponse déjà commencée ne doit pas être écrasée", recorder.Code, recorder.Body.String())
	}
}

func TestRecoverRepanicsOnAbortHandler(t *testing.T) {
	handler := Recover(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if recovered := recover(); recovered != http.ErrAbortHandler { //nolint:errorlint // sentinelle
			t.Errorf("ErrAbortHandler doit être relancée, reçu %v", recovered)
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestAccessLogOmitsQueryString(t *testing.T) {
	logger, logs := bufferLogger()
	handler := AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, NotFound("absent"))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/activation?token=SECRET123", nil))

	output := logs.String()
	if strings.Contains(output, "SECRET123") || strings.Contains(output, "token=") {
		t.Errorf("la query string ne doit jamais être journalisée :\n%s", output)
	}
	for _, want := range []string{"path=/api/v1/activation", "status=404", "level=WARN"} {
		if !strings.Contains(output, want) {
			t.Errorf("le log ne contient pas %q :\n%s", want, output)
		}
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	tag := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" avant")
				next.ServeHTTP(w, r)
				order = append(order, name+" après")
			})
		}
	}
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), tag("A"), tag("B"))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := "A avant, B avant, handler, B après, A après"
	if got := strings.Join(order, ", "); got != want {
		t.Errorf("ordre = %q, attendu %q", got, want)
	}
}

func TestAccessLogRecordsTheMatchedRouteInsteadOfTheRawPath(t *testing.T) {
	logger, logs := bufferLogger()
	handler := AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRoute(r.Context(), "GET /api/v1/incidents/{id}")
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/incidents/0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10", nil))

	// Les durées se regroupent par motif : avec l'identifiant dans la route, chaque incident aurait la sienne.
	if output := logs.String(); !strings.Contains(output, `route="GET /api/v1/incidents/{id}"`) {
		t.Errorf("le motif de route doit être journalisé :\n%s", output)
	}
}

// Une sonde de santé réussie ne se journalise qu'en DEBUG : Docker et le proxy l'interrogent toutes les 10 secondes.
func TestAccessLogKeepsSuccessfulQuietRoutesAtDebug(t *testing.T) {
	logger, logs := bufferLogger()
	handler := AccessLog(logger, "GET /api/v1/health")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRoute(r.Context(), r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil))

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("deux requêtes, %d lignes de log :\n%s", len(lines), logs.String())
	}
	if !strings.Contains(lines[0], "level=DEBUG") || !strings.Contains(lines[0], "path=/api/v1/health") {
		t.Errorf("la sonde réussie doit être en DEBUG :\n%s", lines[0])
	}
	if !strings.Contains(lines[1], "level=INFO") {
		t.Errorf("une requête ordinaire reste en INFO :\n%s", lines[1])
	}
}

// Une sonde qui échoue est le signal qu'on cherche : elle ne doit jamais être rendue silencieuse.
func TestAccessLogStillReportsAFailingQuietRoute(t *testing.T) {
	logger, logs := bufferLogger()
	handler := AccessLog(logger, "GET /api/v1/ready")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRoute(r.Context(), "GET /api/v1/ready")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))

	if output := logs.String(); !strings.Contains(output, "level=ERROR") || !strings.Contains(output, "status=503") {
		t.Errorf("une sonde en échec doit être journalisée en ERROR :\n%s", output)
	}
}

func TestAccessLogMarksRequestsNoRouteMatched(t *testing.T) {
	logger, logs := bufferLogger()
	handler := AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, NotFound("absent"))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/inconnu", nil))

	if output := logs.String(); !strings.Contains(output, "route=unmatched") {
		t.Errorf("une requête sans route doit être marquée unmatched :\n%s", output)
	}
}

func TestSetRouteOutsideAccessLogDoesNothing(t *testing.T) {
	SetRoute(context.Background(), "GET /x") // ne doit ni paniquer ni rien noter
}
