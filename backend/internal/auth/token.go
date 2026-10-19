package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	tokenByteLength = 32
	// tokenEncodedLength est la longueur d'un jeton de 32 octets en base64url sans bourrage.
	tokenEncodedLength = 43
)

// newToken tire un jeton opaque de 256 bits. Le jeton brut part chez l'utilisateur (cookie ou
// lien d'un mail) ; seule son empreinte SHA-256 est stockée : une fuite de la base ne donne donc
// aucun jeton utilisable. Une empreinte rapide suffit, le jeton ayant déjà 256 bits d'entropie.
func newToken() (raw string, hash []byte, err error) {
	buffer := make([]byte, tokenByteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", nil, fmt.Errorf("auth : tirage du jeton : %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buffer)
	return raw, hashToken(raw), nil
}

// hashToken retourne l'empreinte stockée en base pour un jeton brut.
func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// isWellFormedToken vérifie la forme d'un jeton avant tout accès à la base : cela écarte
// d'emblée les valeurs absurdes ou démesurées sans coût.
func isWellFormedToken(raw string) bool {
	if len(raw) != tokenEncodedLength {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	return err == nil && len(decoded) == tokenByteLength
}
