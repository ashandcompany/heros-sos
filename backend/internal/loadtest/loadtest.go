// Package loadtest mesure les temps de réponse de l'API sous charge : des clients en parallèle rejouent
// des requêtes GET pendant une durée donnée, puis le rapport donne p50, p95, p99 et le taux d'erreurs.
//
// Chaque client attend la réponse avant d'envoyer la suivante (charge « fermée »). Sous une surcharge
// forte, cela sous-estime un peu les pires latences : la mesure convient pour comparer un avant et un
// après, pas pour dimensionner une infrastructure.
package loadtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Scenario est une requête GET rejouée en boucle.
type Scenario struct {
	Name string
	// Path retourne le chemin et la query de la n-ième requête, relatifs à l'URL de base.
	Path func(n int64) string
	// Authenticated joue le scénario avec la session ouverte par Login. Les autres sont anonymes : l'API
	// ne montre à un citoyen connecté que ses propres incidents, même sur la liste publique.
	Authenticated bool
}

// Config règle l'exécution d'un scénario.
type Config struct {
	// BaseURL est l'URL de l'API sans barre finale, par exemple http://127.0.0.1:8080/api/v1.
	BaseURL string
	// Client est anonyme (sans cookie) ; AuthClient porte la session des scénarios Authenticated.
	Client, AuthClient *http.Client
	// Workers est le nombre de clients en parallèle.
	Workers int
	// Duration est la durée pendant laquelle de nouvelles requêtes partent. Celles en cours se terminent.
	Duration time.Duration
}

// Result résume un scénario.
type Result struct {
	Scenario           string
	Requests, Errors   int
	Elapsed            time.Duration
	P50, P95, P99, Max time.Duration
}

// RequestsPerSecond est le débit mesuré.
func (r Result) RequestsPerSecond() float64 {
	if r.Elapsed <= 0 {
		return 0
	}
	return float64(r.Requests) / r.Elapsed.Seconds()
}

// ErrorRate est la part des requêtes en erreur (transport ou statut hors 2xx), entre 0 et 1.
func (r Result) ErrorRate() float64 {
	if r.Requests == 0 {
		return 0
	}
	return float64(r.Errors) / float64(r.Requests)
}

// Run rejoue le scénario avec cfg.Workers clients pendant cfg.Duration.
func Run(ctx context.Context, cfg Config, scenario Scenario) Result {
	var (
		counter  atomic.Int64
		wg       sync.WaitGroup
		mu       sync.Mutex
		all      []time.Duration
		failures int
	)
	client := cfg.Client
	if scenario.Authenticated {
		client = cfg.AuthClient
	}
	began := time.Now()
	deadline := began.Add(cfg.Duration)
	for range cfg.Workers {
		wg.Go(func() {
			var latencies []time.Duration
			var failed int
			for time.Now().Before(deadline) && ctx.Err() == nil {
				elapsed, err := get(ctx, client, cfg.BaseURL+scenario.Path(counter.Add(1)))
				latencies = append(latencies, elapsed)
				if err != nil {
					failed++
				}
			}
			mu.Lock()
			all, failures = append(all, latencies...), failures+failed
			mu.Unlock()
		})
	}
	wg.Wait()

	slices.Sort(all)
	return Result{
		Scenario: scenario.Name, Requests: len(all), Errors: failures, Elapsed: time.Since(began),
		P50: percentile(all, 0.50), P95: percentile(all, 0.95), P99: percentile(all, 0.99), Max: percentile(all, 1),
	}
}

// get envoie la requête et lit toute la réponse : le transfert fait partie du temps mesuré.
func get(ctx context.Context, client *http.Client, target string) (time.Duration, error) {
	start := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return time.Since(start), err
	}
	response, err := client.Do(request)
	if err != nil {
		return time.Since(start), err
	}
	defer func() { _ = response.Body.Close() }()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return time.Since(start), err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return time.Since(start), fmt.Errorf("statut %d", response.StatusCode)
	}
	return time.Since(start), nil
}

// percentile retourne la valeur au rang p (méthode du rang le plus proche) d'une série triée.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(float64(len(sorted))*p+0.999999) - 1
	return sorted[min(max(rank, 0), len(sorted)-1)]
}

// Targets désigne les ressources que les scénarios interrogent.
type Targets struct {
	HeroID, VillainID string
	// ReporterID est le compte connecté : sans lui, le scénario « mes incidents » est ignoré.
	ReporterID string
}

