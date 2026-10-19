package crypto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func testKey(fill byte) []byte { return bytes.Repeat([]byte{fill}, MasterKeyLength) }

func newTestSealer(t *testing.T) *Sealer {
	t.Helper()
	sealer, err := NewSealer(map[uint16][]byte{1: testKey(0x11)}, 1)
	if err != nil {
		t.Fatalf("NewSealer : %v", err)
	}
	return sealer
}

func testAAD() AAD {
	return AAD{Table: "users", Column: "last_name", RecordID: uuid.MustParse("0192f4a0-0000-7000-8000-000000000001")}
}

func TestSealOpenRoundTrip(t *testing.T) {
	sealer := newTestSealer(t)
	tests := map[string][]byte{
		"vide":         {},
		"court":        []byte("Rogers"),
		"unicode":      []byte("Éloïse d'Ünïcode ✨"),
		"un mégaoctet": bytes.Repeat([]byte("héros"), 200_000),
	}
	for name, plaintext := range tests {
		t.Run(name, func(t *testing.T) {
			sealed, err := sealer.Seal(plaintext, testAAD())
			if err != nil {
				t.Fatalf("Seal : %v", err)
			}
			if got, want := len(sealed), len(plaintext)+Overhead; got != want {
				t.Errorf("longueur = %d, attendu %d (surcoût %d)", got, want, Overhead)
			}
			opened, err := sealer.Open(sealed, testAAD())
			if err != nil {
				t.Fatalf("Open : %v", err)
			}
			if !bytes.Equal(opened, plaintext) {
				t.Error("le texte déchiffré diffère de l'original")
			}
		})
	}
}

func TestSealString(t *testing.T) {
	sealer := newTestSealer(t)
	sealed, err := sealer.SealString("0612345678", testAAD())
	if err != nil {
		t.Fatal(err)
	}
	got, err := sealer.OpenString(sealed, testAAD())
	if err != nil || got != "0612345678" {
		t.Fatalf("OpenString = (%q, %v)", got, err)
	}
}

func TestSealIsRandomised(t *testing.T) {
	sealer := newTestSealer(t)
	first, _ := sealer.Seal([]byte("identique"), testAAD())
	second, _ := sealer.Seal([]byte("identique"), testAAD())
	if bytes.Equal(first, second) {
		t.Fatal("deux chiffrements du même texte doivent différer (nonces aléatoires)")
	}
}

func TestOpenRejectsAnyTampering(t *testing.T) {
	sealer := newTestSealer(t)
	sealed, err := sealer.Seal([]byte("donnée sensible"), testAAD())
	if err != nil {
		t.Fatal(err)
	}

	for index := range sealed {
		tampered := bytes.Clone(sealed)
		tampered[index] ^= 0x01
		if _, err := sealer.Open(tampered, testAAD()); err == nil {
			t.Fatalf("l'altération de l'octet %d n'a pas été détectée", index)
		}
	}
}

func TestOpenRejectsWrongContext(t *testing.T) {
	sealer := newTestSealer(t)
	sealed, err := sealer.Seal([]byte("donnée sensible"), testAAD())
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]AAD{
		"autre table":   {Table: "hero_profiles", Column: "last_name", RecordID: testAAD().RecordID},
		"autre colonne": {Table: "users", Column: "first_name", RecordID: testAAD().RecordID},
		"autre ligne":   {Table: "users", Column: "last_name", RecordID: uuid.MustParse("0192f4a0-0000-7000-8000-000000000002")},
		// "us"+"ers_last" et "users"+"_last" donneraient la même concaténation sans préfixe de longueur.
		"frontière déplacée": {Table: "users_", Column: "last_name"[1:], RecordID: testAAD().RecordID},
	}
	for name, aad := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := sealer.Open(sealed, aad); !errors.Is(err, ErrDecryption) {
				t.Fatalf("erreur = %v, attendu ErrDecryption", err)
			}
		})
	}
}

func TestOpenRejectsMalformedInput(t *testing.T) {
	sealer := newTestSealer(t)
	valid, _ := sealer.Seal([]byte("x"), testAAD())

	tests := map[string][]byte{
		"nil":                nil,
		"trop court":         {formatVersion, algorithmCascade},
		"version inconnue":   append([]byte{99}, valid[1:]...),
		"algorithme inconnu": append([]byte{formatVersion, 99}, valid[2:]...),
		"tronqué":            valid[:Overhead-1],
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := sealer.Open(input, testAAD()); !errors.Is(err, ErrInvalidFormat) {
				t.Fatalf("erreur = %v, attendu ErrInvalidFormat", err)
			}
		})
	}
}

