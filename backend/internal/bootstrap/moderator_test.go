package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/bootstrap"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
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
