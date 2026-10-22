// Commande loadtest : mesure les temps de réponse de l'API sous charge et échoue si le p95 dépasse le seuil.
//
//	go run ./cmd/loadtest                                  tous les scénarios publics, 20 s chacun
//	go run ./cmd/loadtest -scenarios list,heroes -p95 350ms
//	LOADTEST_PASSWORD=… go run ./cmd/loadtest -identifier jane.foster@heros-sos.local
//
// Le scénario « mes incidents » demande un compte citoyen (-identifier et LOADTEST_PASSWORD) : le mot de
// passe se lit dans l'environnement, jamais dans un argument. Une API à l'arrêt, ou une base vide de
// héros et de vilains (lancer `make seed`), donne une erreur claire. Le code de sortie est 1 si un seuil
// est dépassé.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/loadtest"
)

const passwordEnv = "LOADTEST_PASSWORD"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	baseURL := flag.String("base-url", "http://127.0.0.1:8080/api/v1", "URL de l'API, préfixe /api/v1 compris")
	workers := flag.Int("workers", 16, "nombre de clients en parallèle")
	duration := flag.Duration("duration", 20*time.Second, "durée de chaque scénario")
	timeout := flag.Duration("timeout", 60*time.Second, "délai maximal d'une requête")
	p95 := flag.Duration("p95", 350*time.Millisecond, "seuil du p95 par scénario")
	maxErrors := flag.Float64("max-errors", 0.01, "part maximale de requêtes en erreur (0 à 1)")
	only := flag.String("scenarios", "", "scénarios à jouer, séparés par des virgules (tous par défaut)")
	identifier := flag.String("identifier", "", "email du citoyen pour le scénario « mes incidents »")
	origin := flag.String("origin", "http://localhost:3000", "origine envoyée à la connexion (HEROS_ALLOWED_ORIGINS)")
	flag.Parse()

	// Un client anonyme pour les scénarios publics, un autre portant la session du citoyen : un citoyen
	// connecté ne voit que ses propres incidents, y compris sur la liste publique.
	newClient := func(withSession bool) (*http.Client, error) {
		client := &http.Client{Timeout: *timeout, Transport: &http.Transport{MaxIdleConnsPerHost: *workers}}
		if withSession {
			jar, err := cookiejar.New(nil)
			if err != nil {
				return nil, err
			}
			client.Jar = jar
		}
		return client, nil
	}
	anonymous, err := newClient(false)
	if err != nil {
		return err
	}
	base := strings.TrimRight(*baseURL, "/")

	targets, err := loadtest.Discover(ctx, anonymous, base)
	if err != nil {
		return fmt.Errorf("%w (l'API répond-elle sur %s, et `make seed` a-t-il été lancé ?)", err, base)
	}
	authenticated := anonymous
	if *identifier != "" {
		password := os.Getenv(passwordEnv)
		if password == "" {
			return fmt.Errorf("-identifier demande le mot de passe dans la variable %s", passwordEnv)
		}
		if authenticated, err = newClient(true); err != nil {
			return err
		}
		if targets.ReporterID, err = loadtest.Login(ctx, authenticated, base, *origin, *identifier, password); err != nil {
			return err
		}
	}

	scenarios, err := selectScenarios(loadtest.Scenarios(targets), *only)
	if err != nil {
		return err
	}
	cfg := loadtest.Config{BaseURL: base, Client: anonymous, AuthClient: authenticated, Workers: *workers, Duration: *duration}
	results := make([]loadtest.Result, 0, len(scenarios))
	for _, scenario := range scenarios {
		fmt.Fprintf(os.Stderr, "… %s (%d clients, %s)\n", scenario.Name, *workers, *duration)
		results = append(results, loadtest.Run(ctx, cfg, scenario))
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	fmt.Println()
	if !loadtest.Report(os.Stdout, results, *p95, *maxErrors) {
		return errors.New("au moins un scénario dépasse un seuil")
	}
	return nil
}

// selectScenarios garde ceux demandés, dans l'ordre de présentation. Un nom inconnu est une erreur : une
// faute de frappe ne doit pas faire croire qu'un scénario a passé.
func selectScenarios(all []loadtest.Scenario, only string) ([]loadtest.Scenario, error) {
	if only == "" {
		return all, nil
	}
	wanted := strings.Split(only, ",")
	var selected []loadtest.Scenario
	for _, scenario := range all {
		if slices.Contains(wanted, scenario.Name) {
			selected = append(selected, scenario)
		}
	}
	for _, name := range wanted {
		if !slices.ContainsFunc(all, func(s loadtest.Scenario) bool { return s.Name == name }) {
			return nil, fmt.Errorf("scénario inconnu ou inapplicable : %q (pour « mine », donnez -identifier)", name)
		}
	}
	return selected, nil
}
