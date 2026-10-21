package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/totp"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// MFAStatus dit où en est la double authentification d'une session. C'est ce que le frontend lit pour
// décider quelle page afficher (saisie du code, activation, espace du compte).
type MFAStatus string

const (
	// MFAStatusVerified : la 2FA est activée et validée pour cette session.
	MFAStatusVerified MFAStatus = "verified"
	// MFAStatusVerificationRequired : la 2FA est activée, le code n'a pas encore été saisi. La session ne
	// donne accès qu'à la saisie du code et à la déconnexion.
	MFAStatusVerificationRequired MFAStatus = "verification_required"
	// MFAStatusEnrollmentRequired : la 2FA est obligatoire pour ce compte et pas encore activée. La session ne
	// donne accès qu'à son activation et à la déconnexion.
	MFAStatusEnrollmentRequired MFAStatus = "enrollment_required"
	// MFAStatusNotEnrolled : la 2FA n'est ni activée ni obligatoire. La session est complète.
	MFAStatusNotEnrolled MFAStatus = "not_enrolled"
)

// NewMFAStatus déduit l'état d'une session de celui du compte (2FA activée) et de la session (complète).
func NewMFAStatus(enrolled, verified bool) MFAStatus {
	switch {
	case enrolled && verified:
		return MFAStatusVerified
	case enrolled:
		return MFAStatusVerificationRequired
	case verified:
		return MFAStatusNotEnrolled
	default:
		return MFAStatusEnrollmentRequired
	}
}

var (
	// ErrMFAInvalidCode indique un code (TOTP ou de secours) faux, périmé ou déjà utilisé : le client ne
	// distingue pas ces cas, pour ne rien apprendre à qui devine.
	ErrMFAInvalidCode = errors.New("auth : code de double authentification invalide")
	// ErrMFAAlreadyEnabled indique que la 2FA est déjà activée : elle ne se réinitialise pas par l'API.
	ErrMFAAlreadyEnabled = errors.New("auth : double authentification déjà activée")
	// ErrMFANotSetUp indique que la 2FA n'est pas (ou pas encore) configurée pour ce compte.
	ErrMFANotSetUp = errors.New("auth : double authentification non configurée")
)

const (
	// recoveryCodeBytes : 80 bits d'aléa, soit 16 caractères en base32. Avec cette entropie, l'empreinte
	// SHA-256 stockée ne se retrouve pas par force brute, même avec la base volée.
	recoveryCodeBytes = 10
	// recoveryCodeGroup : un code s'écrit par groupes de quatre, « ABCD-EFGH-IJKL-MNOP ».
	recoveryCodeGroup = 4
)

var recoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// MFADeps regroupe les dépendances du service de double authentification.
type MFADeps struct {
	Store    *MFAStore
	Sessions *SessionService
	Users    user.Repository
	Audit    audit.Recorder
	Tx       database.Transactor
	Clock    clock.Clock
	Security config.Security
}

// MFAService porte les règles de la double authentification par TOTP : activation, validation à la
// connexion, codes de secours. Il ne connaît pas HTTP.
type MFAService struct {
	MFADeps
}

// NewMFAService crée le service.
func NewMFAService(deps MFADeps) *MFAService { return &MFAService{MFADeps: deps} }

// MFASetup est ce qu'il faut pour configurer une application d'authentification.
type MFASetup struct {
	// Secret est le secret en base32, à saisir à la main si l'adresse ne s'ouvre pas.
	Secret string
	// URL est l'adresse otpauth:// que lisent les applications (lien ou QR code).
	URL string
}

// Setup commence l'activation de la 2FA : il tire un secret et le conserve, en attente de confirmation par
// un premier code (Enable). Recommencer remplace le secret d'une activation non confirmée ; une 2FA déjà
// activée ne peut pas être réinitialisée (ErrMFAAlreadyEnabled) : un compte piraté ne doit pas pouvoir
// remplacer sa propre protection.
func (s *MFAService) Setup(ctx context.Context, userID uuid.UUID) (MFASetup, error) {
	account, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return MFASetup{}, err
	}
	secret, err := totp.NewSecret()
	if err != nil {
		return MFASetup{}, err
	}
	started, err := s.Store.StartSetup(ctx, userID, secret, s.Clock.Now())
	if err != nil {
		return MFASetup{}, err
	}
	if !started {
		return MFASetup{}, ErrMFAAlreadyEnabled
	}
	return MFASetup{
		Secret: totp.EncodeSecret(secret),
		URL:    totp.URL(s.Security.MFA.Issuer, account.Email, secret),
	}, nil
}

