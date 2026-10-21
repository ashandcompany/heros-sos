package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/avatar"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/signedlink"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// CodeTooLarge est le code de champ d'une photo dont les dimensions dépassent la limite.
const CodeTooLarge = "too_large"

// HeroProfiles donne le profil de super-héros d'un compte (implémenté par hero.Repository).
type HeroProfiles interface {
	Find(ctx context.Context, userID uuid.UUID) (*hero.Profile, error)
}

// Newsletter gère l'abonnement d'un titulaire de compte (implémenté par newsletter.Service).
type Newsletter interface {
	IsSubscribed(ctx context.Context, userID uuid.UUID, email string) (bool, error)
	SetPreference(ctx context.Context, userID uuid.UUID, email string, subscribed bool) error
	FollowEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error
}

// Mailer envoie les mails d'un changement d'adresse (implémenté par mail.Service).
type Mailer interface {
	SendEmailChangeConfirmation(ctx context.Context, to, firstName, link string) error
	SendEmailChangeApproved(ctx context.Context, to, firstName string) error
	SendEmailChangeNotice(ctx context.Context, to, firstName, newEmail string) error
	SendEmailChangeRejected(ctx context.Context, to, firstName string) error
}

// LinkSigner signe et vérifie le lien de confirmation de la nouvelle adresse (implémenté par
// signedlink.Signer).
type LinkSigner interface {
	Sign(purpose signedlink.Purpose, subject uuid.UUID, expiresAt time.Time) string
	Verify(token string, purpose signedlink.Purpose, now time.Time) (uuid.UUID, error)
}

// AvatarStore conserve les photos de profil (implémenté par avatar.DiskStore).
type AvatarStore interface {
	Save(data []byte) (name string, err error)
	Delete(name string) error
}

// ServiceDeps regroupe les dépendances du service.
type ServiceDeps struct {
	Users      user.Repository
	Heroes     HeroProfiles
	Changes    EmailChangeRepository
	Newsletter Newsletter
	Avatars    AvatarStore
	Mailer     Mailer
	Links      LinkSigner
	App        config.App
	// EmailChangeConfirmTTL est la validité du lien de confirmation de la nouvelle adresse.
	EmailChangeConfirmTTL time.Duration
	Audit                 audit.Recorder
	Tx                    database.Transactor
	Clock                 clock.Clock
	// AvatarMaxDimension est le côté maximal, en pixels, d'une photo acceptée.
	AvatarMaxDimension int
	Logger             *slog.Logger
}

// Service porte les règles du profil. Il ne connaît pas HTTP.
type Service struct {
	ServiceDeps
}

// NewService crée le service.
func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

// Get retourne le profil complet d'un utilisateur.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*Account, error) {
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	account := &Account{User: u}

	pending, err := s.Changes.FindPending(ctx, userID)
	switch {
	case err == nil:
		account.PendingEmail, account.PendingEmailConfirmed = pending.NewEmail, pending.ConfirmedAt != nil
	case !errors.Is(err, ErrChangeNotFound):
		return nil, err
	}

	if account.NewsletterSubscribed, err = s.Newsletter.IsSubscribed(ctx, userID, u.Email); err != nil {
		return nil, err
	}

	if u.Role == user.RoleHero {
		profile, err := s.Heroes.Find(ctx, userID)
		switch {
		case err == nil:
			account.Hero = profile
		case !errors.Is(err, hero.ErrNotFound):
			return nil, err
		}
	}
	return account, nil
}

