// Package auth gère l'authentification : mots de passe, sessions, jetons à usage unique,
// protection CSRF, et les endpoints /auth/*.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

const (
	phcAlgorithm  = "argon2id"
	phcPartsCount = 6 // "", algorithme, version, paramètres, sel, empreinte
	dummyPassword = "mot de passe factice pour égaliser les temps de réponse"
)

// ErrInvalidHash indique qu'une empreinte stockée n'est pas au format attendu.
var ErrInvalidHash = errors.New("auth : empreinte de mot de passe invalide")

// PasswordCheck est le résultat de la vérification d'un mot de passe.
type PasswordCheck struct {
	Match bool
	// NeedsRehash est vrai quand l'empreinte a été calculée avec des paramètres plus faibles
	// que la configuration actuelle : elle doit être recalculée à la prochaine connexion.
	NeedsRehash bool
}

// PasswordHasher hache et vérifie les mots de passe avec argon2id. Chaque calcul consomme
// plusieurs dizaines de Mio de mémoire : le nombre de calculs simultanés est plafonné pour
// qu'un afflux de connexions ne puisse pas épuiser la mémoire du serveur.
type PasswordHasher struct {
	params config.Argon2
	slots  chan struct{}

	dummyOnce sync.Once
	dummyHash string
	dummyErr  error
}

// NewPasswordHasher crée un hacheur avec les paramètres donnés.
func NewPasswordHasher(params config.Argon2) *PasswordHasher {
	return &PasswordHasher{params: params, slots: make(chan struct{}, params.MaxConcurrent)}
}

// acquire réserve un emplacement de calcul, ou échoue si le contexte expire avant.
func (h *PasswordHasher) acquire(ctx context.Context) (release func(), err error) {
	select {
	case h.slots <- struct{}{}:
		return func() { <-h.slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Hash retourne l'empreinte au format PHC : $argon2id$v=19$m=…,t=…,p=…$sel$empreinte.
func (h *PasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	release, err := h.acquire(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth : tirage du sel : %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.params.Time, h.params.MemoryKiB, h.params.Threads, h.params.KeyLength)
	return encodeHash(h.params.Time, h.params.MemoryKiB, h.params.Threads, salt, key), nil
}

// Verify compare un mot de passe à une empreinte. La comparaison se fait en temps constant,
// avec les paramètres inscrits dans l'empreinte (ils ont pu changer depuis sa création).
func (h *PasswordHasher) Verify(ctx context.Context, encoded, password string) (PasswordCheck, error) {
	parsed, err := decodeHash(encoded)
	if err != nil {
		return PasswordCheck{}, err
	}

	release, err := h.acquire(ctx)
	if err != nil {
		return PasswordCheck{}, err
	}
	defer release()

	candidate := argon2.IDKey([]byte(password), parsed.salt, parsed.time, parsed.memoryKiB, parsed.threads, uint32(len(parsed.key)))
	return PasswordCheck{
		Match: subtle.ConstantTimeCompare(candidate, parsed.key) == 1,
		NeedsRehash: parsed.time != h.params.Time || parsed.memoryKiB != h.params.MemoryKiB ||
			parsed.threads != h.params.Threads || uint32(len(parsed.key)) != h.params.KeyLength,
	}, nil
}

// VerifyDummy dépense le même temps qu'une vérification réelle. Il est appelé quand l'identifiant
// n'existe pas, pour que la durée de la réponse ne révèle pas quels comptes existent.
func (h *PasswordHasher) VerifyDummy(ctx context.Context, password string) error {
	h.dummyOnce.Do(func() {
		h.dummyHash, h.dummyErr = h.Hash(context.Background(), dummyPassword)
	})
	if h.dummyErr != nil {
		return h.dummyErr
	}
	_, err := h.Verify(ctx, h.dummyHash, password)
	return err
}

type parsedHash struct {
	time      uint32
	memoryKiB uint32
	threads   uint8
	salt      []byte
	key       []byte
}

func encodeHash(time, memoryKiB uint32, threads uint8, salt, key []byte) string {
	encoding := base64.RawStdEncoding
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		phcAlgorithm, argon2.Version, memoryKiB, time, threads,
		encoding.EncodeToString(salt), encoding.EncodeToString(key))
}

func decodeHash(encoded string) (parsedHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != phcPartsCount || parts[0] != "" || parts[1] != phcAlgorithm {
		return parsedHash{}, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return parsedHash{}, ErrInvalidHash
	}

	var parsed parsedHash
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &parsed.memoryKiB, &parsed.time, &threads); err != nil ||
		threads == 0 || threads > math.MaxUint8 || parsed.time == 0 || parsed.memoryKiB == 0 {
		return parsedHash{}, ErrInvalidHash
	}
	parsed.threads = uint8(threads)

	encoding := base64.RawStdEncoding
	var err error
	if parsed.salt, err = encoding.DecodeString(parts[4]); err != nil || len(parsed.salt) == 0 {
		return parsedHash{}, ErrInvalidHash
	}
	if parsed.key, err = encoding.DecodeString(parts[5]); err != nil || len(parsed.key) == 0 {
		return parsedHash{}, ErrInvalidHash
	}
	return parsed, nil
}
