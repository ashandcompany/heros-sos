package film_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/film"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

var filmTestCtx = context.Background()

type catalogStub struct {
	calls int
	title film.Title
	raw   json.RawMessage
	err   error
}

func (c *catalogStub) Search(context.Context, string) ([]film.Title, error) { return nil, nil }

func (c *catalogStub) Get(context.Context, string) (*film.Title, json.RawMessage, error) {
	c.calls++
	if c.err != nil {
		return nil, nil, c.err
	}
	return &c.title, c.raw, nil
}

func TestFilmCacheAndHeroModeration(t *testing.T) {
	db := testutil.NewDB(t)
	now := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)
	clk := clock.NewFake(now)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	users := user.NewRepository(db, sealer)
	heroes := hero.NewRepository(db)
	catalog := &catalogStub{
		title: film.Title{ID: "tt0133093", Title: "The Matrix", PosterURL: "https://example.org/poster.jpg", Plot: "A hacker discovers reality."},
		raw:   json.RawMessage(`{"Response":"True"}`),
	}
	service := film.NewService(film.ServiceDeps{
		Repo: film.NewRepository(db), Catalog: catalog, Heroes: heroes,
		Villains: villain.NewRepository(db), Audit: audit.NewStore(db, clk),
		Tx: database.NewTransactor(db), Clock: clk,
	})

	heroAccount := newFilmTestUser(t, users, user.RoleHero, "hero@example.org")
	moderator := newFilmTestUser(t, users, user.RoleModerator, "moderator@example.org")
	if err := db.Exec(`INSERT INTO hero_profiles (user_id, alias, verification_status, created_at)
VALUES (?, ?, 'approved', ?)`, heroAccount.ID, "Lumen", now).Error; err != nil {
		t.Fatal(err)
	}

	created, err := service.AddToHero(filmTestCtx, heroAccount.ID, user.RoleHero, uuid.Nil,
		film.FilmInput{IMDBID: "tt0133093"}, netip.MustParseAddr("203.0.113.7"))
	if err != nil {
		t.Fatal(err)
	}
	if created.PosterURL != "https://example.org/poster.jpg" || created.Raw != string(catalog.raw) || catalog.calls != 1 {
		t.Fatalf("film mis en cache = %+v, appels catalogue = %d", created, catalog.calls)
	}
	if _, err := service.AddToHero(filmTestCtx, heroAccount.ID, user.RoleHero, uuid.Nil,
		film.FilmInput{IMDBID: "tt0133093"}, netip.Addr{}); !errors.Is(err, film.ErrAlreadyLinked) {
		t.Fatalf("doublon = %v, attendu %v", err, film.ErrAlreadyLinked)
	}
	if catalog.calls != 1 {
		t.Fatalf("le cache OMDb aurait dû éviter un nouvel appel : %d", catalog.calls)
	}

	items, total, err := service.ListHeroFilms(filmTestCtx, film.HeroFilmFilter{HeroID: &heroAccount.ID}, 20, 0)
	if err != nil || total != 1 || items[0].Status != film.StatusPending {
		t.Fatalf("liste en attente = %+v, total %d, erreur %v", items, total, err)
	}
	if err := service.Moderate(filmTestCtx, moderator.ID, heroAccount.ID, created.ID, film.StatusApproved, netip.Addr{}); err != nil {
		t.Fatal(err)
	}
	public, total, err := service.ListHeroFilms(filmTestCtx, film.HeroFilmFilter{HeroID: &heroAccount.ID, Status: film.StatusApproved}, 20, 0)
	if err != nil || total != 1 || public[0].Status != film.StatusApproved {
		t.Fatalf("film publié = %+v, total %d, erreur %v", public, total, err)
	}
	if err := service.Moderate(filmTestCtx, moderator.ID, heroAccount.ID, created.ID, film.StatusRejected, netip.Addr{}); err != nil {
		t.Fatalf("révocation de la publication = %v", err)
	}
	if err := service.Moderate(filmTestCtx, moderator.ID, heroAccount.ID, created.ID, film.StatusRejected, netip.Addr{}); !errors.Is(err, film.ErrAlreadyReview) {
		t.Fatalf("décision identique = %v, attendu %v", err, film.ErrAlreadyReview)
	}
}