// UpdateContact remplace le téléphone et/ou l'adresse postale, puis retourne le profil à jour. Les
// règles sont celles de l'inscription ; toutes les erreurs de saisie sont retournées ensemble
// (*validation.Error).
func (s *Service) UpdateContact(ctx context.Context, userID uuid.UUID, in ContactInput) (*Account, error) {
	if in.Phone == nil && in.Address == nil {
		return nil, &validation.Error{Fields: map[string]string{
			"mobilePhone": validation.CodeRequired, "address": validation.CodeRequired,
		}}
	}

	var (
		fields validation.Collector
		update user.ContactUpdate
	)
	if in.Phone != nil {
		phone, code := validation.Phone(*in.Phone)
		fields.Add("mobilePhone", code)
		update.Phone = &phone
	}
	if given := in.Address; given != nil {
		var complement string
		if given.Complement != nil {
			complement = *given.Complement
		}
		address := fields.Address("address", validation.Address{
			Street: given.Street, Complement: complement, PostalCode: given.PostalCode, City: given.City, Country: given.Country,
		})
		update.Address = &user.AddressUpdate{
			Street: address.Street, PostalCode: address.PostalCode, City: address.City, Country: address.Country,
		}
		if given.Complement != nil {
			update.Address.Complement = &address.Complement
		}
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}

	if err := s.Users.UpdateContact(ctx, userID, update, s.Clock.Now()); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

// SetAvatar remplace la photo de profil et retourne le nom du nouveau fichier. L'image est contrôlée et
// ré-encodée avant d'être écrite (voir le paquet avatar). Le fichier est écrit avant la base : si
// l'enregistrement échoue il est supprimé, et l'ancienne photo n'est supprimée qu'une fois la
// nouvelle enregistrée. Une panne entre ces deux étapes laisse au pire un fichier orphelin, jamais un
// compte sans photo.
func (s *Service) SetAvatar(ctx context.Context, userID uuid.UUID, data []byte) (string, error) {
	normalized, err := avatar.Normalize(data, s.AvatarMaxDimension)
	switch {
	case errors.Is(err, avatar.ErrTooLarge):
		return "", &validation.Error{Fields: map[string]string{"avatar": CodeTooLarge}}
	case errors.Is(err, avatar.ErrInvalidImage):
		return "", &validation.Error{Fields: map[string]string{"avatar": validation.CodeInvalid}}
	case err != nil:
		return "", err
	}

	name, err := s.Avatars.Save(normalized)
	if err != nil {
		return "", err
	}
	previous, err := s.Users.SetAvatarPath(ctx, userID, name, s.Clock.Now())
	if err != nil {
		s.discard(ctx, name)
		return "", err
	}
	if previous != "" {
		s.discard(ctx, previous)
	}
	return name, nil
}

// discard supprime un fichier devenu inutile. Un échec est journalisé sans faire échouer l'opération :
// il ne reste qu'un fichier orphelin, que personne ne référence.
func (s *Service) discard(ctx context.Context, name string) {
	if err := s.Avatars.Delete(name); err != nil {
		s.Logger.WarnContext(ctx, "suppression d'une photo de profil impossible", "file", name, "error", err)
	}
}

// RequestEmailChange dépose la demande de changement d'adresse d'un compte. Elle remplace la demande
// précédente si elle n'était pas traitée, et ne prend effet qu'en deux temps : le titulaire de la nouvelle
// adresse la confirme par le lien qui lui est envoyé, puis un modérateur l'accepte. Le mail est confié à
// la file d'envoi une fois la demande enregistrée ; un échec est journalisé sans l'annuler (la personne
// peut renouveler sa demande).
func (s *Service) RequestEmailChange(ctx context.Context, userID uuid.UUID, rawEmail string, ip netip.Addr) (*EmailChange, error) {
	email, code := validation.Email(rawEmail)
	if code != "" {
		return nil, &validation.Error{Fields: map[string]string{"email": code}}
	}

	// Un refus anticipé pour confort : le contrôle qui fait foi est l'unicité en base, appliquée à la
	// validation par le modérateur (une adresse peut être prise entre-temps).
	existing, err := s.Users.FindByEmail(ctx, email)
	switch {
	case err == nil && existing.ID == userID:
		return nil, ErrEmailUnchanged
	case err == nil:
		return nil, user.ErrEmailTaken
	case !errors.Is(err, user.ErrNotFound):
		return nil, err
	}
	owner, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	change := &EmailChange{ID: ids.New(), UserID: userID, NewEmail: email, Status: ChangePending, CreatedAt: s.Clock.Now()}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Changes.Replace(ctx, change.ID, userID, email, change.CreatedAt); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &userID, Action: audit.ActionEmailChangeRequested, EntityType: "user", EntityID: &userID,
			Details: map[string]any{"requestId": change.ID.String()}, IP: ip,
		})
	})
	if err != nil {
		return nil, err
	}

	// Le lien désigne la demande : remplacée, elle change d'identifiant et l'ancien lien cesse de valoir.
	token := s.Links.Sign(signedlink.PurposeEmailChangeConfirm, change.ID, change.CreatedAt.Add(s.EmailChangeConfirmTTL))
	if err := s.Mailer.SendEmailChangeConfirmation(ctx, email, owner.FirstName, s.App.EmailChangeConfirmLink(token)); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de confirmation de la nouvelle adresse impossible", "request_id", change.ID, "error", err)
	}
	return change, nil
}

// ConfirmEmailChange confirme la nouvelle adresse désignée par un lien reçu par mail : son titulaire a
// bien accès à la boîte. Cliquer deux fois sur le lien reste un succès ; un lien falsifié, expiré, ou
// devenu sans objet (demande remplacée ou refusée) retourne ErrLinkInvalid. La confirmation ne change pas
// l'adresse du compte : elle rend la demande acceptable par un modérateur.
func (s *Service) ConfirmEmailChange(ctx context.Context, token string, ip netip.Addr) error {
	now := s.Clock.Now()
	requestID, err := s.Links.Verify(token, signedlink.PurposeEmailChangeConfirm, now)
	if errors.Is(err, signedlink.ErrInvalid) {
		return ErrLinkInvalid
	}
	if err != nil {
		return err
	}

	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		change, newlyConfirmed, err := s.Changes.Confirm(ctx, requestID, now)
		if errors.Is(err, ErrChangeNotFound) || errors.Is(err, ErrChangeNotPending) {
			return ErrLinkInvalid
		}
		if err != nil || !newlyConfirmed {
			return err
		}
		return s.record(ctx, change.UserID, change, audit.ActionEmailChangeConfirmed, ip)
	})
}

