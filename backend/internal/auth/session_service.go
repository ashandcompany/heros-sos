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
	// MFAVerified marque la session comme complète. Faux, elle n'ouvre que la 2FA (saisie du code, ou
	// activation) et la déconnexion, jusqu'à ce qu'un code la valide (voir Rotate).
	MFAVerified bool
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
	mfa    config.MFA
	logger *slog.Logger
}

// NewSessionService crée le service des sessions. mfa dit pour qui la 2FA est obligatoire : une session de ce
// compte n'est complète que s'il l'a activée et validée.
func NewSessionService(store *SessionStore, clk clock.Clock, cfg config.Session, mfa config.MFA, logger *slog.Logger) *SessionService {
	return &SessionService{store: store, clock: clk, cfg: cfg, mfa: mfa, logger: logger}
}

// mfaRequired indique si la 2FA est obligatoire pour ce rôle : seuls les modérateurs, et seulement si la
// configuration l'impose (toujours en production).
func (s *SessionService) mfaRequired(role user.Role) bool {
	return role == user.RoleModerator && s.mfa.ModeratorRequired
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
		TokenHash:   hash,
		UserID:      req.UserID,
		CreatedAt:   now,
		LastSeenAt:  now,
		ExpiresAt:   now.Add(s.policy(req.Role, rememberMe).MaxLifetime),
		RememberMe:  rememberMe,
		DeviceLabel: deviceLabel(req.UserAgent),
	}
	if req.MFAVerified {
		session.MFAVerifiedAt = &now
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

	// La 2FA obligatoire qui n'est pas activée limite la session, même si elle a été ouverte avant : une
	// 2FA réinitialisée par l'exploitation, ou l'obligation décidée après la connexion, prennent effet
	// aussitôt.
	verified := view.Session.MFAVerifiedAt != nil
	if s.mfaRequired(role) && !view.MFAEnrolled {
		verified = false
	}

	return Principal{
		UserID:      view.Session.UserID,
		Role:        role,
		MFAVerified: verified,
		MFAEnrolled: view.MFAEnrolled,
	}, nil
}

// Rotate valide la session d'un jeton (la double authentification vient d'être prouvée) en lui donnant un
// NOUVEAU jeton : l'ancien est révoqué, et le nouveau garde la limite absolue de la session et l'appareil
// d'origine. C'est la règle anti-fixation du STACK : un jeton qui aurait fuité avant la validation ne
// devient jamais une session complète. Avec revokeOthers, toutes les autres sessions du compte sont aussi
// fermées (activation de la 2FA : celles ouvertes avant n'ont jamais passé le second facteur).
func (s *SessionService) Rotate(ctx context.Context, rawToken string, revokeOthers bool) (IssuedSession, error) {
	if !isWellFormedToken(rawToken) {
		return IssuedSession{}, ErrSessionInvalid
	}
	oldHash := hashToken(rawToken)
	raw, hash, err := newToken()
	if err != nil {
		return IssuedSession{}, err
	}

	now := s.clock.Now()
	var issued IssuedSession
	err = s.store.withinTx(ctx, func(ctx context.Context) error {
		view, err := s.store.find(ctx, oldHash)
		if errors.Is(err, errSessionNotFound) || (err == nil && view.Session.RevokedAt != nil) {
			return ErrSessionInvalid
		}
		if err != nil {
			return err
		}
		if revokeOthers {
			if _, err := s.store.RevokeAllForUser(ctx, view.Session.UserID, now); err != nil {
				return err
			}
		} else if err := s.store.revoke(ctx, oldHash, now); err != nil {
			return err
		}

		session := Session{
			TokenHash: hash, UserID: view.Session.UserID, CreatedAt: now, LastSeenAt: now,
			ExpiresAt: view.Session.ExpiresAt, RememberMe: view.Session.RememberMe, MFAVerifiedAt: &now,
			DeviceLabel: view.Session.DeviceLabel,
		}
		if view.Session.IPAddress != nil {
			session.IPAddress = *view.Session.IPAddress
		}
		if err := s.store.Create(ctx, &session); err != nil {
			return err
		}
		issued = IssuedSession{Token: raw, ExpiresAt: session.ExpiresAt}
		return nil
	})
	return issued, err
}

// Revoke invalide la session d'un jeton. C'est sans effet pour un jeton inconnu ou déjà révoqué,
// ce qui rend la déconnexion idempotente.
func (s *SessionService) Revoke(ctx context.Context, rawToken string) error {
	if !isWellFormedToken(rawToken) {
		return nil
	}
	return s.store.revoke(ctx, hashToken(rawToken), s.clock.Now())
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
