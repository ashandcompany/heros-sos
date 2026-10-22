package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPercentileUsesTheNearestRank(t *testing.T) {
	var series []time.Duration
	for i := 1; i <= 100; i++ {
		series = append(series, time.Duration(i)*time.Millisecond)
	}
	for _, tc := range []struct {
		p    float64
		want time.Duration
	}{{0.50, 50 * time.Millisecond}, {0.95, 95 * time.Millisecond}, {0.99, 99 * time.Millisecond}, {1, 100 * time.Millisecond}} {
		if got := percentile(series, tc.p); got != tc.want {
			t.Errorf("p%.0f = %v, attendu %v", tc.p*100, got, tc.want)
		}
	}
	if got := percentile(nil, 0.95); got != 0 {
		t.Errorf("série vide : %v", got)
	}
}

func TestRunMeasuresLatencyAndCountsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		time.Sleep(15 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	cfg := Config{BaseURL: server.URL, Client: server.Client(), Workers: 4, Duration: 300 * time.Millisecond}

	good := Run(context.Background(), cfg, Scenario{Name: "ok", Path: func(int64) string { return "/ok" }})
	if good.Requests < 20 || good.Errors != 0 {
		t.Errorf("requêtes = %d, erreurs = %d", good.Requests, good.Errors)
	}
	if good.P50 < 15*time.Millisecond || good.P50 > good.P95 || good.P95 > good.P99 || good.P99 > good.Max {
		t.Errorf("percentiles incohérents : %+v", good)
	}

	bad := Run(context.Background(), cfg, Scenario{Name: "bad", Path: func(int64) string { return "/bad" }})
	if bad.Requests == 0 || bad.Errors != bad.Requests || bad.ErrorRate() != 1 {
		t.Errorf("toutes les requêtes devaient échouer : %+v", bad)
	}
}

func TestReportFailsWhenAThresholdIsExceeded(t *testing.T) {
	fast := Result{Scenario: "list", Requests: 100, Elapsed: time.Second, P50: 5 * time.Millisecond, P95: 20 * time.Millisecond, P99: 30 * time.Millisecond, Max: 40 * time.Millisecond}
	slow := Result{Scenario: "mine", Requests: 100, Elapsed: time.Second, P95: 900 * time.Millisecond}
	flaky := Result{Scenario: "hero", Requests: 100, Errors: 5, Elapsed: time.Second, P95: 10 * time.Millisecond}
	silent := Result{Scenario: "villain", Elapsed: time.Second}

	var out bytes.Buffer
	if !Report(&out, []Result{fast}, 350*time.Millisecond, 0.01) {
		t.Errorf("un scénario sous les seuils doit passer :\n%s", out.String())
	}
	for _, failing := range []Result{slow, flaky, silent} {
		out.Reset()
		if Report(&out, []Result{fast, failing}, 350*time.Millisecond, 0.01) {
			t.Errorf("%s dépasse un seuil mais le rapport passe :\n%s", failing.Scenario, out.String())
		}
		if !strings.Contains(out.String(), "ÉCHEC") {
			t.Errorf("%s : l'échec doit apparaître dans le tableau :\n%s", failing.Scenario, out.String())
		}
	}
}

func TestScenariosSkipWhatTheTargetsCannotServe(t *testing.T) {
	names := func(targets Targets) []string {
		var got []string
		for _, s := range Scenarios(targets) {
			got = append(got, s.Name)
		}
		return got
	}
	if got := strings.Join(names(Targets{}), ","); got != "list,list-open,latest-resolved,heroes,villains" {
		t.Errorf("sans cible : %s", got)
	}
	all := names(Targets{HeroID: "h", VillainID: "v", ReporterID: "r"})
	for _, want := range []string{"mine", "hero-incidents", "hero", "villain-incidents", "villain"} {
		found := false
		for _, name := range all {
			found = found || name == want
		}
		if !found {
			t.Errorf("scénario %q absent : %v", want, all)
		}
	}
	mine := Scenarios(Targets{ReporterID: "a b"})[3]
	if got := mine.Path(1); got != "/incidents?limit=20&reporterId=a+b" {
		t.Errorf("chemin de « mine » = %s : l'identifiant doit être échappé", got)
	}
	for _, scenario := range Scenarios(Targets{HeroID: "h", VillainID: "v", ReporterID: "r"}) {
		if scenario.Authenticated != (scenario.Name == "mine") {
			t.Errorf("%s : seul « mine » utilise la session (Authenticated = %v)", scenario.Name, scenario.Authenticated)
		}
	}
}

