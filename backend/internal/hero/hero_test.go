package hero_test

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

var (
	ctx     = context.Background()
	startAt = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	someIP  = netip.MustParseAddr("203.0.113.7")
)

type fixture struct {
	db      *gorm.DB
	clock   *clock.Fake
	users   user.Repository
	repo    hero.Repository
	service *hero.Service
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	clk := clock.NewFake(startAt)
	users := user.NewRepository(db, sealer)
	repo := hero.NewRepository(db)
	return fixture{
		db: db, clock: clk, users: users, repo: repo,
		service: hero.NewService(repo, users, audit.NewStore(db, clk), database.NewTransactor(db), clk),
	}
}

func (f fixture) newUser(t *testing.T, email string, role user.Role, status user.Status) *user.User {
	t.Helper()
	u := &user.User{
		ID: ids.New(), Email: email, PasswordHash: "$argon2id$x", Role: role, Status: user.StatusPendingActivation,
		Civility: "madame", FirstName: "Natasha", LastName: "Romanoff",
		BirthDate: time.Date(1984, time.November, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := f.users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if status == user.StatusActive {
		if _, err := f.users.Activate(ctx, u.ID, startAt); err != nil {
			t.Fatal(err)
		}
	}
	return u
}

func (f fixture) count(t *testing.T, where string, args ...any) int64 {
	t.Helper()
	var total int64
	if err := f.db.Table("audit_logs").Where(where, args...).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	return total
}

func TestRequestRequiresAnActiveCitizen(t *testing.T) {
	f := newFixture(t)
	input := hero.RequestInput{Alias: "Black Widow"}

	tests := []struct {
		name string
		role user.Role
		stat user.Status
		want error
	}{
		{"citoyen actif", user.RoleCitizen, user.StatusActive, nil},
		{"citoyen non activé", user.RoleCitizen, user.StatusPendingActivation, hero.ErrNotEligible},
		{"modérateur", user.RoleModerator, user.StatusActive, hero.ErrNotEligible},
		{"déjà héros", user.RoleHero, user.StatusActive, hero.ErrAlreadyHero},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			account := f.newUser(t, "compte"+string(rune('a'+i))+"@example.org", test.role, test.stat)
			in := input
			in.Alias = "Pseudo " + string(rune('A'+i))
			if err := f.service.Request(ctx, account.ID, in, someIP); !errors.Is(err, test.want) {
				t.Fatalf("erreur = %v, attendu %v", err, test.want)
			}
		})
	}
}

func TestRequestValidatesTheForm(t *testing.T) {
	f := newFixture(t)
	account := f.newUser(t, "valide@example.org", user.RoleCitizen, user.StatusActive)

	var validationErr *validation.Error
	err := f.service.Request(ctx, account.ID, hero.RequestInput{
		Alias: "x", Bio: string(make([]rune, validation.MaxBioLength+1)), Message: "",
	}, someIP)
	if !errors.As(err, &validationErr) || validationErr.Fields["alias"] == "" || validationErr.Fields["bio"] == "" {
		t.Fatalf("erreur = %v", err)
	}
	if f.count(t, "true") != 0 {
		t.Error("une saisie invalide ne doit rien journaliser")
	}
}

func TestResubmissionOnlyAfterRejection(t *testing.T) {
	f := newFixture(t)
	moderator := f.newUser(t, "mod@example.org", user.RoleModerator, user.StatusActive)
	account := f.newUser(t, "candidat@example.org", user.RoleCitizen, user.StatusActive)

	if err := f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Premier Pseudo"}, someIP); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Autre Pseudo"}, someIP); !errors.Is(err, hero.ErrAlreadyPending) {
		t.Fatalf("redépôt pendant l'examen : %v", err)
	}
	if err := f.service.Reject(ctx, moderator.ID, account.ID, "Non", someIP); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Second Pseudo", Bio: "Nouvelle bio"}, someIP); err != nil {
		t.Fatalf("redépôt après refus : %v", err)
	}

	profile, err := f.service.Mine(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status != hero.StatusPending || profile.Alias != "Second Pseudo" || profile.Bio != "Nouvelle bio" ||
		profile.RejectionReason != "" || profile.ReviewedBy != nil || profile.ReviewedAt != nil {
		t.Errorf("profil après redépôt = %+v", profile)
	}
}

func TestApproveGrantsTheRoleAtomicallyAndOnlyOnce(t *testing.T) {
	f := newFixture(t)
	moderator := f.newUser(t, "mod@example.org", user.RoleModerator, user.StatusActive)
	account := f.newUser(t, "candidat@example.org", user.RoleCitizen, user.StatusActive)
	_ = f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Black Widow"}, someIP)

	if err := f.service.Approve(ctx, moderator.ID, account.ID, someIP); err != nil {
		t.Fatal(err)
	}
	got, _ := f.users.FindByID(ctx, account.ID)
	if got.Role != user.RoleHero || got.DisplayName() != "Black Widow" {
		t.Errorf("compte après validation = %+v", got)
	}
	profile, _ := f.service.Mine(ctx, account.ID)
	if profile.Status != hero.StatusApproved || profile.ReviewedBy == nil || *profile.ReviewedBy != moderator.ID {
		t.Errorf("profil = %+v", profile)
	}

	if err := f.service.Approve(ctx, moderator.ID, account.ID, someIP); !errors.Is(err, hero.ErrNotPending) {
		t.Errorf("seconde validation : %v", err)
	}
	if err := f.service.Reject(ctx, moderator.ID, account.ID, "", someIP); !errors.Is(err, hero.ErrNotPending) {
		t.Errorf("refus après validation : %v", err)
	}
	if err := f.service.Approve(ctx, moderator.ID, uuid.New(), someIP); !errors.Is(err, hero.ErrNotFound) {
		t.Errorf("demande inconnue : %v", err)
	}
	if f.count(t, "action = 'hero.approve'") != 1 {
		t.Error("une seule validation doit être journalisée")
	}
}

