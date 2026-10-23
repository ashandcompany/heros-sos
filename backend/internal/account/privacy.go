package account

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Droit d'accès et droit à l'effacement (RGPD). L'export rassemble le profil et l'activité du titulaire.
// La suppression d'un compte l'anonymise plutôt que d'effacer la ligne : ses incidents, ses interventions et
// ses avis restent cohérents, rattachés à un « compte supprimé » (voir user.Repository.Anonymize).

var (
	// ErrNotErasable indique un compte qu'un modérateur ne peut pas anonymiser : le sien (il passe par la
	// suppression de son compte, qui redemande son mot de passe) ou celui d'un autre modérateur.
	ErrNotErasable = errors.New("account : ce compte ne peut pas être anonymisé par un modérateur")
	// ErrAlreadyErased indique un compte déjà anonymisé.
	ErrAlreadyErased = errors.New("account : compte déjà supprimé")
)

// IdentityChecker redemande le mot de passe et, si la 2FA est activée, un code, avant une action irréversible
// (implémenté par auth.Service et auth.MFAService).
type IdentityChecker interface {
	CheckPassword(ctx context.Context, userID uuid.UUID, password string) error
	CheckCodeIfEnrolled(ctx context.Context, userID uuid.UUID, code string, ip netip.Addr) error
}

// PasswordHasher calcule l'empreinte d'un mot de passe (implémenté par auth.PasswordHasher).
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
}

// Export est l'ensemble des données d'un compte : son profil et son activité.
type Export struct {
	Account  *Account
	Activity *Activity
}

// Export rassemble les données d'un compte, pour son titulaire.
func (s *Service) Export(ctx context.Context, userID uuid.UUID, ip netip.Addr) (*Export, error) {
	account, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	activity, err := s.Privacy.Activity(ctx, userID, s.Clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.Audit.Record(ctx, audit.Entry{
		ActorID: &userID, Action: audit.ActionAccountExported, EntityType: "user", EntityID: &userID, IP: ip,
	}); err != nil {
		return nil, err
	}
	return &Export{Account: account, Activity: activity}, nil
}

// DeleteOwnAccount supprime (anonymise) le compte de son titulaire, qui redonne son mot de passe et, s'il l'a
// activée, un code de double authentification. Un mail de confirmation part vers l'ancienne adresse.
func (s *Service) DeleteOwnAccount(ctx context.Context, userID uuid.UUID, password, code string, ip netip.Addr) error {
	if err := s.Identity.CheckPassword(ctx, userID, password); err != nil {
		return err
	}
	if err := s.Identity.CheckCodeIfEnrolled(ctx, userID, code, ip); err != nil {
		return err
	}
	owner, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.erase(ctx, owner, userID, ip); err != nil {
		return err
	}
	if err := s.Mailer.SendAccountDeleted(ctx, owner.Email, owner.FirstName); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de suppression du compte impossible", "user_id", userID, "error", err)
	}
	return nil
}

// AnonymizeByModerator anonymise le compte d'un citoyen ou d'un super-héros à la demande d'un modérateur (une
// demande d'effacement reçue par le formulaire de contact, par exemple). Aucun mail n'est envoyé.
func (s *Service) AnonymizeByModerator(ctx context.Context, moderatorID, targetID uuid.UUID, ip netip.Addr) error {
	if moderatorID == targetID {
		return ErrNotErasable
	}
	target, err := s.Users.FindByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == user.RoleModerator {
		return ErrNotErasable
	}
	return s.erase(ctx, target, moderatorID, ip)
}

// erase anonymise un compte et efface ce qui ne servait qu'à lui, en une transaction, puis supprime sa photo.
func (s *Service) erase(ctx context.Context, target *user.User, actorID uuid.UUID, ip netip.Addr) error {
	if target.DeletedAt != nil {
		return ErrAlreadyErased
	}
	// Le mot de passe d'un compte supprimé est un secret aléatoire aussitôt oublié : personne ne le connaît.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("account : tirage du secret : %w", err)
	}
	hash, err := s.Hasher.Hash(ctx, base64.RawURLEncoding.EncodeToString(secret))
	if err != nil {
		return fmt.Errorf("account : empreinte du secret : %w", err)
	}

	var avatarPath string
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		previous, err := s.Users.Anonymize(ctx, target.ID, hash, s.Clock.Now())
		if errors.Is(err, user.ErrNotFound) {
			return ErrAlreadyErased // anonymisé entre-temps par une autre requête
		}
		if err != nil {
			return err
		}
		avatarPath = previous
		if err := s.Privacy.EraseRelated(ctx, target.ID, target.Email); err != nil {
			return err
		}
		// Le journal ne garde que l'identifiant : rien qui permette de retrouver la personne.
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &actorID, Action: audit.ActionAccountErased, EntityType: "user", EntityID: &target.ID,
			Details: map[string]any{"byOwner": actorID == target.ID}, IP: ip,
		})
	})
	if err != nil {
		return err
	}
	if avatarPath != "" {
		s.discard(ctx, avatarPath)
	}
	return nil
}