func newFilmTestUser(t *testing.T, users user.Repository, role user.Role, email string) *user.User {
	t.Helper()
	account := &user.User{
		ID: ids.New(), Email: email, PasswordHash: "$argon2id$x", Role: role, Status: user.StatusActive,
		Civility: "madame", FirstName: "Natasha", LastName: "Romanoff",
		BirthDate: time.Date(1984, time.November, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := users.Create(filmTestCtx, account); err != nil {
		t.Fatal(err)
	}
	return account
}

func TestPublicFilmsGroupedByFilm(t *testing.T) {
	db := testutil.NewDB(t)
	now := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	clk := clock.NewFake(now)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	users := user.NewRepository(db, sealer)
	service := film.NewService(film.ServiceDeps{
		Repo: film.NewRepository(db),
		Catalog: &catalogStub{
			title: film.Title{ID: "tt0133093", Title: "The Matrix"}, raw: json.RawMessage(`{"Response":"True"}`),
		},
		Heroes: hero.NewRepository(db), Villains: villain.NewRepository(db), Audit: audit.NewStore(db, clk),
		Tx: database.NewTransactor(db), Clock: clk,
	})

	lumen := newFilmTestUser(t, users, user.RoleHero, "lumen@example.org")
	brasier := newFilmTestUser(t, users, user.RoleHero, "brasier@example.org")
	moderator := newFilmTestUser(t, users, user.RoleModerator, "moderator@example.org")
	for alias, account := range map[string]*user.User{"Lumen": lumen, "Brasier": brasier} {
		if err := db.Exec(`INSERT INTO hero_profiles (user_id, alias, verification_status, created_at)
VALUES (?, ?, 'approved', ?)`, account.ID, alias, now).Error; err != nil {
			t.Fatal(err)
		}
	}

	// The Matrix est rattaché aux deux héros : une seule entrée publique.
	for _, heroID := range []uuid.UUID{lumen.ID, brasier.ID} {
		if _, err := service.AddApprovedToHero(filmTestCtx, moderator.ID, heroID,
			film.FilmInput{IMDBID: "tt0133093"}, netip.Addr{}); err != nil {
			t.Fatal(err)
		}
	}
	clk.Advance(time.Minute)
	if _, err := service.AddApprovedToHero(filmTestCtx, moderator.ID, lumen.ID,
		film.FilmInput{Title: "Lumen à 100%", VideoURL: "https://example.org/lumen"}, netip.Addr{}); err != nil {
		t.Fatal(err)
	}
	// En attente de modération : invisible du public.
	if _, err := service.AddToHero(filmTestCtx, brasier.ID, user.RoleHero, uuid.Nil,
		film.FilmInput{Title: "Brasier secret", VideoURL: "https://example.org/brasier"}, netip.Addr{}); err != nil {
		t.Fatal(err)
	}

	all, total, err := service.ListPublicFilms(filmTestCtx, film.PublicFilmFilter{}, 20, 0)
	if err != nil || total != 2 || len(all) != 2 || all[0].Title != "Lumen à 100%" {
		t.Fatalf("vidéothèque = %+v, total %d, erreur %v", all, total, err)
	}
	if heroes := all[1].Heroes; len(heroes) != 2 || heroes[0].Alias != "Brasier" || heroes[1].Alias != "Lumen" {
		t.Fatalf("héros de The Matrix = %+v", heroes)
	}

	byHero, total, err := service.ListPublicFilms(filmTestCtx, film.PublicFilmFilter{HeroID: &brasier.ID}, 20, 0)
	if err != nil || total != 1 || byHero[0].Title != "The Matrix" || len(byHero[0].Heroes) != 2 {
		t.Fatalf("films de Brasier = %+v, total %d, erreur %v", byHero, total, err)
	}
	for query, want := range map[string]int64{"matrix": 1, "100%": 1, "0_": 0, "secret": 0} {
		if _, total, err := service.ListPublicFilms(filmTestCtx, film.PublicFilmFilter{Title: query}, 20, 0); err != nil || total != want {
			t.Fatalf("recherche %q : total %d, attendu %d, erreur %v", query, total, want, err)
		}
	}
	paged, total, err := service.ListPublicFilms(filmTestCtx, film.PublicFilmFilter{}, 1, 1)
	if err != nil || total != 2 || len(paged) != 1 || paged[0].Title != "The Matrix" {
		t.Fatalf("deuxième page = %+v, total %d, erreur %v", paged, total, err)
	}

	heroes, err := service.ListFilmHeroes(filmTestCtx)
	if err != nil || len(heroes) != 2 || heroes[0].Alias != "Brasier" || heroes[0].FilmCount != 1 ||
		heroes[1].Alias != "Lumen" || heroes[1].FilmCount != 2 {
		t.Fatalf("héros de la vidéothèque = %+v, erreur %v", heroes, err)
	}

	pending, total, err := service.ListHeroFilms(filmTestCtx, film.HeroFilmFilter{Status: "all", Title: "SECRET"}, 20, 0)
	if err != nil || total != 1 || pending[0].Status != film.StatusPending {
		t.Fatalf("recherche de modération = %+v, total %d, erreur %v", pending, total, err)
	}
}