// Scenarios retourne les scénarios applicables aux cibles données, dans l'ordre de présentation.
func Scenarios(targets Targets) []Scenario {
	fixed := func(name, path string) Scenario {
		return Scenario{Name: name, Path: func(int64) string { return path }}
	}
	authenticated := func(name, path string) Scenario {
		scenario := fixed(name, path)
		scenario.Authenticated = true
		return scenario
	}
	scenarios := []Scenario{
		fixed("list", "/incidents?limit=20"),
		fixed("list-open", "/incidents?limit=20&status=reported&status=in_progress"),
	}
	if targets.ReporterID != "" {
		scenarios = append(scenarios, authenticated("mine", "/incidents?limit=20&reporterId="+url.QueryEscape(targets.ReporterID)))
	}
	if targets.HeroID != "" {
		scenarios = append(scenarios,
			fixed("hero-incidents", "/heroes/"+targets.HeroID+"/incidents?limit=20"),
			fixed("hero", "/heroes/"+targets.HeroID))
	}
	if targets.VillainID != "" {
		scenarios = append(scenarios,
			fixed("villain-incidents", "/villains/"+targets.VillainID+"/incidents?limit=20"),
			fixed("villain", "/villains/"+targets.VillainID))
	}
	return append(scenarios, fixed("heroes", "/heroes?limit=20"), fixed("villains", "/villains?limit=20"))
}

// Report écrit le tableau des résultats et retourne vrai si tous les seuils sont respectés : p95 sous
// p95Limit et taux d'erreurs sous maxErrorRate pour chaque scénario.
func Report(w io.Writer, results []Result, p95Limit time.Duration, maxErrorRate float64) bool {
	ok := true
	_, _ = fmt.Fprintf(w, "%-18s %9s %8s %8s %9s %9s %9s %9s\n", "scénario", "requêtes", "erreurs", "req/s", "p50", "p95", "p99", "max")
	for _, r := range results {
		verdict := "ok"
		if r.P95 > p95Limit || r.ErrorRate() > maxErrorRate || r.Requests == 0 {
			verdict, ok = "ÉCHEC", false
		}
		_, _ = fmt.Fprintf(w, "%-18s %9d %8d %8.1f %9s %9s %9s %9s  %s\n", r.Scenario, r.Requests, r.Errors,
			r.RequestsPerSecond(), round(r.P50), round(r.P95), round(r.P99), round(r.Max), verdict)
	}
	_, _ = fmt.Fprintf(w, "\nseuil : p95 ≤ %s, erreurs ≤ %.0f %%\n", p95Limit, maxErrorRate*100)
	return ok
}

func round(d time.Duration) time.Duration {
	if d >= time.Second {
		return d.Round(10 * time.Millisecond)
	}
	return d.Round(100 * time.Microsecond)
}

// page est l'enveloppe commune des listes de l'API.
type page struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
}

// Discover cherche un héros et un vilain publics à interroger.
func Discover(ctx context.Context, client *http.Client, baseURL string) (Targets, error) {
	var targets Targets
	var err error
	if targets.HeroID, err = firstID(ctx, client, baseURL+"/heroes?limit=1"); err != nil {
		return Targets{}, fmt.Errorf("loadtest : aucun héros public : %w", err)
	}
	if targets.VillainID, err = firstID(ctx, client, baseURL+"/villains?limit=1"); err != nil {
		return Targets{}, fmt.Errorf("loadtest : aucun vilain : %w", err)
	}
	return targets, nil
}

func firstID(ctx context.Context, client *http.Client, target string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("statut %d", response.StatusCode)
	}
	var body page
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", err
	}
	if len(body.Items) == 0 {
		return "", errors.New("liste vide")
	}
	return body.Items[0].ID, nil
}

// Login ouvre une session : le cookie est gardé par le jar du client. origin est envoyé comme le ferait
// un navigateur (l'API refuse les écritures d'une origine inconnue). Retourne l'identifiant du compte.
func Login(ctx context.Context, client *http.Client, baseURL, origin, identifier, password string) (string, error) {
	payload, err := json.Marshal(map[string]any{"identifier": identifier, "password": password, "rememberMe": false})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/login", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("loadtest : connexion : %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		// Ni l'identifiant ni le mot de passe n'apparaissent dans l'erreur.
		return "", fmt.Errorf("loadtest : connexion refusée (statut %d)", response.StatusCode)
	}
	var body struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.User.ID == "" {
		return "", errors.New("loadtest : réponse de connexion illisible")
	}
	return body.User.ID, nil
}
