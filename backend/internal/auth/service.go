package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// AccountMailer envoie les mails liés au compte (implémenté par mail.Service).
type AccountMailer interface {
	SendActivation(ctx context.Context, to, firstName, link string) error
	SendPasswordReset(ctx context.Context, to, firstName, link string) error
	SendPasswordChanged(ctx context.Context, to, firstName string) error
}

// ServiceDeps regroupe les dépendances du service d'authentification.
type ServiceDeps struct {
	Users    user.Repository
	Tokens   *TokenStore
	MFA      *MFAStore
	Sessions *SessionService
	Hasher   *PasswordHasher
	Mailer   AccountMailer
	Audit    audit.Recorder
	Tx       database.Transactor
	Clock    clock.Clock
	Security config.Security
	App      config.App
	Logger   *slog.Logger
}

// Service porte les règles d'authentification : inscription, activation, connexion,
// déconnexion, mot de passe oublié. Il ne connaît pas HTTP.
type Service struct {
	ServiceDeps
}

// NewService crée le service d'authentification.
func NewService(deps ServiceDeps) *Service {
	return &Service{ServiceDeps: deps}
}

// RegisterInput est la saisie du formulaire d'inscription (valeurs brutes, non validées).
type RegisterInput struct {
	Civility  string
	FirstName string
	LastName  string
	BirthDate string
	Email     string
	Phone     string
	Address   user.Address
	Password  string
}

// Register crée un compte citoyen en attente d'activation et envoie le lien d'activation.
// Toutes les erreurs de saisie sont retournées ensemble (*validation.Error). Une adresse déjà
// utilisée retourne user.ErrEmailTaken : l'unicité est garantie par la base, sans fenêtre de
// concurrence entre une vérification et l'insertion.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*user.User, error) {
	now := s.Clock.Now()
	u, err := s.validateRegistration(in, now)
	if err != nil {
		return nil, err
	}

	if u.PasswordHash, err = s.Hasher.Hash(ctx, in.Password); err != nil {
		return nil, fmt.Errorf("auth : hachage du mot de passe : %w", err)
	}
	u.ID, u.Role, u.Status = ids.New(), user.RoleCitizen, user.StatusPendingActivation

	rawToken, err := s.createUserWithToken(ctx, u, now)
	if err != nil {
		return nil, err
	}
	s.sendActivation(ctx, u, rawToken)
	return u, nil
}

func (s *Service) validateRegistration(in RegisterInput, now time.Time) (*user.User, error) {
	var fields validation.Collector
	u := &user.User{}

	fields.Add("civility", validation.Civility(in.Civility))
	u.Civility = in.Civility

	var code string
	u.FirstName, code = validation.Name(in.FirstName)
	fields.Add("firstName", code)
	u.LastName, code = validation.Name(in.LastName)
	fields.Add("lastName", code)
	u.BirthDate, code = validation.BirthDate(in.BirthDate, now)
	fields.Add("birthDate", code)
	u.Email, code = validation.Email(in.Email)
	fields.Add("email", code)
	u.Phone, code = validation.Phone(in.Phone)
	fields.Add("mobilePhone", code)

	u.Address = user.Address(fields.Address("address", validation.Address(in.Address)))

	fields.Add("password", validation.Password(in.Password, s.passwordPolicy()))
	return u, fields.Err()
}

func (s *Service) passwordPolicy() validation.PasswordPolicy {
	return validation.PasswordPolicy{MinLength: s.Security.Password.MinLength, MaxLength: s.Security.Password.MaxLength}
}

// createUserWithToken enregistre le compte et son jeton d'activation dans une transaction :
// jamais de compte sans moyen de l'activer.
func (s *Service) createUserWithToken(ctx context.Context, u *user.User, now time.Time) (rawToken string, err error) {
	rawToken, hash, err := newToken()
	if err != nil {
		return "", err
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Users.Create(ctx, u); err != nil {
			return err
		}
		return s.Tokens.Create(ctx, u.ID, TokenActivation, hash, now.Add(s.Security.Tokens.ActivationTTL), now)
	})
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