// PurgeStaleEmailChanges supprime les demandes jamais confirmées depuis plus de retention : leur lien a
// expiré, elles encombreraient la file des modérateurs sans pouvoir aboutir.
func (s *Service) PurgeStaleEmailChanges(ctx context.Context, retention time.Duration) (int64, error) {
	return s.Changes.DeleteStaleUnconfirmed(ctx, s.Clock.Now().Add(-retention))
}

// ListEmailChanges retourne les demandes d'un statut (toutes si vide), les plus anciennes d'abord, avec
// l'identité des demandeurs, lue en une seule requête pour toute la page.
func (s *Service) ListEmailChanges(ctx context.Context, status ChangeStatus, limit, offset int) ([]EmailChangeView, int64, error) {
	changes, total, err := s.Changes.List(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, len(changes))
	for i := range changes {
		ids[i] = changes[i].UserID
	}
	accounts, err := s.Users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("account : demandeurs : %w", err)
	}
	views := make([]EmailChangeView, len(changes))
	for i, change := range changes {
		account, ok := accounts[change.UserID]
		if !ok {
			return nil, 0, fmt.Errorf("account : demandeur %s : %w", change.UserID, user.ErrNotFound)
		}
		views[i] = EmailChangeView{
			EmailChange: change, CurrentEmail: account.Email, FirstName: account.FirstName, LastName: account.LastName,
		}
	}
	return views, total, nil
}

// ApproveEmailChange valide une demande : l'adresse de connexion du compte devient la nouvelle. Seule une
// demande dont l'adresse est confirmée peut l'être (ErrChangeNotConfirmed sinon). Le changement de statut,
// celui de l'adresse, celui de l'abonnement à la newsletter et la trace d'audit forment une seule
// transaction. Si l'adresse a été prise entre-temps, rien n'est modifié et user.ErrEmailTaken est
// retourné : la demande reste en attente, à refuser.
//
// Une fois la décision enregistrée, la nouvelle adresse est prévenue, et l'ancienne aussi : si le
// changement n'est pas voulu, c'est elle qui peut le plus vite donner l'alerte.
func (s *Service) ApproveEmailChange(ctx context.Context, moderatorID, requestID uuid.UUID, ip netip.Addr) error {
	var owner *user.User
	var change *EmailChange
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		now := s.Clock.Now()
		var err error
		if change, err = s.Changes.Review(ctx, requestID, moderatorID, ChangeApproved, now); err != nil {
			return err
		}
		// Lu avant le changement : on y garde l'ancienne adresse, qu'il faut prévenir.
		if owner, err = s.Users.FindByID(ctx, change.UserID); err != nil {
			return err
		}
		if err := s.Users.ChangeEmail(ctx, change.UserID, change.NewEmail, now); err != nil {
			return err
		}
		if err := s.Newsletter.FollowEmailChange(ctx, change.UserID, change.NewEmail); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, change, audit.ActionEmailChangeApproved, ip)
	})
	if err != nil {
		return err
	}

	if err := s.Mailer.SendEmailChangeApproved(ctx, change.NewEmail, owner.FirstName); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail d'acceptation impossible", "request_id", change.ID, "error", err)
	}
	if err := s.Mailer.SendEmailChangeNotice(ctx, owner.Email, owner.FirstName, change.NewEmail); err != nil {
		s.Logger.ErrorContext(ctx, "envoi de l'avertissement à l'ancienne adresse impossible", "request_id", change.ID, "error", err)
	}
	return nil
}

// RejectEmailChange refuse une demande : l'adresse du compte ne change pas, et son titulaire en est
// prévenu à son adresse actuelle.
func (s *Service) RejectEmailChange(ctx context.Context, moderatorID, requestID uuid.UUID, ip netip.Addr) error {
	var owner *user.User
	var change *EmailChange
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if change, err = s.Changes.Review(ctx, requestID, moderatorID, ChangeRejected, s.Clock.Now()); err != nil {
			return err
		}
		if owner, err = s.Users.FindByID(ctx, change.UserID); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, change, audit.ActionEmailChangeRejected, ip)
	})
	if err != nil {
		return err
	}

	if err := s.Mailer.SendEmailChangeRejected(ctx, owner.Email, owner.FirstName); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de refus impossible", "request_id", change.ID, "error", err)
	}
	return nil
}

// SetNewsletter abonne ou désabonne un titulaire de compte, sans mail de confirmation : son adresse a
// été vérifiée à l'activation.
func (s *Service) SetNewsletter(ctx context.Context, userID uuid.UUID, subscribed bool) error {
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	return s.Newsletter.SetPreference(ctx, userID, u.Email, subscribed)
}

// record journalise une décision de modération. Les adresses email, données personnelles, n'y figurent
// pas : l'identifiant de la demande suffit à la retrouver.
func (s *Service) record(ctx context.Context, moderatorID uuid.UUID, change *EmailChange, action string, ip netip.Addr) error {
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &moderatorID, Action: action, EntityType: "user", EntityID: &change.UserID,
		Details: map[string]any{"requestId": change.ID.String()}, IP: ip,
	})
}
