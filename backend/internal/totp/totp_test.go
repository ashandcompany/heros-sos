package totp

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// Vecteurs de test de la RFC 4226, annexe D : le secret ASCII « 12345678901234567890 ».
func TestCodeMatchesRFC4226TestVectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	want := []string{"755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"}
	for counter, code := range want {
		if got := Code(secret, int64(counter)); got != code {
			t.Errorf("HOTP(%d) = %s, attendu %s", counter, got, code)
		}
	}
}

// Vecteurs de la RFC 6238, annexe B, SHA-1 : leurs codes à 8 chiffres se terminent par nos 6 chiffres.
func TestCodeMatchesRFC6238TestVectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	for unix, want := range map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037", 20000000000: "353130"} {
		if got := Code(secret, Step(time.Unix(unix, 0))); got != want {
			t.Errorf("TOTP(%d) = %s, attendu %s", unix, got, want)
		}
	}
}

func TestVerifyAcceptsTheCurrentStepAndItsNeighboursOnly(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1_700_000_000, 0)
	current := Step(now)

	for offset := int64(-Skew); offset <= Skew; offset++ {
		step, ok := Verify(secret, Code(secret, current+offset), now, 0)
		if !ok || step != current+offset {
			t.Errorf("pas %+d : step=%d ok=%t", offset, step, ok)
		}
	}
	for _, offset := range []int64{-Skew - 1, Skew + 1, -10, 10} {
		if _, ok := Verify(secret, Code(secret, current+offset), now, 0); ok {
			t.Errorf("le pas %+d est hors de la fenêtre", offset)
		}
	}
}

// Un code ne sert qu'une fois : un code déjà accepté (ou plus ancien) est refusé même dans la fenêtre.
func TestVerifyRefusesAReplayedCode(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1_700_000_000, 0)
	step, ok := Verify(secret, Code(secret, Step(now)), now, 0)
	if !ok {
		t.Fatal("le premier usage doit réussir")
	}
	if _, ok := Verify(secret, Code(secret, Step(now)), now, step); ok {
		t.Error("le même code rejoué doit être refusé")
	}
	// Un code plus ancien que le dernier accepté l'est aussi, même s'il est encore dans la fenêtre.
	if _, ok := Verify(secret, Code(secret, Step(now)-1), now, step); ok {
		t.Error("un code antérieur au dernier accepté doit être refusé")
	}
	// Le code du pas suivant, en revanche, passe : on ne bloque pas le prochain code légitime.
	if got, ok := Verify(secret, Code(secret, Step(now)+1), now, step); !ok || got != step+1 {
		t.Errorf("le pas suivant doit passer : %d %t", got, ok)
	}
}

func TestVerifyRefusesMalformedCodes(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1_700_000_000, 0)
	good := Code(secret, Step(now))

	for name, code := range map[string]string{
		"vide": "", "trop court": good[:5], "trop long": good + "0", "lettres": "12a456", "signe": "+12345", "hexa": "0x1234",
	} {
		if _, ok := Verify(secret, code, now, 0); ok {
			t.Errorf("%s (%q) doit être refusé", name, code)
		}
	}
	// Les espaces d'une saisie en groupes de trois chiffres sont tolérés.
	if _, ok := Verify(secret, good[:3]+" "+good[3:], now, 0); !ok {
		t.Error("« 123 456 » doit être accepté")
	}
}

func TestSecretIsRandomAndEncodesToBase32(t *testing.T) {
	first, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	second, _ := NewSecret()
	if len(first) != SecretLength || string(first) == string(second) {
		t.Fatalf("secrets : %x %x", first, second)
	}
	encoded := EncodeSecret(first)
	if len(encoded) != 32 || strings.ContainsAny(encoded, "=0189 ") || encoded != strings.ToUpper(encoded) {
		t.Errorf("secret encodé = %q (32 caractères base32 majuscules, sans bourrage)", encoded)
	}
}

func TestURLFollowsTheKeyURIFormat(t *testing.T) {
	secret := []byte("12345678901234567890")
	raw := URL("Heros SOS", "nick.fury@shield.example", secret)

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
		t.Errorf("adresse = %q", raw)
	}
	if label := strings.TrimPrefix(parsed.Path, "/"); label != "Heros SOS:nick.fury@shield.example" {
		t.Errorf("étiquette = %q", label)
	}
	query := parsed.Query()
	if query.Get("secret") != "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" || query.Get("issuer") != "Heros SOS" ||
		query.Get("algorithm") != "SHA1" || query.Get("digits") != "6" || query.Get("period") != "30" {
		t.Errorf("paramètres = %v", query)
	}
	// Un nom qui contient « : » ou un espace ne casse pas l'étiquette.
	if got := URL("A:B", "x y@z.io", secret); strings.Count(strings.TrimPrefix(strings.SplitN(got, "?", 2)[0], "otpauth://totp/"), ":") != 1 {
		t.Errorf("l'étiquette ne doit contenir qu'un séparateur : %q", got)
	}
}