// MFAEnabled est le résultat d'une activation réussie.
type MFAEnabled struct {
	// Session est la session complète, avec un nouveau jeton : l'appelant doit le poser dans le cookie.
	Session IssuedSession
	// RecoveryCodes sont les codes de secours, montrés cette seule fois : seules leurs empreintes sont gardées.
	RecoveryCodes []string
}

// Enable confirme l'activation avec un premier code, remet les codes de secours, valide la session courante
// (nouveau jeton) et ferme toutes les autres sessions du compte : celles ouvertes avant n'ont jamais passé le
// second facteur.
func (s *MFAService) Enable(ctx context.Context, userID uuid.UUID, sessionToken, code string, ip netip.Addr) (MFAEnabled, error) {
	state, err := s.Store.Find(ctx, userID)
	switch {
	case errors.Is(err, errMFANotFound):
		return MFAEnabled{}, ErrMFANotSetUp
	case err != nil:
		return MFAEnabled{}, err
	case state.Confirmed:
		return MFAEnabled{}, ErrMFAAlreadyEnabled
	}

	now := s.Clock.Now()
	step, ok := totp.Verify(state.Secret, code, now, state.LastUsedStep)
	if !ok {
		return MFAEnabled{}, s.rejectCode(ctx, userID, ip)
	}

	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		return MFAEnabled{}, err
	}
	var issued IssuedSession
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		confirmed, err := s.Store.Confirm(ctx, userID, step, now)
		if err != nil {
			return err
		}
		if !confirmed { // activée entre-temps par une autre requête
			return ErrMFAAlreadyEnabled
		}
		if err := s.Store.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
			return err
		}
		if issued, err = s.Sessions.Rotate(ctx, sessionToken, true); err != nil {
			return err
		}
		return s.record(ctx, userID, audit.ActionMFAEnabled, ip)
	})
	if err != nil {
		return MFAEnabled{}, err
	}
	return MFAEnabled{Session: issued, RecoveryCodes: codes}, nil
}

// MFAVerifyInput est ce que l'utilisateur saisit : un code TOTP, ou à défaut un code de secours.
type MFAVerifyInput struct {
	Code         string
	RecoveryCode string
}

// Verify valide la session d'un compte qui a activé la 2FA : un code TOTP, ou un code de secours (à usage
// unique). Il retourne la session complète, avec un nouveau jeton que l'appelant pose dans le cookie.
//
// Un code faux compte comme un échec de connexion : après quelques-uns le compte est verrouillé avec un
// délai croissant et ses sessions sont fermées (*LockedError). Le compteur n'est remis à zéro que par un code
// juste, pas par le mot de passe seul (voir Service.Login) : sans cela, celui qui connaît le mot de passe se
// reconnecterait entre deux séries de devinettes.
func (s *MFAService) Verify(ctx context.Context, userID uuid.UUID, sessionToken string, in MFAVerifyInput, ip netip.Addr) (IssuedSession, error) {
	account, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return IssuedSession{}, err
	}
	now := s.Clock.Now()
	if account.IsLocked(now) {
		return IssuedSession{}, &LockedError{Until: *account.LockedUntil}
	}
	state, err := s.confirmed(ctx, userID)
	if err != nil {
		return IssuedSession{}, err
	}

	if in.RecoveryCode != "" {
		used, err := s.Store.UseRecoveryCode(ctx, userID, hashRecoveryCode(in.RecoveryCode), now)
		if err != nil {
			return IssuedSession{}, err
		}
		if !used {
			return IssuedSession{}, s.rejectCode(ctx, userID, ip)
		}
		if err := s.record(ctx, userID, audit.ActionMFARecoveryUsed, ip); err != nil {
			return IssuedSession{}, err
		}
	} else if err := s.checkCode(ctx, userID, state, in.Code, now, ip); err != nil {
		return IssuedSession{}, err
	}

	if err := s.Users.RecordLoginSuccess(ctx, userID, now); err != nil {
		return IssuedSession{}, err
	}
	return s.Sessions.Rotate(ctx, sessionToken, false)
}

