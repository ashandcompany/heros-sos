package auth

import (
	"errors"
	"time"
)

var (
	// ErrInvalidCredentials couvre tous les échecs de connexion « normaux » (identifiant
	// inconnu, mauvais mot de passe) : le client ne doit pas savoir lequel.
	ErrInvalidCredentials = errors.New("auth : identifiant ou mot de passe incorrect")
	// ErrAccountNotActivated est retourné seulement après un mot de passe correct.
	ErrAccountNotActivated = errors.New("auth : compte non activé")
	// ErrAccountDeactivated est retourné seulement après un mot de passe correct.
	ErrAccountDeactivated = errors.New("auth : compte désactivé")
	// ErrAccountLocked indique un verrouillage temporaire après trop d'échecs.
	ErrAccountLocked = errors.New("auth : compte temporairement verrouillé")
	// ErrTokenInvalid indique un jeton inconnu, expiré, déjà utilisé ou mal formé.
	ErrTokenInvalid = errors.New("auth : jeton invalide, expiré ou déjà utilisé")
)

// LockedError précise jusqu'à quand un compte est verrouillé. errors.Is(err, ErrAccountLocked)
// reste vrai.
type LockedError struct {
	Until time.Time
}

func (e *LockedError) Error() string { return ErrAccountLocked.Error() }

// Is permet errors.Is(err, ErrAccountLocked).
func (e *LockedError) Is(target error) bool { return target == ErrAccountLocked }
