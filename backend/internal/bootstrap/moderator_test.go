package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/bootstrap"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

func deps(t *testing.T) (bootstrap.ModeratorDeps, user.Repository) {
	t.Helper()
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	users := user.NewRepository(testutil.NewDB(t), sealer)
	return bootstrap.ModeratorDeps{
		Users:    users,
		Hasher:   auth.NewPasswordHasher(config.Argon2{Time: 1, MemoryKiB: 64, Threads: 1, KeyLength: 32, SaltLength: 16, MaxConcurrent: 2}),
		Clock:    clock.NewFake(time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)),
		Password: config.PasswordPolicy{MinLength: 10, MaxLength: 128},
	}, users
}

func input() bootstrap.ModeratorInput {
	return bootstrap.ModeratorInput{
		Email: "Fury@Shield.example", FirstName: "Nick", LastName: "Fury", BirthDate: "1950-12-04", Password: "Eyepatch1234",
	}
}

func TestCreateModeratorMakesAnActiveModerator(t *testing.T) {
	d, users := deps(t)

	created, err := bootstrap.CreateModerator(context.Background(), d, input())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := users.FindByEmail(context.Background(), "fury@shield.example")
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != created.ID || stored.Role != user.RoleModerator || stored.Status != user.StatusActive || stored.ActivatedAt == nil {
		t.Errorf("compte = %+v", stored)
	}
	if stored.FirstName != "Nick" || stored.LastName != "Fury" {
		t.Errorf("identité = %q %q", stored.FirstName, stored.LastName)
	}
	check, err := d.Hasher.Verify(context.Background(), stored.PasswordHash, "Eyepatch1234")
	if err != nil || !check.Match {
		t.Errorf("le mot de passe doit être stocké haché et vérifiable : (%+v, %v)", check, err)
	}
}

func TestCreateModeratorValidatesLikeARegistration(t *testing.T) {
	d, _ := deps(t)
	in := bootstrap.ModeratorInput{Email: "pas-un-email", FirstName: "", LastName: "1", BirthDate: "demain", Password: "faible"}

	_, err := bootstrap.CreateModerator(context.Background(), d, in)

	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("erreur = %v", err)
	}
	for _, field := range []string{"email", "firstName", "lastName", "birthDate", "password"} {
		if validationErr.Fields[field] == "" {
			t.Errorf("champ %q non signalé : %v", field, validationErr.Fields)
		}
	}
}

func TestCreateModeratorRefusesAnExistingAddress(t *testing.T) {
	d, _ := deps(t)
	if _, err := bootstrap.CreateModerator(context.Background(), d, input()); err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.CreateModerator(context.Background(), d, input()); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("erreur = %v, attendu ErrEmailTaken", err)
	}
}

func TestResetMFARemovesTheSecondFactorAndClosesEverySession(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	clk := clock.NewFake(time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC))
	users := user.NewRepository(db, sealer)
	mfa := auth.NewMFAStore(db, sealer)
	sessions := auth.NewSessionService(auth.NewSessionStore(db), clk, config.Session{
		Member:    config.SessionPolicy{IdleTimeout: time.Hour, MaxLifetime: 24 * time.Hour},
		Moderator: config.SessionPolicy{IdleTimeout: time.Hour, MaxLifetime: 8 * time.Hour}, TouchPeriod: time.Minute,
	}, config.MFA{}, slog.New(slog.DiscardHandler))

	d, _ := deps(t)
	d.Users = users
	moderator, err := bootstrap.CreateModerator(ctx, d, input())
	if err != nil {
		t.Fatal(err)
	}
	if started, err := mfa.StartSetup(ctx, moderator.ID, bytes.Repeat([]byte{1}, 20), clk.Now()); err != nil || !started {
		t.Fatalf("StartSetup : %v %v", started, err)
	}
	if confirmed, err := mfa.Confirm(ctx, moderator.ID, 1, clk.Now()); err != nil || !confirmed {
		t.Fatalf("Confirm : %v %v", confirmed, err)
	}
	if err := mfa.ReplaceRecoveryCodes(ctx, moderator.ID, [][]byte{[]byte("a"), []byte("b")}); err != nil {
		t.Fatal(err)
	}
	issued, err := sessions.Issue(ctx, auth.IssueRequest{UserID: moderator.ID, Role: user.RoleModerator, MFAVerified: true})
	if err != nil {
		t.Fatal(err)
	}

	reset := bootstrap.ResetMFADeps{
		Users: users, MFA: mfa, Sessions: sessions, Audit: audit.NewStore(db, clk), Tx: database.NewTransactor(db), Clock: clk,
	}
	if _, err := bootstrap.ResetMFA(ctx, reset, "fury@shield.example"); err != nil {
		t.Fatal(err)
	}

	if enrolled, _ := mfa.IsEnrolled(ctx, moderator.ID); enrolled {
		t.Error("la 2FA doit être supprimée")
	}
	if remaining, _ := mfa.RecoveryCodesRemaining(ctx, moderator.ID); remaining != 0 {
		t.Errorf("les codes de secours doivent être supprimés : %d", remaining)
	}
	if _, err := sessions.Authenticate(ctx, issued.Token); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Errorf("les sessions doivent être fermées : %v", err)
	}
	var entries int64
	db.Table("audit_logs").Where("action = ? AND actor_id IS NULL AND entity_id = ?", audit.ActionMFAReset, moderator.ID).Count(&entries)
	if entries != 1 {
		t.Errorf("la réinitialisation doit être journalisée : %d", entries)
	}

	// Un compte inconnu ou une adresse invalide ne réinitialise rien.
	if _, err := bootstrap.ResetMFA(ctx, reset, "inconnu@shield.example"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("compte inconnu : %v", err)
	}
	var invalid *validation.Error
	if _, err := bootstrap.ResetMFA(ctx, reset, "pas-un-email"); !errors.As(err, &invalid) {
		t.Errorf("adresse invalide : %v", err)
	}
}