// sendActivation confie le mail à la file d'envoi. Un échec n'annule pas l'inscription : il est
// journalisé et l'utilisateur peut redemander le lien (ResendActivation).
func (s *Service) sendActivation(ctx context.Context, u *user.User, rawToken string) {
	link := s.App.ActivationLink(rawToken)
	if err := s.Mailer.SendActivation(ctx, u.Email, u.FirstName, link); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail d'activation impossible", "user_id", u.ID, "error", err)
	}
}

// Activate active le compte correspondant à un jeton. Le jeton n'est consommé que si
// l'activation réussit : les deux opérations sont atomiques.
func (s *Service) Activate(ctx context.Context, rawToken string) error {
	if !isWellFormedToken(rawToken) {
		return ErrTokenInvalid
	}
	now := s.Clock.Now()
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		userID, err := s.Tokens.Consume(ctx, TokenActivation, hashToken(rawToken), now)
		if err != nil {
			return err
		}
		changed, err := s.Users.Activate(ctx, userID, now)
		if err != nil {
			return err
		}
		if !changed {
			return ErrTokenInvalid
		}
		return nil
	})
}

// ResendActivation renvoie un lien d'activation à un compte en attente. Il ne révèle jamais si
// l'adresse correspond à un compte : tous les cas retournent nil, sauf une adresse invalide.
func (s *Service) ResendActivation(ctx context.Context, email string) error {
	normalized, code := validation.Email(email)
	if code != "" {
		return &validation.Error{Fields: map[string]string{"email": code}}
	}
	u, err := s.Users.FindByEmail(ctx, normalized)
	if errors.Is(err, user.ErrNotFound) || (err == nil && u.Status != user.StatusPendingActivation) {
		return nil
	}
	if err != nil {
		return err
	}

	now := s.Clock.Now()
	rawToken, hash, err := newToken()
	if err != nil {
		return err
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Tokens.DeleteUnused(ctx, u.ID, TokenActivation); err != nil {
			return err
		}
		return s.Tokens.Create(ctx, u.ID, TokenActivation, hash, now.Add(s.Security.Tokens.ActivationTTL), now)
	})
	if err != nil {
		return err
	}
	s.sendActivation(ctx, u, rawToken)
	return nil
}

// LoginInput est la saisie du formulaire de connexion.
type LoginInput struct {
	// Identifier est l'adresse email, ou le pseudo d'un super-héros.
	Identifier string
	Password   string
	RememberMe bool
	IP         netip.Addr
	UserAgent  string
}

// LoginResult est une connexion réussie. Quand MFA n'est pas « verified » ni « not_enrolled », la session
// est limitée : elle n'ouvre que la saisie du code (ou l'activation de la 2FA) et la déconnexion.
type LoginResult struct {
	User    *user.User
	Session IssuedSession
	MFA     MFAStatus
}

// Login authentifie un utilisateur et ouvre une session.
//
// Un identifiant inconnu, mal formé ou un mauvais mot de passe produisent la même erreur
// (ErrInvalidCredentials) après le même travail (une vérification argon2id), pour ne rien
// révéler des comptes existants. Les états « non activé » et « désactivé » ne sont annoncés
// qu'avec le bon mot de passe.
func (s *Service) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
	now := s.Clock.Now()
	u, check, err := s.verifyCredentials(ctx, in, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err := statusError(u.Status); err != nil {
		return LoginResult{}, err
	}
	if check.NeedsRehash {
		s.upgradePasswordHash(ctx, u, in.Password, now)
	}
	return s.openSession(ctx, u, in, now)
}

