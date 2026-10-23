package signedlink

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	now     = time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	subject = uuid.MustParse("0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10")
)

func masterKey(fill byte) []byte { return bytes.Repeat([]byte{fill}, masterKeyLength) }

func newSigner(t *testing.T, keys map[uint16][]byte, active uint16) *Signer {
	t.Helper()
	signer, err := NewSigner(keys, active)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func TestSignedLinkRoundTrip(t *testing.T) {
	signer := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	token := signer.Sign(PurposeNewsletterConfirm, subject, now.Add(time.Hour))

	got, err := signer.Verify(token, PurposeNewsletterConfirm, now)
	if err != nil || got != subject {
		t.Fatalf("Verify = %v, %v", got, err)
	}
	// Seul de l'URL-safe sans bourrage : le lien se place tel quel dans un paramètre de requête.
	if strings.ContainsAny(token, "+/=") {
		t.Errorf("jeton non URL-safe : %q", token)
	}
	if token == signer.Sign(PurposeNewsletterConfirm, uuid.New(), now.Add(time.Hour)) {
		t.Error("deux objets différents ne peuvent pas produire le même lien")
	}
}

func TestSignedLinkExpiry(t *testing.T) {
	signer := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	deadline := now.Add(time.Hour)
	token := signer.Sign(PurposeNewsletterConfirm, subject, deadline)

	for name, at := range map[string]time.Time{"juste avant": deadline, "bien avant": now} {
		if _, err := signer.Verify(token, PurposeNewsletterConfirm, at); err != nil {
			t.Errorf("%s : %v", name, err)
		}
	}
	if _, err := signer.Verify(token, PurposeNewsletterConfirm, deadline.Add(time.Second)); !errors.Is(err, ErrInvalid) {
		t.Errorf("un lien expiré doit être refusé : %v", err)
	}

	forever := signer.Sign(PurposeNewsletterConfirm, subject, time.Time{})
	if _, err := signer.Verify(forever, PurposeNewsletterConfirm, now.AddDate(50, 0, 0)); err != nil {
		t.Errorf("un lien sans limite ne doit jamais expirer : %v", err)
	}
}

func TestSignedLinkRejectsWrongPurposeAndTampering(t *testing.T) {
	signer := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	token := signer.Sign(PurposeNewsletterConfirm, subject, now.Add(time.Hour))

	if _, err := signer.Verify(token, Purpose(99), now); !errors.Is(err, ErrInvalid) {
		t.Errorf("une autre intention doit être refusée : %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	// Modifier un seul bit, n'importe où (en-tête, objet, expiration ou signature), invalide le lien.
	for position := range raw {
		forged := bytes.Clone(raw)
		forged[position] ^= 0x01
		if _, err := signer.Verify(base64.RawURLEncoding.EncodeToString(forged), PurposeNewsletterConfirm, now); !errors.Is(err, ErrInvalid) {
			t.Fatalf("octet %d falsifié accepté : %v", position, err)
		}
	}
}

func TestSignedLinkRejectsMalformedInput(t *testing.T) {
	signer := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	valid := signer.Sign(PurposeNewsletterConfirm, subject, now.Add(time.Hour))

	for name, token := range map[string]string{
		"vide":             "",
		"pas du base64":    "%%%",
		"trop court":       valid[:len(valid)-4],
		"trop long":        valid + "AAAA",
		"bourrage":         valid + "=",
		"texte arbitraire": "collez-ici-le-jeton-du-lien-recu-par-mail",
	} {
		if _, err := signer.Verify(token, PurposeNewsletterConfirm, now); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s : %v", name, err)
		}
	}
}

func TestSignedLinkKeyRotation(t *testing.T) {
	oldOnly := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	token := oldOnly.Sign(PurposeNewsletterConfirm, subject, now.Add(time.Hour))

	rotated := newSigner(t, map[uint16][]byte{1: masterKey(1), 2: masterKey(2)}, 2)
	if got, err := rotated.Verify(token, PurposeNewsletterConfirm, now); err != nil || got != subject {
		t.Errorf("un lien signé par l'ancienne clé doit rester valide : %v", err)
	}
	fresh := rotated.Sign(PurposeNewsletterConfirm, subject, now.Add(time.Hour))
	if _, err := oldOnly.Verify(fresh, PurposeNewsletterConfirm, now); !errors.Is(err, ErrInvalid) {
		t.Errorf("une clé retirée ne peut plus vérifier : %v", err)
	}

	// Même identifiant de clé mais autre clé maîtresse : la signature ne correspond pas.
	impostor := newSigner(t, map[uint16][]byte{1: masterKey(9)}, 1)
	if _, err := impostor.Verify(token, PurposeNewsletterConfirm, now); !errors.Is(err, ErrInvalid) {
		t.Errorf("une autre clé maîtresse doit être refusée : %v", err)
	}
}

func TestNewSignerValidatesKeys(t *testing.T) {
	if _, err := NewSigner(map[uint16][]byte{1: masterKey(1)}, 2); err == nil {
		t.Error("une clé active absente doit être refusée")
	}
	if _, err := NewSigner(map[uint16][]byte{1: []byte("trop courte")}, 1); err == nil {
		t.Error("une clé de mauvaise taille doit être refusée")
	}
}

// Un lien fabriqué pour une action ne vaut jamais pour une autre : confirmer une adresse ne doit pas
// désabonner, ni l'inverse.
func TestSignedLinkPurposesAreNotInterchangeable(t *testing.T) {
	signer := newSigner(t, map[uint16][]byte{1: masterKey(1)}, 1)
	purposes := []Purpose{PurposeNewsletterConfirm, PurposeEmailChangeConfirm, PurposeNewsletterUnsubscribe}

	for _, issued := range purposes {
		token := signer.Sign(issued, subject, now.Add(time.Hour))
		for _, wanted := range purposes {
			got, err := signer.Verify(token, wanted, now)
			switch {
			case issued == wanted && (err != nil || got != subject):
				t.Errorf("intention %d : Verify = %v, %v", issued, got, err)
			case issued != wanted && !errors.Is(err, ErrInvalid):
				t.Errorf("un lien d'intention %d ne doit pas valoir pour l'intention %d : %v", issued, wanted, err)
			}
		}
	}
}
