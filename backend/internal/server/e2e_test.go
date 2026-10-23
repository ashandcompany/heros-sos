package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/bootstrap"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/server"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

const (
	frontOrigin   = "http://localhost:3000"
	goodPassword  = "Shield1234abc"
	otherPassword = "Hammer5678xyz"
	apiBase       = "/api/v1"
)

var (
	startTime  = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	tokenInURL = regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`)
)

// e2e est une application complète (vrai routeur, vraie base PostgreSQL isolée) servie par un
// serveur HTTP de test.
type e2e struct {
	t      *testing.T
	db     *gorm.DB
	cfg    config.Config
	server *server.Server
	http   *httptest.Server
	clock  *clock.Fake
	mailer *mail.Recorder
	users  user.Repository
	spec   *contract
}

func newE2E(t *testing.T, overrides map[string]string) *e2e {
	t.Helper()

	env := map[string]string{
		// Limites larges : chaque test qui vérifie une limite la resserre explicitement.
		"HEROS_RATE_REGISTER_PER_IP":          "10000/1h",
		"HEROS_RATE_LOGIN_PER_IP":             "10000/1h",
		"HEROS_RATE_LOGIN_PER_IDENTIFIER":     "10000/1h",
		"HEROS_RATE_PASSWORD_RESET_PER_IP":    "10000/1h",
		"HEROS_RATE_PASSWORD_RESET_PER_EMAIL": "10000/1h",
		"HEROS_RATE_CONTACT_PER_IP":           "10000/1h",
		"HEROS_RATE_NEWSLETTER_PER_IP":        "10000/1h",
		"HEROS_RATE_NEWSLETTER_PER_EMAIL":     "10000/1h",
		"HEROS_RATE_EMAIL_CHANGE_PER_USER":    "10000/1h",
		"HEROS_RATE_AVATAR_PER_USER":          "10000/1h",
		"HEROS_RATE_PASSWORD_CHECK_PER_USER":  "10000/1h",
		"HEROS_RATE_HERO_LOCATION_PER_USER":   "10000/1h",
		// Pas de cache sur les statistiques du tableau de bord : les tests veulent voir leurs écritures.
		"HEROS_DASHBOARD_STATS_TTL": "0",
	}
	for name, value := range overrides {
		env[name] = value
	}
	cfg := testutil.Config(t, env)
	db := testutil.NewDB(t)
	clk := clock.NewFake(startTime)
	recorder := mail.NewRecorder()

	// HEROS_TEST_LOG=1 affiche les journaux de l'application : utile pour comprendre une erreur 500.
	logger := slog.New(slog.DiscardHandler)
	if os.Getenv("HEROS_TEST_LOG") != "" {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	app, err := server.NewWithDB(cfg, logger, db,
		server.WithClock(clk), server.WithMailer(recorder))
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(app.Handler())
	t.Cleanup(httpServer.Close)

	sealer, err := crypto.NewSealer(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		t.Fatal(err)
	}
	return &e2e{
		t: t, db: db, cfg: cfg, server: app, http: httpServer, clock: clk, mailer: recorder,
		users: user.NewRepository(db, sealer), spec: loadContract(t),
	}
}

// actor est un navigateur : son propre jeu de cookies et, une fois connecté, son jeton CSRF.
type actor struct {
	e      *e2e
	client *http.Client
	csrf   string
	email  string
}

func (e *e2e) newActor() *actor {
	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	return &actor{e: e, client: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

// code retourne le code d'erreur d'une réponse d'erreur de l'API.
func (r response) code() string {
	var payload struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(r.body, &payload)
	return payload.Error.Code
}

func (r response) fields() map[string]string {
	var payload struct {
		Error struct{ Fields map[string]string } `json:"error"`
	}
	_ = json.Unmarshal(r.body, &payload)
	return payload.Error.Fields
}

func (r response) decode(t *testing.T, target any) {
	t.Helper()
	if err := json.Unmarshal(r.body, target); err != nil {
		t.Fatalf("réponse non JSON (%d) : %s", r.status, r.body)
	}
}

// forbidden liste ce qui ne doit jamais apparaître dans une réponse de l'API.
var forbidden = []string{"$argon2id", "passwordHash", "password_hash", "token_hash"}

// request envoie une requête comme le ferait un navigateur : l'en-tête Origin accompagne les
// écritures, et le jeton CSRF est joint une fois connecté.
func (a *actor) request(method, path string, body any, mutate ...func(*http.Request)) response {
	a.e.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			a.e.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, a.e.http.URL+apiBase+path, reader)
	if err != nil {
		a.e.t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		request.Header.Set("Origin", frontOrigin)
		if a.csrf != "" {
			request.Header.Set(auth.CSRFHeader, a.csrf)
		}
	}
	for _, change := range mutate {
		change(request)
	}

	httpResponse, err := a.client.Do(request)
	if err != nil {
		a.e.t.Fatal(err)
	}
	defer func() { _ = httpResponse.Body.Close() }()
	payload, _ := io.ReadAll(httpResponse.Body)

	for _, secret := range forbidden {
		if strings.Contains(string(payload), secret) {
			a.e.t.Fatalf("%s %s : la réponse contient %q :\n%s", method, path, secret, payload)
		}
	}
	// Chaque échange des tests de bout en bout est vérifié contre le contrat OpenAPI (sauf dans
	// les rares tests qui visent volontairement des routes qu'il ne décrit pas).
	if a.e.spec != nil {
		a.e.spec.verify(a.e.t, request, encodedBody(body), httpResponse.StatusCode, httpResponse.Header, payload)
	}
	return response{status: httpResponse.StatusCode, header: httpResponse.Header, body: payload}
}

// encodedBody ré-encode le corps d'une requête pour la validation du contrat.
func encodedBody(body any) []byte {
	if body == nil {
		return nil
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil
	}
	return encoded
}

func (a *actor) get(path string) response { return a.request(http.MethodGet, path, nil) }
func (a *actor) post(path string, body any) response {
	return a.request(http.MethodPost, path, body)
}

func registrationBody(email string) map[string]any {
	return map[string]any{
		"civility": "monsieur", "firstName": "Steve", "lastName": "Rogers", "birthDate": "1990-07-04",
		"email": email, "mobilePhone": "06 12 34 56 78", "password": goodPassword,
		"address": map[string]any{"street": "569 Leaman Place", "postalCode": "11201", "city": "Brooklyn", "country": "US"},
	}
}

// lastToken extrait le jeton du lien du dernier mail envoyé.
func (e *e2e) lastToken() string {
	e.t.Helper()
	message, ok := e.mailer.Last()
	if !ok {
		e.t.Fatal("aucun mail envoyé")
	}
	match := tokenInURL.FindStringSubmatch(message.Text)
	if match == nil {
		e.t.Fatalf("aucun jeton dans le mail :\n%s", message.Text)
	}
	return match[1]
}

// signUp inscrit et active un compte citoyen, sans le connecter.
func (e *e2e) signUp(email string) *actor {
	e.t.Helper()
	a := e.newActor()
	a.email = email
	if got := a.post("/auth/register", registrationBody(email)); got.status != http.StatusCreated {
		e.t.Fatalf("inscription : %d %s", got.status, got.body)
	}
	if got := a.post("/auth/activate", map[string]string{"token": e.lastToken()}); got.status != http.StatusNoContent {
		e.t.Fatalf("activation : %d %s", got.status, got.body)
	}
	return a
}

// login connecte l'acteur et mémorise son jeton CSRF.
func (a *actor) login(identifier, password string) response {
	a.e.t.Helper()
	got := a.post("/auth/login", map[string]any{"identifier": identifier, "password": password})
	if got.status == http.StatusOK {
		var body struct{ CSRFToken string }
		got.decode(a.e.t, &body)
		a.csrf = body.CSRFToken
	}
	return got
}

// signUpAndLogin crée un citoyen actif et connecté.
func (e *e2e) signUpAndLogin(email string) *actor {
	e.t.Helper()
	a := e.signUp(email)
	if got := a.login(email, goodPassword); got.status != http.StatusOK {
		e.t.Fatalf("connexion : %d %s", got.status, got.body)
	}
	return a
}

// newModerator crée un modérateur actif (comme le fait la commande createmoderator) et le
// connecte.
func (e *e2e) newModerator(email string) *actor {
	e.t.Helper()
	_, err := bootstrap.CreateModerator(context.Background(), bootstrap.ModeratorDeps{
		Users: e.users, Hasher: auth.NewPasswordHasher(e.cfg.Security.Argon2), Clock: e.clock,
		Password: e.cfg.Security.Password,
	}, bootstrap.ModeratorInput{
		Email: email, FirstName: "Nick", LastName: "Fury", BirthDate: "1950-12-04", Password: goodPassword,
	})
	if err != nil {
		e.t.Fatal(err)
	}
	a := e.newActor()
	a.email = email
	if got := a.login(email, goodPassword); got.status != http.StatusOK {
		e.t.Fatalf("connexion du modérateur : %d %s", got.status, got.body)
	}
	return a
}

func (e *e2e) count(table, where string, args ...any) int64 {
	e.t.Helper()
	var total int64
	if err := e.db.Table(table).Where(where, args...).Count(&total).Error; err != nil {
		e.t.Fatal(err)
	}
	return total
}

func (a *actor) userID() string {
	a.e.t.Helper()
	var body struct{ User struct{ ID string } }
	got := a.get("/auth/me")
	if got.status != http.StatusOK {
		a.e.t.Fatalf("/auth/me : %d %s", got.status, got.body)
	}
	got.decode(a.e.t, &body)
	return body.User.ID
}