// verifyCredentials retrouve le compte et contrôle le mot de passe : identifiant inconnu, compte verrouillé,
// mot de passe faux (l'échec est compté, et verrouille le compte après trop d'essais). Il ne regarde pas l'état
// du compte, qui ne s'annonce qu'avec le bon mot de passe.
func (s *Service) verifyCredentials(ctx context.Context, in LoginInput, now time.Time) (*user.User, PasswordCheck, error) {
	identifier, code := validation.Identifier(in.Identifier)
	if code != "" || in.Password == "" {
		return nil, PasswordCheck{}, s.rejectUnknown(ctx, in.Password)
	}

	u, err := s.findByIdentifier(ctx, identifier)
	if errors.Is(err, user.ErrNotFound) {
		return nil, PasswordCheck{}, s.rejectUnknown(ctx, in.Password)
	}
	if err != nil {
		return nil, PasswordCheck{}, err
	}
	if u.IsLocked(now) {
		return nil, PasswordCheck{}, &LockedError{Until: *u.LockedUntil}
	}

	check, err := s.Hasher.Verify(ctx, u.PasswordHash, in.Password)
	if err != nil {
		return nil, PasswordCheck{}, fmt.Errorf("auth : vérification du mot de passe : %w", err)
	}
	if !check.Match {
		if _, err := s.Users.RecordLoginFailure(ctx, u.ID, now, s.Security.Lockout); err != nil {
			return nil, PasswordCheck{}, err
		}
		return nil, PasswordCheck{}, ErrInvalidCredentials
	}
	return u, check, nil
}

// statusError est l'erreur d'un compte qui ne peut pas se connecter, ou nil s'il est actif.
func statusError(status user.Status) error {
	switch status {
	case user.StatusPendingActivation:
		return ErrAccountNotActivated
	case user.StatusDeactivated:
		return ErrAccountDeactivated
	case user.StatusActive:
	}
	return nil
}

// openSession ouvre la session d'un compte dont le mot de passe est bon. Elle n'est complète que sans second
// facteur à fournir : ni activé, ni obligatoire pour ce rôle.
func (s *Service) openSession(ctx context.Context, u *user.User, in LoginInput, now time.Time) (LoginResult, error) {
	enrolled, err := s.MFA.IsEnrolled(ctx, u.ID)
	if err != nil {
		return LoginResult{}, err
	}
	// Le mot de passe seul ne remet pas à zéro le compteur d'échecs d'un compte qui a la 2FA : c'est le code
	// qui le fera (MFAService.Verify). Sans cela, celui qui connaît le mot de passe se reconnecterait entre
	// deux séries de devinettes du code.
	if !enrolled {
		if err := s.Users.RecordLoginSuccess(ctx, u.ID, now); err != nil {
			return LoginResult{}, err
		}
	}
	complete := !enrolled && !s.Sessions.mfaRequired(u.Role)
	issued, err := s.Sessions.Issue(ctx, IssueRequest{
		UserID: u.ID, Role: u.Role, RememberMe: in.RememberMe, IP: in.IP, UserAgent: in.UserAgent,
		MFAVerified: complete,
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: u, Session: issued, MFA: NewMFAStatus(enrolled, complete)}, nil
}

// rejectUnknown dépense le temps d'une vérification de mot de passe puis refuse la connexion.
func (s *Service) rejectUnknown(ctx context.Context, password string) error {
	if err := s.Hasher.VerifyDummy(ctx, password); err != nil {
		return fmt.Errorf("auth : vérification factice : %w", err)
	}
	return ErrInvalidCredentials
}

// findByIdentifier cherche par email, ou par pseudo pour un super-héros (cahier §5.2).
func (s *Service) findByIdentifier(ctx context.Context, identifier string) (*user.User, error) {
	if validation.IsEmailLike(identifier) {
		return s.Users.FindByEmail(ctx, identifier)
	}
	return s.Users.FindHeroByAlias(ctx, identifier)
}

// upgradePasswordHash recalcule l'empreinte avec les paramètres actuels. Un échec n'empêche pas
// la connexion : l'ancienne empreinte reste valide.
func (s *Service) upgradePasswordHash(ctx context.Context, u *user.User, password string, now time.Time) {
	hash, err := s.Hasher.Hash(ctx, password)
	if err == nil {
		err = s.Users.UpdatePasswordHash(ctx, u.ID, hash, now)
	}
	if err != nil {
		s.Logger.WarnContext(ctx, "mise à niveau de l'empreinte du mot de passe impossible", "user_id", u.ID, "error", err)
	}
}

// Logout ferme la session d'un jeton. C'est idempotent.
func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	return s.Sessions.Revoke(ctx, sessionToken)
}

