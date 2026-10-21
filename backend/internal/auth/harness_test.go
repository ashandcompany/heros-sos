package auth_test

import (
	"context"
	"log/slog"
	"net/netip"
	"regexp"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

const (
	goodPassword  = "Shield1234abc"
	otherPassword = "Hammer5678xyz"
)

var (
	startTime  = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	testIP     = netip.MustParseAddr("203.0.113.7")
	tokenInURL = regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`)
)

// harness assemble le service d'authentification sur une vraie base PostgreSQL isolée, avec une
// horloge contrôlée et un enregistreur de mails.
type harness struct {
	t        *testing.T
	db       *gorm.DB
	clock    *clock.Fake
	cfg      config.Config
	users    user.Repository
	tokens   *auth.TokenStore
	sessions *auth.SessionService
	hasher   *auth.PasswordHasher
	service  *auth.Service
	mailer   *mail.Recorder
	sealer   *crypto.Sealer
}

func newHarness(t *testing.T, overrides map[string]string) *harness {
	t.Helper()
	cfg := testutil.Config(t, overrides)
	db := testutil.NewDB(t)
	logger := slog.New(slog.DiscardHandler)
	clk := clock.NewFake(startTime)

	sealer, err := crypto.NewSealer(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		t.Fatal(err)
	}
	users := user.NewRepository(db, sealer)
	tokens := auth.NewTokenStore(db)
	sessions := auth.NewSessionService(auth.NewSessionStore(db), clk, cfg.Session, cfg.Security.MFA, logger)
	hasher := auth.NewPasswordHasher(cfg.Security.Argon2)

	recorder := mail.NewRecorder()
	mailService, err := mail.NewService(recorder, cfg.Mail.FromName, cfg.Security.Tokens)
	if err != nil {
		t.Fatal(err)
	}

	service := auth.NewService(auth.ServiceDeps{
		Users: users, Tokens: tokens, MFA: auth.NewMFAStore(db, sealer), Sessions: sessions, Hasher: hasher, Mailer: mailService,
		Audit: audit.NewStore(db, clk), Tx: database.NewTransactor(db), Clock: clk,
		Security: cfg.Security, App: cfg.App, Logger: logger,
	})
	return &harness{
		t: t, db: db, clock: clk, cfg: cfg, users: users, tokens: tokens, sessions: sessions,
		hasher: hasher, service: service, mailer: recorder, sealer: sealer,
	}
}

func registration(email string) auth.RegisterInput {
	return auth.RegisterInput{
		Civility:  "monsieur",
		FirstName: "Steve",
		LastName:  "Rogers",
		BirthDate: "1990-07-04",
		Email:     email,
		Phone:     "06 12 34 56 78",
		Address:   user.Address{Street: "569 Leaman Place", PostalCode: "11201", City: "Brooklyn", Country: "US"},
		Password:  goodPassword,
	}
}

// registerAndActivate crée un compte citoyen actif en passant par le parcours réel.
func (h *harness) registerAndActivate(email string) *user.User {
	h.t.Helper()
	u, err := h.service.Register(context.Background(), registration(email))
	if err != nil {
		h.t.Fatalf("Register(%s) : %v", email, err)
	}
	if err := h.service.Activate(context.Background(), h.lastToken()); err != nil {
		h.t.Fatalf("Activate(%s) : %v", email, err)
	}
	return u
}

// createUser insère directement un compte, avec le rôle et le statut voulus.
func (h *harness) createUser(email string, role user.Role, status user.Status) *user.User {
	h.t.Helper()
	hash, err := h.hasher.Hash(context.Background(), goodPassword)
	if err != nil {
		h.t.Fatal(err)
	}
	u := &user.User{
		ID: ids.New(), Email: email, PasswordHash: hash, Role: role, Status: user.StatusPendingActivation,
		Civility: "madame", FirstName: "Natasha", LastName: "Romanoff",
		BirthDate: time.Date(1984, time.November, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := h.users.Create(context.Background(), u); err != nil {
		h.t.Fatal(err)
	}
	if status == user.StatusActive {
		if _, err := h.users.Activate(context.Background(), u.ID, h.clock.Now()); err != nil {
			h.t.Fatal(err)
		}
	}
	return u
}

func (h *harness) login(identifier, password string) (auth.LoginResult, error) {
	h.t.Helper()
	return h.service.Login(context.Background(), auth.LoginInput{
		Identifier: identifier, Password: password, IP: testIP, UserAgent: "Mozilla/5.0 (Macintosh) Chrome/126.0 Safari/537.36",
	})
}

// lastToken extrait le jeton du lien contenu dans le dernier mail envoyé.
func (h *harness) lastToken() string {
	h.t.Helper()
	message, ok := h.mailer.Last()
	if !ok {
		h.t.Fatal("aucun mail envoyé")
	}
	match := tokenInURL.FindStringSubmatch(message.Text)
	if match == nil {
		h.t.Fatalf("aucun jeton dans le mail :\n%s", message.Text)
	}
	return match[1]
}

func (h *harness) count(table, where string, args ...any) int64 {
	h.t.Helper()
	var total int64
	if err := h.db.Table(table).Where(where, args...).Count(&total).Error; err != nil {
		h.t.Fatal(err)
	}
	return total
}
