// Package crypto chiffre les données personnelles stockées en base.
//
// Chaque valeur est chiffrée par une cascade de deux algorithmes standards avec deux clés
// indépendantes : AES-256-GCM puis XChaCha20-Poly1305. Si l'un des deux est un jour affaibli,
// l'autre protège encore la donnée. Les deux clés de 256 bits sont dérivées (HKDF-SHA-512)
// d'une clé maîtresse de 512 bits.
//
// Format d'une valeur chiffrée :
//
//	[version 1 octet | algorithme 1 octet | identifiant de clé 2 octets |
//	 nonce AES-GCM 12 octets | nonce XChaCha20 24 octets | chiffré avec ses deux tags]
//
// L'en-tête et le contexte (table, colonne, identifiant de la ligne) sont authentifiés par les
// deux couches : un chiffré ne peut pas être déplacé vers une autre ligne ou une autre colonne.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	formatVersion    byte = 1
	algorithmCascade byte = 1

	// MasterKeyLength est la taille d'une clé maîtresse : 512 bits.
	MasterKeyLength = 64

	derivedKeyLength = 32
	keyIDLength      = 2
	headerLength     = 2 + keyIDLength

	// gcmNonceLength est la taille standard du nonce d'AES-GCM ; tagLength celle du tag
	// d'authentification d'AES-GCM comme de Poly1305.
	gcmNonceLength = 12
	tagLength      = 16

	hkdfInfoAES     = "heros-sos/crypto/v1/aes-256-gcm"
	hkdfInfoXChaCha = "heros-sos/crypto/v1/xchacha20-poly1305"
)

// Overhead est le nombre d'octets ajoutés à chaque valeur chiffrée : en-tête, deux nonces
// (12 et 24 octets) et deux tags d'authentification de 16 octets.
const Overhead = headerLength + gcmNonceLength + chacha20poly1305.NonceSizeX + 2*tagLength

var (
	// ErrInvalidFormat indique qu'une valeur n'a pas la structure d'un chiffré de ce paquet.
	ErrInvalidFormat = errors.New("crypto : format de donnée chiffrée invalide")
	// ErrUnknownKey indique que la clé qui a chiffré la valeur n'est plus configurée.
	ErrUnknownKey = errors.New("crypto : clé de chiffrement inconnue")
	// ErrDecryption indique un échec d'authentification : mauvaise clé, contexte différent
	// ou donnée altérée. Le message ne dit pas quelle couche a échoué.
	ErrDecryption = errors.New("crypto : déchiffrement impossible")
)

// AAD est le contexte lié à un chiffré. Il n'est pas stocké : il est reconstruit à la lecture
// et doit être identique à celui de l'écriture.
type AAD struct {
	Table    string
	Column   string
	RecordID uuid.UUID
}

// bytes encode le contexte de façon non ambiguë : chaque champ est précédé de sa longueur.
func (a AAD) bytes() []byte {
	buffer := make([]byte, 0, 2+len(a.Table)+2+len(a.Column)+len(a.RecordID))
	buffer = binary.BigEndian.AppendUint16(buffer, uint16(len(a.Table)))
	buffer = append(buffer, a.Table...)
	buffer = binary.BigEndian.AppendUint16(buffer, uint16(len(a.Column)))
	buffer = append(buffer, a.Column...)
	return append(buffer, a.RecordID[:]...)
}

// layers contient les deux chiffrements d'une version de clé. Les AEAD de la bibliothèque
// standard sont sûrs pour un usage concurrent.
type layers struct {
	inner cipher.AEAD // AES-256-GCM
	outer cipher.AEAD // XChaCha20-Poly1305
}

// Sealer chiffre et déchiffre. Il est immuable après sa création : un même Sealer peut être
// partagé par toutes les goroutines sans verrou.
type Sealer struct {
	activeID uint16
	keys     map[uint16]layers
	random   io.Reader
}

// NewSealer dérive les sous-clés de chaque clé maîtresse. activeID désigne la clé utilisée
// pour chiffrer ; toutes les clés fournies servent à déchiffrer (rotation).
func NewSealer(masterKeys map[uint16][]byte, activeID uint16) (*Sealer, error) {
	return newSealer(masterKeys, activeID, rand.Reader)
}

func newSealer(masterKeys map[uint16][]byte, activeID uint16, random io.Reader) (*Sealer, error) {
	if _, ok := masterKeys[activeID]; !ok {
		return nil, fmt.Errorf("crypto : la clé active %d n'est pas configurée", activeID)
	}

	keys := make(map[uint16]layers, len(masterKeys))
	for id, master := range masterKeys {
		set, err := deriveLayers(id, master)
		if err != nil {
			return nil, err
		}
		keys[id] = set
	}
	return &Sealer{activeID: activeID, keys: keys, random: random}, nil
}

