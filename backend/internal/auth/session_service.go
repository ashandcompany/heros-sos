package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// ErrSessionInvalid indique une session absente, expirée, révoquée ou dont le compte n'est
// plus actif. Les causes ne sont pas distinguées : le client n'a pas à les connaître.
var ErrSessionInvalid = errors.New("auth : session invalide ou expirée")

// IssueRequest décrit la session à ouvrir.
type IssueRequest struct {
	UserID     uuid.UUID
	Role       user.Role
	RememberMe bool
	IP         netip.Addr
	UserAgent  string
}

// IssuedSession est une session qui vient d'être ouverte. Token n'existe qu'ici : il part dans
// le cookie et n'est jamais stocké.
type IssuedSession struct {
	Token     string
	ExpiresAt time.Time
}

// SessionService ouvre, valide et révoque les sessions.
type SessionService struct {
	store  *SessionStore
	clock  clock.Clock
	cfg    config.Session
	logger *slog.Logger
}

// NewSessionService crée le service des sessions.
func NewSessionService(store *SessionStore, clk clock.Clock, cfg config.Session, logger *slog.Logger) *SessionService {
	return &SessionService{store: store, clock: clk, cfg: cfg, logger: logger}
}

// policy choisit les durées d'une session : les modérateurs ont des sessions courtes et sans
// « se souvenir de moi ».
func (s *SessionService) policy(role user.Role, rememberMe bool) config.SessionPolicy {
	switch {
	case role == user.RoleModerator:
		return s.cfg.Moderator
	case rememberMe:
		return s.cfg.RememberMe
	default:
		return s.cfg.Member
	}
}

// Issue ouvre une session. Chaque connexion produit un nouveau jeton (aucune session
// existante n'est réutilisée), ce qui interdit la fixation de session.
func (s *SessionService) Issue(ctx context.Context, req IssueRequest) (IssuedSession, error) {
	raw, hash, err := newToken()
	if err != nil {
		return IssuedSession{}, err
	}

	now := s.clock.Now()
	rememberMe := req.RememberMe && req.Role != user.RoleModerator
	session := Session{
		TokenHash:  hash,
		UserID:     req.UserID,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(s.policy(req.Role, rememberMe).MaxLifetime),
		RememberMe: rememberMe,
		// La double authentification (TOTP) arrive en version 2 : une session est aujourd'hui
		// considérée comme complète dès l'ouverture.
		MFAVerifiedAt: &now,
		DeviceLabel:   deviceLabel(req.UserAgent),
	}
	if req.IP.IsValid() {
		session.IPAddress = req.IP.String()
	}
	if err := s.store.Create(ctx, &session); err != nil {
		return IssuedSession{}, err
	}
	return IssuedSession{Token: raw, ExpiresAt: session.ExpiresAt}, nil
}

// Authenticate valide un jeton de session et retourne l'utilisateur correspondant.
func (s *SessionService) Authenticate(ctx context.Context, rawToken string) (Principal, error) {
	if !isWellFormedToken(rawToken) {
		return Principal{}, ErrSessionInvalid
	}
	hash := hashToken(rawToken)

	view, err := s.store.find(ctx, hash)
	if errors.Is(err, errSessionNotFound) {
		return Principal{}, ErrSessionInvalid
	}
	if err != nil {
		return Principal{}, err
	}

	now := s.clock.Now()
	role := user.Role(view.Role)
	policy := s.policy(role, view.Session.RememberMe)
	switch {
	case view.Session.RevokedAt != nil,
		!now.Before(view.Session.ExpiresAt),
		now.Sub(view.Session.LastSeenAt) > policy.IdleTimeout,
		user.Status(view.UserStatus) != user.StatusActive:
		return Principal{}, ErrSessionInvalid
	}

	// La date de dernière activité n'est écrite qu'une fois par période : une écriture par
	// requête surchargerait la base. Un échec d'écriture ne doit pas rejeter la requête.
	if now.Sub(view.Session.LastSeenAt) >= s.cfg.TouchPeriod {
		if err := s.store.touch(ctx, hash, now); err != nil {
			s.logger.WarnContext(ctx, "mise à jour de last_seen_at impossible", "error", err)
		}
	}

	return Principal{
		UserID:      view.Session.UserID,
		Role:        role,
		MFAVerified: view.Session.MFAVerifiedAt != nil,
	}, nil
}

// Revoke invalide la session d'un jeton. C'est sans effet pour un jeton inconnu ou déjà révoqué,
// ce qui rend la déconnexion idempotente.
func (s *SessionService) Revoke(ctx context.Context, rawToken string) error {
	if !isWellFormedToken(rawToken) {
		return nil
	}
	_, err := s.store.revoke(ctx, hashToken(rawToken), s.clock.Now())
	return err
}

// RevokeAll invalide toutes les sessions d'un utilisateur.
func (s *SessionService) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	_, err := s.store.RevokeAllForUser(ctx, userID, s.clock.Now())
	return err
}

// Cleanup supprime les sessions expirées ou révoquées depuis plus de retention.
func (s *SessionService) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	return s.store.DeleteStale(ctx, s.clock.Now().Add(-retention))
}