// RegenerateRecoveryCodes remplace les codes de secours (épuisés, ou perdus). Il demande le code TOTP
// courant : une session volée ne suffit pas à s'en procurer de nouveaux.
func (s *MFAService) RegenerateRecoveryCodes(ctx context.Context, userID uuid.UUID, code string, ip netip.Addr) ([]string, error) {
	state, err := s.confirmed(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, userID, state, code, s.Clock.Now(), ip); err != nil {
		return nil, err
	}

	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Store.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
			return err
		}
		return s.record(ctx, userID, audit.ActionMFARecoveryRegenerated, ip)
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// RecoveryCodesRemaining compte les codes de secours qui n'ont pas servi.
func (s *MFAService) RecoveryCodesRemaining(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.Store.RecoveryCodesRemaining(ctx, userID)
}

// confirmed retourne la 2FA d'un compte qui l'a activée (ErrMFANotSetUp sinon).
func (s *MFAService) confirmed(ctx context.Context, userID uuid.UUID) (*MFA, error) {
	state, err := s.Store.Find(ctx, userID)
	switch {
	case errors.Is(err, errMFANotFound):
		return nil, ErrMFANotSetUp
	case err != nil:
		return nil, err
	case !state.Confirmed:
		return nil, ErrMFANotSetUp
	}
	return state, nil
}

// checkCode contrôle un code TOTP et le consomme. Le pas n'est mémorisé qu'après le contrôle, en une seule
// instruction : deux requêtes simultanées avec le même code ne peuvent pas réussir toutes les deux.
func (s *MFAService) checkCode(ctx context.Context, userID uuid.UUID, state *MFA, code string, now time.Time, ip netip.Addr) error {
	step, ok := totp.Verify(state.Secret, code, now, state.LastUsedStep)
	if ok {
		consumed, err := s.Store.ConsumeStep(ctx, userID, step)
		if err != nil {
			return err
		}
		ok = consumed
	}
	if !ok {
		return s.rejectCode(ctx, userID, ip)
	}
	return nil
}

// rejectCode compte un code refusé comme un échec de connexion, et le journalise. Au seuil de verrouillage,
// les sessions du compte sont fermées : il faut se reconnecter, et le compte est verrouillé pour un temps.
func (s *MFAService) rejectCode(ctx context.Context, userID uuid.UUID, ip netip.Addr) error {
	failure, err := s.Users.RecordLoginFailure(ctx, userID, s.Clock.Now(), s.Security.Lockout)
	if err != nil {
		return err
	}
	if err := s.Audit.Record(ctx, audit.Entry{
		ActorID: &userID, Action: audit.ActionMFAFailed, EntityType: "user", EntityID: &userID,
		Details: map[string]any{"failures": failure.Count}, IP: ip,
	}); err != nil {
		return err
	}
	if failure.LockedUntil == nil {
		return ErrMFAInvalidCode
	}
	if err := s.Sessions.RevokeAll(ctx, userID); err != nil {
		return err
	}
	return &LockedError{Until: *failure.LockedUntil}
}

func (s *MFAService) record(ctx context.Context, userID uuid.UUID, action string, ip netip.Addr) error {
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &userID, Action: action, EntityType: "user", EntityID: &userID, IP: ip,
	})
}

// newRecoveryCodes tire les codes de secours : le texte à montrer, et l'empreinte à garder.
func (s *MFAService) newRecoveryCodes() (codes []string, hashes [][]byte, err error) {
	for range s.Security.MFA.RecoveryCodes {
		raw := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, fmt.Errorf("auth : tirage d'un code de secours : %w", err)
		}
		code := recoveryEncoding.EncodeToString(raw)
		codes = append(codes, formatRecoveryCode(code))
		hashes = append(hashes, hashRecoveryCode(code))
	}
	return codes, hashes, nil
}

// formatRecoveryCode écrit un code par groupes de quatre caractères, plus faciles à recopier.
func formatRecoveryCode(code string) string {
	var groups []string
	for start := 0; start < len(code); start += recoveryCodeGroup {
		groups = append(groups, code[start:min(start+recoveryCodeGroup, len(code))])
	}
	return strings.Join(groups, "-")
}

// hashRecoveryCode normalise la saisie (casse, tirets et espaces ignorés) avant de calculer l'empreinte :
// le code se recopie à la main.
func hashRecoveryCode(code string) []byte {
	normalized := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(code))
	sum := sha256.Sum256([]byte(normalized))
	return sum[:]
}
