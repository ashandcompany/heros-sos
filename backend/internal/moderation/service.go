// Package moderation regroupe les actions des modérateurs sur les comptes : lister, vérifier,
// désactiver et réactiver (cahier §3.1.3). La validation des super-héros est dans le paquet hero.
package moderation

import (
	"context"
	"errors"
	"net/netip"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

var (
	// ErrSelfAction interdit à un modérateur de se désactiver lui-même.
	ErrSelfAction = errors.New("moderation : action interdite sur son propre compte")
	// ErrProtectedAccount interdit de désactiver un modérateur par l'API : ces comptes se gèrent
	// hors ligne, pour qu'un compte compromis ne puisse pas verrouiller toute l'équipe.
	ErrProtectedAccount = errors.New("moderation : compte de modérateur protégé")
)

// SessionRevoker ferme les sessions d'un utilisateur (auth.SessionService).
type SessionRevoker interface {
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

// Service porte les actions de modération sur les comptes.
type Service struct {
	users    user.Repository
	sessions SessionRevoker
	audit    audit.Recorder
	tx       database.Transactor
	clock    clock.Clock
}

// NewService crée le service.
func NewService(users user.Repository, sessions SessionRevoker, recorder audit.Recorder, tx database.Transactor, clk clock.Clock) *Service {
	return &Service{users: users, sessions: sessions, audit: recorder, tx: tx, clock: clk}
}

// ListUsers retourne une page de comptes filtrée (rôle, statut, début d'adresse email).
func (s *Service) ListUsers(ctx context.Context, filter user.Filter) ([]user.User, int64, error) {
	return s.users.List(ctx, filter)
}

// Deactivate désactive un compte et ferme immédiatement toutes ses sessions. L'opération est
// idempotente : désactiver un compte déjà désactivé réussit sans écrire de nouvelle trace.
func (s *Service) Deactivate(ctx context.Context, actorID, targetID uuid.UUID, ip netip.Addr) error {
	if actorID == targetID {
		return ErrSelfAction
	}
	target, err := s.users.FindByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == user.RoleModerator {
		return ErrProtectedAccount
	}

	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		changed, err := s.users.Deactivate(ctx, targetID, s.clock.Now())
		if err != nil || !changed {
			return err
		}
		if err := s.sessions.RevokeAll(ctx, targetID); err != nil {
			return err
		}
		return s.record(ctx, actorID, targetID, audit.ActionUserDeactivated, ip)
	})
}

// Reactivate rétablit un compte désactivé. C'est idempotent.
func (s *Service) Reactivate(ctx context.Context, actorID, targetID uuid.UUID, ip netip.Addr) error {
	if _, err := s.users.FindByID(ctx, targetID); err != nil {
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		changed, err := s.users.Reactivate(ctx, targetID, s.clock.Now())
		if err != nil || !changed {
			return err
		}
		return s.record(ctx, actorID, targetID, audit.ActionUserReactivated, ip)
	})
}

// Verify marque un citoyen comme « vérifié » : une information de confiance, qui ne donne aucun
// droit supplémentaire. C'est idempotent : la date de la première vérification est conservée.
func (s *Service) Verify(ctx context.Context, actorID, targetID uuid.UUID, ip netip.Addr) error {
	if _, err := s.users.FindByID(ctx, targetID); err != nil {
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		changed, err := s.users.MarkVerified(ctx, targetID, actorID, s.clock.Now())
		if err != nil || !changed {
			return err
		}
		return s.record(ctx, actorID, targetID, audit.ActionUserVerified, ip)
	})
}

func (s *Service) record(ctx context.Context, actorID, targetID uuid.UUID, action string, ip netip.Addr) error {
	return s.audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: "user", EntityID: &targetID, IP: ip,
	})
}