func TestTheDashboardScenarioNeedsAHeroSession(t *testing.T) {
	for _, scenario := range Scenarios(Targets{HeroID: "h", VillainID: "v"}) {
		if scenario.Name == "dashboard" {
			t.Fatal("« dashboard » ne doit être joué que sur demande")
		}
	}
	var found bool
	for _, scenario := range Scenarios(Targets{ReporterID: "r", Dashboard: true}) {
		if scenario.Name == "dashboard" {
			found = true
			if !scenario.Authenticated || scenario.Path(1) != "/hero/dashboard" {
				t.Errorf("« dashboard » : Authenticated = %v, chemin %s", scenario.Authenticated, scenario.Path(1))
			}
		}
	}
	if !found {
		t.Error("« dashboard » absent alors qu'il est demandé")
	}
}

func TestRunUsesTheSessionClientOnlyForAuthenticatedScenarios(t *testing.T) {
	var anonymous, withSession atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("session"); err == nil {
			withSession.Add(1)
		} else {
			anonymous.Add(1)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	serverURL, _ := url.Parse(server.URL)
	jar.SetCookies(serverURL, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
	cfg := Config{
		BaseURL: server.URL, Client: &http.Client{}, AuthClient: &http.Client{Jar: jar},
		Workers: 2, Duration: 100 * time.Millisecond,
	}

	Run(context.Background(), cfg, Scenario{Name: "public", Path: func(int64) string { return "/x" }})
	if withSession.Load() != 0 || anonymous.Load() == 0 {
		t.Fatalf("un scénario public ne doit pas porter de session : %d anonymes, %d avec session", anonymous.Load(), withSession.Load())
	}
	anonymous.Store(0)
	Run(context.Background(), cfg, Scenario{Name: "mine", Authenticated: true, Path: func(int64) string { return "/x" }})
	if anonymous.Load() != 0 || withSession.Load() == 0 {
		t.Errorf("un scénario authentifié doit porter la session : %d anonymes, %d avec session", anonymous.Load(), withSession.Load())
	}
}

func TestDiscoverAndLoginUseTheAPI(t *testing.T) {
	const password = "s3cret-password"
	mux := http.NewServeMux()
	mux.HandleFunc("GET /heroes", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"hero-1"}]}`))
	})
	mux.HandleFunc("GET /villains", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"villain-1"}]}`))
	})
	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Identifier, Password string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Header.Get("Origin") != "http://localhost:3000" || body.Password != password {
			http.Error(w, "refusé", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		_, _ = w.Write([]byte(`{"user":{"id":"citizen-1"}}`))
	})
	mux.HandleFunc("GET /whoami", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "abc" {
			http.Error(w, "anonyme", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	ctx := context.Background()

	targets, err := Discover(ctx, client, server.URL)
	if err != nil || targets.HeroID != "hero-1" || targets.VillainID != "villain-1" {
		t.Fatalf("Discover = %+v, %v", targets, err)
	}
	id, err := Login(ctx, client, server.URL, "http://localhost:3000", "jane@example.test", password)
	if err != nil || id != "citizen-1" {
		t.Fatalf("Login = %q, %v", id, err)
	}
	if elapsed, err := get(ctx, client, server.URL+"/whoami"); err != nil || elapsed <= 0 {
		t.Errorf("le cookie de session n'est pas renvoyé : %v", err)
	}

	_, err = Login(ctx, client, server.URL, "http://localhost:3000", "jane@example.test", "mauvais")
	if err == nil || strings.Contains(err.Error(), "mauvais") || strings.Contains(err.Error(), "jane@") {
		t.Errorf("la connexion refusée doit échouer sans divulguer les identifiants : %v", err)
	}
}
