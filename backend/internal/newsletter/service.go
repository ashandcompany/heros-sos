package newsletter

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/signedlink"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// Mailer envoie le mail de confirmation (implémenté par mail.Service).
type Mailer interface {
	SendNewsletterConfirmation(ctx context.Context, to, link string) error
	SendNewsletter(ctx context.Context, to, subject, bodyHTML, unsubscribeLink string) error
}

// LinkSigner signe et vérifie les liens de confirmation (implémenté par signedlink.Signer).
type LinkSigner interface {
	Sign(purpose signedlink.Purpose, subject uuid.UUID, expiresAt time.Time) string
	Verify(token string, purpose signedlink.Purpose, now time.Time) (uuid.UUID, error)
}

// ServiceDeps regroupe les dépendances du service.
type ServiceDeps struct {
	Repo   Repository
	Issues IssueRepository
	Audit  audit.Recorder
	Tx     database.Transactor
	Links  LinkSigner
	Mailer Mailer
	App    config.App
	// ConfirmTTL est la durée de validité du lien de confirmation.
	ConfirmTTL time.Duration
	Clock      clock.Clock
	Logger     *slog.Logger
}

// Service porte les règles des abonnements. Il ne connaît pas HTTP.
type Service struct {
	ServiceDeps
}

// NewService crée le service.
func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

// Subscribe enregistre la demande d'un visiteur et lui envoie le lien qui confirme son adresse : sans
// ce clic, aucune newsletter ne lui sera envoyée (double opt-in). La réponse ne dépend jamais de
// l'état de l'adresse (inconnue, en attente ou déjà abonnée) : elle ne révèle rien, et une adresse
// déjà abonnée ne reçoit pas de mail de plus. Seul le format de l'adresse peut produire une erreur.
//
// Un échec d'envoi du mail est journalisé sans échouer la demande : le visiteur peut la renouveler.
func (s *Service) Subscribe(ctx context.Context, rawEmail string) error {
	email, code := validation.Email(rawEmail)
	if code != "" {
		return &validation.Error{Fields: map[string]string{"email": code}}
	}

	now := s.Clock.Now()
	id, confirmationNeeded, err := s.Repo.Request(ctx, ids.New(), email, now)
	if err != nil || !confirmationNeeded {
		return err
	}

	token := s.Links.Sign(signedlink.PurposeNewsletterConfirm, id, now.Add(s.ConfirmTTL))
	if err := s.Mailer.SendNewsletterConfirmation(ctx, email, s.App.NewsletterConfirmLink(token)); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de confirmation de la newsletter impossible", "subscription_id", id, "error", err)
	}
	return nil
}

// Confirm valide l'abonnement désigné par un lien reçu par mail. Cliquer deux fois sur le lien reste un
// succès ; un lien falsifié, expiré ou devenu sans objet (l'abonné s'est retiré entre-temps) retourne
// ErrLinkInvalid.
func (s *Service) Confirm(ctx context.Context, token string) error {
	now := s.Clock.Now()
	id, err := s.Links.Verify(token, signedlink.PurposeNewsletterConfirm, now)
	if errors.Is(err, signedlink.ErrInvalid) {
		return ErrLinkInvalid
	}
	if err != nil {
		return err
	}
	return s.Repo.Confirm(ctx, id, now)
}

// UnsubscribeLink construit le lien de désinscription d'un abonné, à placer dans chaque newsletter qui
// lui est envoyée (Service.SendNewsletter de mail le rend obligatoire). Il est signé sans date limite :
// le lien d'une ancienne newsletter doit rester utilisable, et ne porte que l'identifiant de
// l'abonnement, jamais l'adresse.
func (s *Service) UnsubscribeLink(subscriptionID uuid.UUID) string {
	token := s.Links.Sign(signedlink.PurposeNewsletterUnsubscribe, subscriptionID, time.Time{})
	return s.App.NewsletterUnsubscribeLink(token)
}

// Unsubscribe désabonne l'abonné désigné par un lien reçu dans une newsletter. Cliquer deux fois sur le
// lien reste un succès ; un lien falsifié, ou d'une autre intention, retourne ErrLinkInvalid. Se
// désabonner ne demande aucune connexion : le lien suffit, comme la loi l'exige.
func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	id, err := s.Links.Verify(token, signedlink.PurposeNewsletterUnsubscribe, s.Clock.Now())
	if errors.Is(err, signedlink.ErrInvalid) {
		return ErrLinkInvalid
	}
	if err != nil {
		return err
	}
	return s.Repo.Unsubscribe(ctx, id, s.Clock.Now())
}

// IsSubscribed indique si le titulaire d'un compte reçoit la newsletter.
func (s *Service) IsSubscribed(ctx context.Context, userID uuid.UUID, email string) (bool, error) {
	return s.Repo.IsSubscribed(ctx, userID, email)
}

// SetPreference abonne ou désabonne le titulaire d'un compte, sans confirmation par mail : son
// adresse a déjà été vérifiée à l'activation du compte.
func (s *Service) SetPreference(ctx context.Context, userID uuid.UUID, email string, subscribed bool) error {
	return s.Repo.SetAccountPreference(ctx, ids.New(), userID, email, subscribed, s.Clock.Now())
}

// FollowEmailChange aligne l'abonnement d'un compte sur sa nouvelle adresse : sans cela, la newsletter
// continuerait d'être envoyée à l'ancienne, que son titulaire ne contrôle peut-être plus.
func (s *Service) FollowEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error {
	return s.Repo.FollowEmail(ctx, userID, newEmail)
}

// PurgeStale supprime les demandes jamais confirmées depuis plus de retention : l'adresse d'une
// personne qui n'a rien confirmé n'a pas à être conservée.
func (s *Service) PurgeStale(ctx context.Context, retention time.Duration) (int64, error) {
	return s.Repo.DeleteStalePending(ctx, s.Clock.Now().Add(-retention))
}