func TestOpenWithWrongOrUnknownKey(t *testing.T) {
	sealer := newTestSealer(t)
	sealed, _ := sealer.Seal([]byte("secret"), testAAD())

	otherKey, err := NewSealer(map[uint16][]byte{1: testKey(0x22)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherKey.Open(sealed, testAAD()); !errors.Is(err, ErrDecryption) {
		t.Errorf("mauvaise clé : erreur = %v, attendu ErrDecryption", err)
	}

	otherID, err := NewSealer(map[uint16][]byte{2: testKey(0x11)}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherID.Open(sealed, testAAD()); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("clé inconnue : erreur = %v, attendu ErrUnknownKey", err)
	}
}

func TestKeyRotation(t *testing.T) {
	oldSealer, err := NewSealer(map[uint16][]byte{1: testKey(0x11)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := oldSealer.Seal([]byte("ancienne valeur"), testAAD())

	rotated, err := NewSealer(map[uint16][]byte{1: testKey(0x11), 2: testKey(0x33)}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := rotated.Open(legacy, testAAD()); err != nil || string(got) != "ancienne valeur" {
		t.Fatalf("une ancienne valeur doit rester lisible : (%q, %v)", got, err)
	}
	if !rotated.NeedsRotation(legacy) {
		t.Error("une valeur chiffrée avec l'ancienne clé doit être signalée")
	}

	fresh, _ := rotated.Seal([]byte("nouvelle valeur"), testAAD())
	if id, _ := KeyIDOf(fresh); id != 2 {
		t.Errorf("la nouvelle valeur utilise la clé %d, attendu 2", id)
	}
	if rotated.NeedsRotation(fresh) {
		t.Error("une valeur à jour ne doit pas être signalée")
	}
	if rotated.ActiveKeyID() != 2 {
		t.Errorf("clé active = %d", rotated.ActiveKeyID())
	}
}

func TestNewSealerValidation(t *testing.T) {
	tests := map[string]struct {
		keys   map[uint16][]byte
		active uint16
	}{
		"aucune clé":         {map[uint16][]byte{}, 1},
		"clé active absente": {map[uint16][]byte{1: testKey(1)}, 2},
		"clé trop courte":    {map[uint16][]byte{1: []byte("courte")}, 1},
		"clé trop longue":    {map[uint16][]byte{1: append(testKey(1), 0)}, 1},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewSealer(test.keys, test.active); err == nil {
				t.Fatal("NewSealer devrait échouer")
			}
		})
	}
}

func TestDerivedKeysAreIndependent(t *testing.T) {
	aesKey, xchachaKey, err := deriveKeys(1, testKey(0x11))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(aesKey, xchachaKey) {
		t.Fatal("les deux couches ne doivent jamais partager la même clé")
	}
	otherAES, otherXChaCha, _ := deriveKeys(2, testKey(0x11))
	if bytes.Equal(aesKey, otherAES) || bytes.Equal(xchachaKey, otherXChaCha) {
		t.Fatal("la dérivation doit dépendre de l'identifiant de clé")
	}
	if len(aesKey) != 32 || len(xchachaKey) != 32 {
		t.Fatalf("tailles dérivées = %d et %d, attendu 32", len(aesKey), len(xchachaKey))
	}
}

// counterReader produit un flux d'octets prévisible pour figer le format dans un test.
type counterReader struct{ next byte }

func (r *counterReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.next
		r.next++
	}
	return len(p), nil
}

// TestGoldenVector protège contre toute modification involontaire du format, de la dérivation
// des clés ou des algorithmes : des données déjà en base deviendraient illisibles.
// Si ce test échoue après un changement volontaire, il faut migrer les données existantes.
func TestGoldenVector(t *testing.T) {
	const want = "0101000100010203040506070809" +
		"0a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223"

	sealer, err := newSealer(map[uint16][]byte{1: testKey(0x11)}, 1, &counterReader{})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := sealer.Seal([]byte("Rogers"), testAAD())
	if err != nil {
		t.Fatal(err)
	}

	got := hex.EncodeToString(sealed)
	// L'en-tête et les nonces sont déterministes ; le reste dépend des clés dérivées.
	if got[:len(want)] != want {
		t.Fatalf("en-tête ou nonces modifiés :\n got %s\nwant %s", got[:len(want)], want)
	}
	if got != goldenCiphertext {
		t.Fatalf("le format ou la dérivation a changé :\n got %s\nwant %s", got, goldenCiphertext)
	}

	opened, err := sealer.Open(sealed, testAAD())
	if err != nil || string(opened) != "Rogers" {
		t.Fatalf("Open = (%q, %v)", opened, err)
	}
}

// goldenCiphertext est le chiffré de référence de "Rogers" avec la clé maîtresse 0x11 répétée,
// le contexte de testAAD et le flux d'octets de counterReader.
const goldenCiphertext = "01010001000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223a4553490887a3e1bfc5dfdf2a846af8db3922d4bf2effe170d279fec8d8af1ad8dfb1a497cf0"

func TestConcurrentUse(t *testing.T) {
	sealer := newTestSealer(t)
	const goroutines, iterations = 32, 100

	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for worker := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			aad := AAD{Table: "users", Column: "city", RecordID: uuid.New()}
			plaintext := bytes.Repeat([]byte{byte(worker)}, 64)
			for range iterations {
				sealed, err := sealer.Seal(plaintext, aad)
				if err != nil {
					errs <- err
					return
				}
				opened, err := sealer.Open(sealed, aad)
				if err != nil || !bytes.Equal(opened, plaintext) {
					errs <- errors.Join(errors.New("aller-retour incorrect"), err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
