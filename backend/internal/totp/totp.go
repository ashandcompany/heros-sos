// Package totp implémente les mots de passe à usage unique basés sur le temps (RFC 6238, sur HOTP,
// RFC 4226) que lisent les applications d'authentification (Google Authenticator, Aegis, 1Password…).
//
// Les paramètres (SHA-1, 6 chiffres, pas de 30 secondes) sont ceux que ces applications supposent quand
// l'adresse otpauth:// ne les précise pas : ce sont des constantes de protocole, pas des réglages. SHA-1
// n'est pas un point faible ici (HMAC-SHA-1 reste sûr), et c'est le seul algorithme que toutes les
// applications acceptent.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // HMAC-SHA-1 : imposé par les applications d'authentification, sûr en HMAC.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// SecretLength est la taille du secret, en octets : 160 bits, la taille de la sortie de SHA-1 (RFC 4226 §4).
	SecretLength = 20
	// Digits est le nombre de chiffres d'un code.
	Digits = 6
	// Period est la durée de validité d'un code, en secondes.
	Period = 30
	// Skew est le nombre de pas acceptés avant et après le pas courant, pour tolérer une horloge
	// légèrement décalée (±30 secondes) sans élargir sensiblement la fenêtre d'une devinette.
	Skew = 1

	modulus = 1_000_000 // 10^Digits
)

// secretEncoding : base32 sans bourrage, en majuscules, comme le lisent les applications.
var secretEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret tire un secret aléatoire.
func NewSecret() ([]byte, error) {
	secret := make([]byte, SecretLength)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("totp : tirage du secret : %w", err)
	}
	return secret, nil
}

// EncodeSecret écrit le secret en base32, pour le saisir à la main dans une application.
func EncodeSecret(secret []byte) string { return secretEncoding.EncodeToString(secret) }

// Step est le numéro du pas de temps qui contient t.
func Step(t time.Time) int64 { return t.Unix() / Period }

// Code calcule le code d'un pas (HOTP, RFC 4226 §5.3).
func Code(secret []byte, step int64) string {
	mac := hmac.New(sha1.New, secret)
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))
	mac.Write(counter[:])
	sum := mac.Sum(nil)

	// Troncature dynamique : 4 octets à une position donnée par le dernier quartet, bit de signe retiré.
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", Digits, value%modulus)
}

// Verify contrôle un code saisi à l'instant now. Il accepte le pas courant et ses voisins (Skew), mais
// jamais un pas déjà utilisé : lastUsedStep est le dernier pas accepté pour ce secret, et un code observé
// (épaule, enregistrement) ne peut pas être rejoué. Il retourne le pas accepté, que l'appelant doit
// mémoriser (de façon atomique) avant de considérer le code comme consommé.
func Verify(secret []byte, code string, now time.Time, lastUsedStep int64) (step int64, ok bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != Digits {
		return 0, false
	}
	if _, err := strconv.ParseUint(code, 10, 32); err != nil {
		return 0, false
	}

	current := Step(now)
	// Tous les pas sont évalués, sans sortie anticipée : le temps de réponse ne dit pas quel pas
	// correspondait.
	matched := int64(-1)
	for candidate := current - Skew; candidate <= current+Skew; candidate++ {
		if subtle.ConstantTimeCompare([]byte(code), []byte(Code(secret, candidate))) == 1 && candidate > lastUsedStep {
			matched = candidate
		}
	}
	return matched, matched >= 0
}

// URL construit l'adresse otpauth:// qu'une application d'authentification lit (par QR code ou par lien) :
// https://github.com/google/google-authenticator/wiki/Key-Uri-Format.
func URL(issuer, account string, secret []byte) string {
	label := escapeLabel(issuer) + ":" + escapeLabel(account)
	query := url.Values{
		"secret":    {EncodeSecret(secret)},
		"issuer":    {issuer},
		"algorithm": {"SHA1"},
		"digits":    {strconv.Itoa(Digits)},
		"period":    {strconv.Itoa(Period)},
	}
	return "otpauth://totp/" + label + "?" + query.Encode()
}

// escapeLabel encode un élément de l'étiquette « émetteur:compte ». PathEscape laisse passer les deux-points,
// qui sont pourtant le séparateur de l'étiquette : un émetteur « A:B » se lirait comme émetteur « A ».
func escapeLabel(value string) string {
	return strings.ReplaceAll(url.PathEscape(value), ":", "%3A")
}