// LogoutAll ferme toutes les sessions d'un utilisateur.
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID, ip netip.Addr) error {
	if err := s.Sessions.RevokeAll(ctx, userID); err != nil {
		return err
	}
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &userID, Action: audit.ActionLogoutAllSessions, EntityType: "user", EntityID: &userID, IP: ip,
	})
}

// Me retourne le compte d'un utilisateur authentifié.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return s.Users.FindByID(ctx, userID)
}

// RequestPasswordReset envoie un lien de réinitialisation. Il ne révèle jamais si l'adresse
// correspond à un compte actif : seul le format de l'adresse peut produire une erreur. Une
// nouvelle demande invalide les liens précédents.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized, code := validation.Email(email)
	if code != "" {
		return &validation.Error{Fields: map[string]string{"email": code}}
	}
	u, err := s.Users.FindByEmail(ctx, normalized)
	if errors.Is(err, user.ErrNotFound) || (err == nil && u.Status != user.StatusActive) {
		return nil
	}
	if err != nil {
		return err
	}

	now := s.Clock.Now()
	rawToken, hash, err := newToken()
	if err != nil {
		return err
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Tokens.DeleteUnused(ctx, u.ID, TokenPasswordReset); err != nil {
			return err
		}
		return s.Tokens.Create(ctx, u.ID, TokenPasswordReset, hash, now.Add(s.Security.Tokens.PasswordResetTTL), now)
	})
	if err != nil {
		return err
	}

	link := s.App.PasswordResetLink(rawToken)
	if err := s.Mailer.SendPasswordReset(ctx, u.Email, u.FirstName, link); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de réinitialisation impossible", "user_id", u.ID, "error", err)
	}
	return nil
}

// ConfirmPasswordReset change le mot de passe avec un jeton reçu par mail. Le mot de passe est
// validé avant que le jeton soit consommé : une saisie faible ne brûle pas le lien. Tout se
// passe ensuite dans une transaction, et toutes les sessions de l'utilisateur sont fermées.
func (s *Service) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string, ip netip.Addr) error {
	if !isWellFormedToken(rawToken) {
		return ErrTokenInvalid
	}
	if code := validation.Password(newPassword, s.passwordPolicy()); code != "" {
		return &validation.Error{Fields: map[string]string{"newPassword": code}}
	}
	hash, err := s.Hasher.Hash(ctx, newPassword)
	if err != nil {
		return fmt.Errorf("auth : hachage du mot de passe : %w", err)
	}

	now := s.Clock.Now()
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		userID, err := s.Tokens.Consume(ctx, TokenPasswordReset, hashToken(rawToken), now)
		if err != nil {
			return err
		}
		if err := s.Users.UpdatePasswordHash(ctx, userID, hash, now); err != nil {
			return err
		}
		if err := s.Sessions.RevokeAll(ctx, userID); err != nil {
			return err
		}
		if err := s.Tokens.DeleteUnused(ctx, userID, TokenPasswordReset); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &userID, Action: audit.ActionPasswordReset, EntityType: "user", EntityID: &userID, IP: ip,
		})
	})
}

// PasswordChange est le résultat d'un changement de mot de passe : la session qui remplace la session courante
// (les autres sont fermées) et l'état de sa double authentification.
type PasswordChange struct {
	Session IssuedSession
	MFA     MFAStatus
}

