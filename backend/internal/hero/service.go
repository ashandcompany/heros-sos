package hero

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Service porte les règles du statut de super-héros.
type Service struct {
	repo  Repository
	users user.Repository
	audit audit.Recorder
	tx    database.Transactor
	clock clock.Clock
}

// NewService crée le service.
func NewService(repo Repository, users user.Repository, recorder audit.Recorder, tx database.Transactor, clk clock.Clock) *Service {
	return &Service{repo: repo, users: users, audit: recorder, tx: tx, clock: clk}
}

// RequestInput est la saisie du formulaire « Devenir super-héros ».
type RequestInput struct {
	Alias   string
	Bio     string
	Message string
}

// Applicant est une demande accompagnée de l'identité du demandeur, pour les modérateurs
// (l'identité civile d'un héros n'est jamais publique).
type Applicant struct {
	Profile
	Email     string
	FirstName string
	LastName  string
}

// Request dépose la demande d'un citoyen actif. Elle part « en attente » jusqu'à la décision d'un
// modérateur ; une demande refusée peut être redéposée.
func (s *Service) Request(ctx context.Context, userID uuid.UUID, in RequestInput, ip netip.Addr) error {
	var fields validation.Collector
	alias, code := validation.Alias(in.Alias)
	fields.Add("alias", code)
	bio, code := validation.OptionalText(in.Bio, validation.MaxBioLength)
	fields.Add("bio", code)
	message, code := validation.OptionalText(in.Message, validation.MaxRequestMessageLength)
	fields.Add("requestMessage", code)
	if err := fields.Err(); err != nil {
		return err
	}

	applicant, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	switch {
	case applicant.Role == user.RoleHero:
		return ErrAlreadyHero
	case applicant.Role != user.RoleCitizen || applicant.Status != user.StatusActive:
		return ErrNotEligible
	}

	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Request(ctx, userID, alias, bio, message, s.clock.Now()); err != nil {
			return err
		}
		return s.audit.Record(ctx, audit.Entry{
			ActorID: &userID, Action: audit.ActionHeroRequested, EntityType: "user", EntityID: &userID, IP: ip,
		})
	})
}

// Mine retourne la demande de l'utilisateur (ErrNotFound s'il n'en a pas).
func (s *Service) Mine(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.Find(ctx, userID)
}

// List retourne les demandes d'un statut, les plus anciennes d'abord, avec l'identité des
// demandeurs, lue en une seule requête pour toute la page.
func (s *Service) List(ctx context.Context, status VerificationStatus, limit, offset int) ([]Applicant, int64, error) {
	profiles, total, err := s.repo.List(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, len(profiles))
	for i := range profiles {
		ids[i] = profiles[i].UserID
	}
	accounts, err := s.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("hero : demandeurs : %w", err)
	}
	applicants := make([]Applicant, 0, len(profiles))
	for _, profile := range profiles {
		account, ok := accounts[profile.UserID]
		if !ok {
			return nil, 0, fmt.Errorf("hero : demandeur %s : %w", profile.UserID, user.ErrNotFound)
		}
		applicants = append(applicants, Applicant{
			Profile: profile, Email: account.Email, FirstName: account.FirstName, LastName: account.LastName,
		})
	}
	return applicants, total, nil
}

// ListDirectory retourne l'annuaire des super-héros pour les modérateurs, par pseudo, avec leur identité
// civile, lue en une seule requête pour toute la page.
func (s *Service) ListDirectory(ctx context.Context, limit, offset int) ([]DirectoryView, int64, error) {
	entries, total, err := s.repo.ListDirectory(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, len(entries))
	for i := range entries {
		ids[i] = entries[i].UserID
	}
	accounts, err := s.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("hero : comptes des héros : %w", err)
	}
	views := make([]DirectoryView, len(entries))
	for i, entry := range entries {
		account, ok := accounts[entry.UserID]
		if !ok {
			return nil, 0, fmt.Errorf("hero : compte du héros %s : %w", entry.UserID, user.ErrNotFound)
		}
		views[i] = DirectoryView{
			DirectoryEntry: entry, Civility: account.Civility, FirstName: account.FirstName,
			LastName: account.LastName, BirthDate: account.BirthDate, Phone: account.Phone, Address: account.Address,
		}
	}
	return views, total, nil
}

// ListPublic retourne les fiches des héros publics selon l'ordre demandé.
func (s *Service) ListPublic(ctx context.Context, sort PublicSort, limit, offset int) ([]PublicProfile, int64, error) {
	return s.repo.ListPublic(ctx, sort, limit, offset)
}

// FindPublic retourne une fiche uniquement si le compte est actif et le profil approuvé.
func (s *Service) FindPublic(ctx context.Context, userID uuid.UUID) (*PublicProfile, error) {
	return s.repo.FindPublic(ctx, userID)
}

// Approve accepte une demande : le demandeur devient super-héros. Le changement de statut, celui
// du rôle et la trace d'audit forment une seule transaction.
func (s *Service) Approve(ctx context.Context, moderatorID, userID uuid.UUID, ip netip.Addr) error {
	return s.review(ctx, moderatorID, userID, StatusApproved, "", audit.ActionHeroApproved, ip)
}

// Reject refuse une demande, avec un motif facultatif transmis au demandeur.
func (s *Service) Reject(ctx context.Context, moderatorID, userID uuid.UUID, reason string, ip netip.Addr) error {
	reason, code := validation.OptionalText(reason, validation.MaxRejectionReasonLength)
	if code != "" {
		return &validation.Error{Fields: map[string]string{"reason": code}}
	}
	return s.review(ctx, moderatorID, userID, StatusRejected, reason, audit.ActionHeroRejected, ip)
}

func (s *Service) review(ctx context.Context, moderatorID, userID uuid.UUID, to VerificationStatus, reason, action string, ip netip.Addr) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		now := s.clock.Now()
		changed, err := s.repo.Review(ctx, userID, moderatorID, to, reason, now)
		if err != nil {
			return err
		}
		if !changed {
			if _, err := s.repo.Find(ctx, userID); err != nil {
				return err // ErrNotFound : aucune demande pour cet utilisateur
			}
			return ErrNotPending
		}
		if to == StatusApproved {
			if err := s.users.SetRole(ctx, userID, user.RoleHero, now); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, audit.Entry{
			ActorID: &moderatorID, Action: action, EntityType: "user", EntityID: &userID, IP: ip,
		})
	})
}