func TestApproveRollsBackWhenTheRoleCannotBeChanged(t *testing.T) {
	f := newFixture(t)
	moderator := f.newUser(t, "mod@example.org", user.RoleModerator, user.StatusActive)
	account := f.newUser(t, "candidat@example.org", user.RoleCitizen, user.StatusActive)
	_ = f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Black Widow"}, someIP)

	// Le compte disparaît entre la demande et la validation : la demande disparaît avec lui
	// (suppression en cascade) et rien ne doit rester à moitié fait.
	if err := f.db.Exec(`DELETE FROM users WHERE id = ?`, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.service.Approve(ctx, moderator.ID, account.ID, someIP); !errors.Is(err, hero.ErrNotFound) {
		t.Fatalf("erreur = %v", err)
	}
	if f.count(t, "action = 'hero.approve'") != 0 {
		t.Error("aucune trace d'audit pour une validation qui n'a pas eu lieu")
	}
}

func TestConcurrentModeratorsCannotBothDecide(t *testing.T) {
	f := newFixture(t)
	account := f.newUser(t, "candidat@example.org", user.RoleCitizen, user.StatusActive)
	_ = f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Black Widow"}, someIP)

	const moderators = 10
	var wg sync.WaitGroup
	var approved, refused, blocked atomic.Int32
	for i := range moderators {
		mod := f.newUser(t, "mod"+string(rune('a'+i))+"@example.org", user.RoleModerator, user.StatusActive)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if i%2 == 0 {
				err = f.service.Approve(ctx, mod.ID, account.ID, someIP)
			} else {
				err = f.service.Reject(ctx, mod.ID, account.ID, "Non", someIP)
			}
			switch {
			case err == nil && i%2 == 0:
				approved.Add(1)
			case err == nil:
				refused.Add(1)
			case errors.Is(err, hero.ErrNotPending):
				blocked.Add(1)
			default:
				t.Errorf("erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()

	if approved.Load()+refused.Load() != 1 || blocked.Load() != moderators-1 {
		t.Fatalf("décisions = %d validée(s) + %d refusée(s), %d bloquée(s) : une seule décision est permise",
			approved.Load(), refused.Load(), blocked.Load())
	}
	// Le rôle doit être cohérent avec la décision retenue.
	got, _ := f.users.FindByID(ctx, account.ID)
	profile, _ := f.service.Mine(ctx, account.ID)
	if (profile.Status == hero.StatusApproved) != (got.Role == user.RoleHero) {
		t.Fatalf("incohérence : demande %q mais rôle %q", profile.Status, got.Role)
	}
}

func TestConcurrentRequestsForTheSameAliasYieldOneWinner(t *testing.T) {
	f := newFixture(t)

	const candidates = 10
	var wg sync.WaitGroup
	var won, taken atomic.Int32
	for i := range candidates {
		account := f.newUser(t, "candidat"+string(rune('a'+i))+"@example.org", user.RoleCitizen, user.StatusActive)
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Spider-Man"}, someIP); {
			case err == nil:
				won.Add(1)
			case errors.Is(err, hero.ErrAliasTaken):
				taken.Add(1)
			default:
				t.Errorf("erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()

	if won.Load() != 1 || taken.Load() != candidates-1 {
		t.Fatalf("gagnants = %d, refusés = %d", won.Load(), taken.Load())
	}
}

func TestListShowsOldestFirstWithApplicantIdentity(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"premier", "second", "troisieme"} {
		account := f.newUser(t, name+"@example.org", user.RoleCitizen, user.StatusActive)
		f.clock.Advance(time.Minute)
		if err := f.service.Request(ctx, account.ID, hero.RequestInput{Alias: "Pseudo " + name}, someIP); err != nil {
			t.Fatal(err)
		}
	}

	applicants, total, err := f.service.List(ctx, hero.StatusPending, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(applicants) != 2 {
		t.Fatalf("total = %d, page = %d", total, len(applicants))
	}
	if applicants[0].Alias != "Pseudo premier" || applicants[1].Alias != "Pseudo second" {
		t.Errorf("ordre = %s, %s : les plus anciennes demandes d'abord", applicants[0].Alias, applicants[1].Alias)
	}
	if applicants[0].FirstName != "Natasha" || applicants[0].LastName != "Romanoff" || applicants[0].Email != "premier@example.org" {
		t.Errorf("identité du demandeur = %+v", applicants[0])
	}

	none, total, _ := f.service.List(ctx, hero.StatusApproved, 10, 0)
	if total != 0 || len(none) != 0 {
		t.Errorf("aucune demande validée : %d", total)
	}
}