// deriveKeys dérive les deux clés indépendantes d'une clé maîtresse.
func deriveKeys(id uint16, master []byte) (aesKey, xchachaKey []byte, err error) {
	if len(master) != MasterKeyLength {
		return nil, nil, fmt.Errorf("crypto : la clé %d fait %d octets, %d attendus", id, len(master), MasterKeyLength)
	}
	salt := binary.BigEndian.AppendUint16(nil, id)

	aesKey, err = hkdf.Key(sha512.New, master, salt, hkdfInfoAES, derivedKeyLength)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto : dérivation de la clé AES : %w", err)
	}
	xchachaKey, err = hkdf.Key(sha512.New, master, salt, hkdfInfoXChaCha, derivedKeyLength)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto : dérivation de la clé XChaCha20 : %w", err)
	}
	return aesKey, xchachaKey, nil
}

func deriveLayers(id uint16, master []byte) (layers, error) {
	aesKey, xchachaKey, err := deriveKeys(id, master)
	if err != nil {
		return layers{}, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return layers{}, fmt.Errorf("crypto : initialisation d'AES : %w", err)
	}
	inner, err := cipher.NewGCM(block)
	if err != nil {
		return layers{}, fmt.Errorf("crypto : initialisation de GCM : %w", err)
	}
	outer, err := chacha20poly1305.NewX(xchachaKey)
	if err != nil {
		return layers{}, fmt.Errorf("crypto : initialisation de XChaCha20-Poly1305 : %w", err)
	}
	return layers{inner: inner, outer: outer}, nil
}

// ActiveKeyID retourne l'identifiant de la clé utilisée pour chiffrer.
func (s *Sealer) ActiveKeyID() uint16 { return s.activeID }

// Seal chiffre plaintext pour le contexte aad.
func (s *Sealer) Seal(plaintext []byte, aad AAD) ([]byte, error) {
	set := s.keys[s.activeID]
	header := makeHeader(s.activeID)
	associated := associatedData(header, aad)

	innerNonce := make([]byte, set.inner.NonceSize())
	outerNonce := make([]byte, set.outer.NonceSize())
	if _, err := io.ReadFull(s.random, innerNonce); err != nil {
		return nil, fmt.Errorf("crypto : tirage du nonce AES : %w", err)
	}
	if _, err := io.ReadFull(s.random, outerNonce); err != nil {
		return nil, fmt.Errorf("crypto : tirage du nonce XChaCha20 : %w", err)
	}

	inner := set.inner.Seal(nil, innerNonce, plaintext, associated)
	outer := set.outer.Seal(nil, outerNonce, inner, associated)

	sealed := make([]byte, 0, len(header)+len(innerNonce)+len(outerNonce)+len(outer))
	sealed = append(sealed, header...)
	sealed = append(sealed, innerNonce...)
	sealed = append(sealed, outerNonce...)
	return append(sealed, outer...), nil
}

// Open déchiffre une valeur produite par Seal pour le même contexte.
func (s *Sealer) Open(sealed []byte, aad AAD) ([]byte, error) {
	id, err := KeyIDOf(sealed)
	if err != nil {
		return nil, err
	}
	set, ok := s.keys[id]
	if !ok {
		return nil, fmt.Errorf("%w : %d", ErrUnknownKey, id)
	}

	innerNonceSize, outerNonceSize := set.inner.NonceSize(), set.outer.NonceSize()
	if len(sealed) < Overhead {
		return nil, ErrInvalidFormat
	}
	header := sealed[:headerLength]
	innerNonce := sealed[headerLength : headerLength+innerNonceSize]
	outerNonce := sealed[headerLength+innerNonceSize : headerLength+innerNonceSize+outerNonceSize]
	ciphertext := sealed[headerLength+innerNonceSize+outerNonceSize:]
	associated := associatedData(header, aad)

	inner, err := set.outer.Open(nil, outerNonce, ciphertext, associated)
	if err != nil {
		return nil, ErrDecryption
	}
	plaintext, err := set.inner.Open(nil, innerNonce, inner, associated)
	if err != nil {
		return nil, ErrDecryption
	}
	return plaintext, nil
}

// SealString chiffre une chaîne.
func (s *Sealer) SealString(value string, aad AAD) ([]byte, error) {
	return s.Seal([]byte(value), aad)
}

// OpenString déchiffre une chaîne.
func (s *Sealer) OpenString(sealed []byte, aad AAD) (string, error) {
	plaintext, err := s.Open(sealed, aad)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// NeedsRotation indique si une valeur a été chiffrée avec une autre clé que la clé active,
// pour la rechiffrer lors d'une rotation.
func (s *Sealer) NeedsRotation(sealed []byte) bool {
	id, err := KeyIDOf(sealed)
	return err == nil && id != s.activeID
}

// KeyIDOf lit l'identifiant de clé inscrit dans l'en-tête d'une valeur chiffrée.
func KeyIDOf(sealed []byte) (uint16, error) {
	if len(sealed) < headerLength || sealed[0] != formatVersion || sealed[1] != algorithmCascade {
		return 0, ErrInvalidFormat
	}
	return binary.BigEndian.Uint16(sealed[2:headerLength]), nil
}

// associatedData construit les données authentifiées par les deux couches : en-tête puis contexte.
func associatedData(header []byte, aad AAD) []byte {
	context := aad.bytes()
	associated := make([]byte, 0, len(header)+len(context))
	associated = append(associated, header...)
	return append(associated, context...)
}

func makeHeader(keyID uint16) []byte {
	return binary.BigEndian.AppendUint16([]byte{formatVersion, algorithmCascade}, keyID)
}
