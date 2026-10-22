// Package signedlink fabrique et vérifie des liens signés en Ed25519, pour les actions envoyées par
// mail qui n'ont pas besoin d'un jeton en base (confirmation ou désinscription de la newsletter,
// confirmation de la nouvelle adresse d'un compte).
//
// Un lien désigne un objet (un identifiant, jamais une adresse email : l'URL ne porte aucune donnée
// personnelle), une intention (Purpose) et une date limite. Il est stateless : rien n'est stocké, et
// l'action qu'il déclenche doit donc être idempotente ou protégée par l'état de l'objet.
//
// La clé de signature est dérivée (HKDF-SHA-512, libellé propre à ce paquet) de la clé maîtresse du
// chiffrement des données : aucun secret de plus à gérer, et l'identifiant de la clé maîtresse sert
// d'identifiant de clé (`kid`). La rotation suit celle des clés : les liens signés par une ancienne
// clé restent valides tant qu'elle est configurée.
//
// Format, avant encodage en base64url sans bourrage :
//
//	[version 1 octet | identifiant de clé 2 octets | intention 1 octet |
//	 expiration 8 octets (secondes Unix, 0 = sans limite) | objet 16 octets | signature 64 octets]
package signedlink

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	formatVersion byte = 1

	keyIDLength      = 2
	purposeLength    = 1
	expiryLength     = 8
	subjectLength    = len(uuid.UUID{})
	payloadLength    = 1 + keyIDLength + purposeLength + expiryLength + subjectLength
	tokenLength      = payloadLength + ed25519.SignatureSize
	derivedSeedBytes = ed25519.SeedSize

	// signatureContext sépare ces signatures de toute autre signature faite avec la même clé.
	signatureContext = "heros-sos/signed-link/v1\x00"
	hkdfInfo         = "heros-sos/signed-link/v1/ed25519"

	// masterKeyLength est la taille d'une clé maîtresse (voir internal/crypto).
	masterKeyLength = 64
)

// ErrInvalid indique un lien mal formé, falsifié, expiré, d'une autre intention ou signé par une
// clé inconnue. Le motif n'est volontairement pas distingué : il n'a pas à être connu du client.
var ErrInvalid = errors.New("signedlink : lien invalide ou expiré")

// Purpose est l'intention d'un lien : un lien fabriqué pour une action ne vaut pas pour une autre.
type Purpose byte

const (
	// PurposeNewsletterConfirm confirme un abonnement à la newsletter (double opt-in).
	PurposeNewsletterConfirm Purpose = 1
	// PurposeEmailChangeConfirm confirme la nouvelle adresse d'un compte : l'objet est la demande de
	// changement, dont l'identifiant change quand son titulaire la remplace.
	PurposeEmailChangeConfirm Purpose = 2
	// PurposeNewsletterUnsubscribe désabonne de la newsletter. Il est émis sans date limite : le lien d'une
	// ancienne newsletter doit rester utilisable.
	PurposeNewsletterUnsubscribe Purpose = 3
)

// Signer signe et vérifie des liens. Il est immuable après sa création : il peut être partagé par
// toutes les goroutines.
type Signer struct {
	activeID uint16
	keys     map[uint16]ed25519.PrivateKey
}

// NewSigner dérive la clé de signature de chaque clé maîtresse. activeID désigne la clé qui signe ;
// toutes les clés fournies servent à vérifier (rotation).
func NewSigner(masterKeys map[uint16][]byte, activeID uint16) (*Signer, error) {
	if _, ok := masterKeys[activeID]; !ok {
		return nil, fmt.Errorf("signedlink : la clé active %d n'est pas configurée", activeID)
	}
	keys := make(map[uint16]ed25519.PrivateKey, len(masterKeys))
	for id, master := range masterKeys {
		if len(master) != masterKeyLength {
			return nil, fmt.Errorf("signedlink : la clé %d fait %d octets, %d attendus", id, len(master), masterKeyLength)
		}
		salt := binary.BigEndian.AppendUint16(nil, id)
		seed, err := hkdf.Key(sha512.New, master, salt, hkdfInfo, derivedSeedBytes)
		if err != nil {
			return nil, fmt.Errorf("signedlink : dérivation de la clé %d : %w", id, err)
		}
		keys[id] = ed25519.NewKeyFromSeed(seed)
	}
	return &Signer{activeID: activeID, keys: keys}, nil
}

// Sign retourne un lien valable jusqu'à expiresAt (zéro : sans limite de durée).
func (s *Signer) Sign(purpose Purpose, subject uuid.UUID, expiresAt time.Time) string {
	var expiry uint64
	if !expiresAt.IsZero() {
		expiry = uint64(expiresAt.Unix())
	}

	payload := make([]byte, 0, tokenLength)
	payload = append(payload, formatVersion)
	payload = binary.BigEndian.AppendUint16(payload, s.activeID)
	payload = append(payload, byte(purpose))
	payload = binary.BigEndian.AppendUint64(payload, expiry)
	payload = append(payload, subject[:]...)

	signature := ed25519.Sign(s.keys[s.activeID], signedMessage(payload))
	return base64.RawURLEncoding.EncodeToString(append(payload, signature...))
}

// Verify contrôle un lien et retourne l'objet qu'il désigne. Il retourne ErrInvalid dans tous les
// cas de refus.
func (s *Signer) Verify(token string, purpose Purpose, now time.Time) (uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenLength || raw[0] != formatVersion {
		return uuid.Nil, ErrInvalid
	}
	payload, signature := raw[:payloadLength], raw[payloadLength:]

	key, known := s.keys[binary.BigEndian.Uint16(payload[1:])]
	if !known || !ed25519.Verify(key.Public().(ed25519.PublicKey), signedMessage(payload), signature) {
		return uuid.Nil, ErrInvalid
	}

	const purposeAt = 1 + keyIDLength
	if Purpose(payload[purposeAt]) != purpose {
		return uuid.Nil, ErrInvalid
	}
	const expiryAt = purposeAt + purposeLength
	// uint64 : une date de plus de 2262 ans ne déborde pas, et zéro signifie « sans limite ».
	if expiry := binary.BigEndian.Uint64(payload[expiryAt:]); expiry != 0 && uint64(max(now.Unix(), 0)) > expiry {
		return uuid.Nil, ErrInvalid
	}

	subject, err := uuid.FromBytes(payload[expiryAt+expiryLength:])
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	return subject, nil
}

func signedMessage(payload []byte) []byte {
	return append([]byte(signatureContext), payload...)
}