// ChangePassword remplace le mot de passe d'un utilisateur connecté, qui redonne le mot de passe actuel. Toutes
// ses sessions sont fermées, et la session courante (sessionToken) est remplacée par une nouvelle : un jeton
// volé cesse de valoir. Un mot de passe actuel faux est une erreur de champ (currentPassword) ; il ne compte
// pas dans le verrouillage de la connexion, qu'une session volée pourrait sinon déclencher, mais la route est
// limitée en débit. Le titulaire est prévenu par mail ; un échec d'envoi est journalisé sans annuler le
// changement.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, sessionToken, currentPassword, newPassword string, ip netip.Addr) (PasswordChange, error) {
	if currentPassword == "" {
		return PasswordChange{}, &validation.Error{Fields: map[string]string{"currentPassword": validation.CodeRequired}}
	}
	if code := validation.Password(newPassword, s.passwordPolicy()); code != "" {
		return PasswordChange{}, &validation.Error{Fields: map[string]string{"newPassword": code}}
	}
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return PasswordChange{}, err
	}
	check, err := s.Hasher.Verify(ctx, u.PasswordHash, currentPassword)
	if err != nil {
		return PasswordChange{}, fmt.Errorf("auth : vérification du mot de passe : %w", err)
	}
	if !check.Match {
		return PasswordChange{}, &validation.Error{Fields: map[string]string{"currentPassword": validation.CodeInvalid}}
	}
	hash, err := s.Hasher.Hash(ctx, newPassword)
	if err != nil {
		return PasswordChange{}, fmt.Errorf("auth : hachage du mot de passe : %w", err)
	}
	enrolled, err := s.MFA.IsEnrolled(ctx, userID)
	if err != nil {
		return PasswordChange{}, err
	}

	now := s.Clock.Now()
	var issued IssuedSession
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Users.UpdatePasswordHash(ctx, userID, hash, now); err != nil {
			return err
		}
		if issued, err = s.Sessions.Rotate(ctx, sessionToken, true); err != nil {
			return err
		}
		if err := s.Tokens.DeleteUnused(ctx, userID, TokenPasswordReset); err != nil {
			return err
		}
		return s.Audit.Record(ctx, audit.Entry{
			ActorID: &userID, Action: audit.ActionPasswordChanged, EntityType: "user", EntityID: &userID, IP: ip,
		})
	})
	if err != nil {
		return PasswordChange{}, err
	}
	if err := s.Mailer.SendPasswordChanged(ctx, u.Email, u.FirstName); err != nil {
		s.Logger.ErrorContext(ctx, "envoi du mail de changement de mot de passe impossible", "user_id", userID, "error", err)
	}
	return PasswordChange{Session: issued, MFA: NewMFAStatus(enrolled, true)}, nil
}

// Cleanup supprime les sessions et les jetons périmés depuis plus de retention.
func (s *Service) Cleanup(ctx context.Context, sessionRetention, tokenRetention time.Duration) (sessions, tokens int64, err error) {
	if sessions, err = s.Sessions.Cleanup(ctx, sessionRetention); err != nil {
		return 0, 0, err
	}
	tokens, err = s.Tokens.DeleteStale(ctx, s.Clock.Now().Add(-tokenRetention))
	return sessions, tokens, err
}

// CheckPassword redemande le mot de passe actuel d'un utilisateur connecté, avant une action irréversible
// (suppression du compte). Un mot de passe absent ou faux est une erreur de champ (password) ; comme pour
// ChangePassword, il ne compte pas dans le verrouillage de la connexion et la route est limitée en débit.
func (s *Service) CheckPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if password == "" {
		return &validation.Error{Fields: map[string]string{"password": validation.CodeRequired}}
	}
	u, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	check, err := s.Hasher.Verify(ctx, u.PasswordHash, password)
	if err != nil {
		return fmt.Errorf("auth : vérification du mot de passe : %w", err)
	}
	if !check.Match {
		return &validation.Error{Fields: map[string]string{"password": validation.CodeInvalid}}
	}
	return nil
}
