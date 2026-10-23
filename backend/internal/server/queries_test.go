package server_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// queryCounter compte les requêtes SQL que GORM envoie : branché sur la base de test, il mesure ce qu'une
// requête HTTP coûte vraiment en allers-retours.
type queryCounter struct {
	mu      sync.Mutex
	queries int
}

func (c *queryCounter) LogMode(gormlogger.LogLevel) gormlogger.Interface { return c }
func (c *queryCounter) Info(context.Context, string, ...any)             {}
func (c *queryCounter) Warn(context.Context, string, ...any)             {}
func (c *queryCounter) Error(context.Context, string, ...any)            {}
func (c *queryCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries++
}

// queriesFor appelle la route et retourne le nombre de requêtes SQL qu'elle a faites.
func (c *queryCounter) queriesFor(t *testing.T, a *actor, path string) int {
	t.Helper()
	c.mu.Lock()
	c.queries = 0
	c.mu.Unlock()
	if got := a.get(path); got.status != http.StatusOK {
		t.Fatalf("GET %s : %d %s", path, got.status, got.body)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.queries
}

// Une liste de modération lit l'identité de ses lignes en une seule requête : le nombre de requêtes ne dépend
// pas du nombre de lignes. Un FindByID dans une boucle (une requête par ligne) ferait échouer ce test.
func TestModerationListsDoNotQueryOncePerRow(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	counter := &queryCounter{}
	e.db.Logger = counter

	lists := []struct {
		name, path string
		add        func(n int)
	}{
		{"annuaire des héros", "/admin/heroes", func(n int) {
			e.approvedHero(moderator, fmt.Sprintf("hero%d@example.org", n), fmt.Sprintf("Hero %d", n))
		}},
		{"demandes de super-héros", "/admin/hero-requests", func(n int) {
			applicant := e.signUpAndLogin(fmt.Sprintf("applicant%d@example.org", n))
			if got := applicant.post("/heroes/requests", map[string]string{"alias": fmt.Sprintf("Candidate %d", n)}); got.status != http.StatusCreated {
				t.Fatalf("demande : %d %s", got.status, got.body)
			}
		}},
		{"changements d'adresse email", "/admin/email-change-requests", func(n int) {
			owner := e.signUpAndLogin(fmt.Sprintf("owner%d@example.org", n))
			if got := owner.requestEmailChange(t, fmt.Sprintf("new%d@example.org", n)); got.status != http.StatusCreated {
				t.Fatalf("demande de changement : %d %s", got.status, got.body)
			}
		}},
	}
	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			list.add(1)
			oneRow := counter.queriesFor(t, moderator, list.path)
			for n := 2; n <= 4; n++ {
				list.add(n)
			}
			fourRows := counter.queriesFor(t, moderator, list.path)
			if fourRows != oneRow {
				t.Errorf("%d requêtes SQL pour 1 ligne, %d pour 4 : la liste fait une requête par ligne", oneRow, fourRows)
			}
		})
	}
}

// L'authentification d'une requête est un seul aller-retour : la session, le rôle, le statut du compte et l'état de
// la 2FA se lisent ensemble. GET /auth/me y ajoute la lecture du profil, et rien d'autre.
func TestAuthenticatingARequestCostsOneQuery(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	counter := &queryCounter{}
	e.db.Logger = counter

	if got := counter.queriesFor(t, citizen, "/auth/me"); got != 2 {
		t.Errorf("GET /auth/me : %d requêtes SQL, attendu 2 (session + profil)", got)
	}
}
