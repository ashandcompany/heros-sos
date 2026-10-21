package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

func testArgon2() config.Argon2 {
	return config.Argon2{Time: 1, MemoryKiB: 64, Threads: 1, KeyLength: 32, SaltLength: 16, MaxConcurrent: 2}
}

func TestPasswordHashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2())
	ctx := context.Background()

	hash, err := hasher.Hash(ctx, "Correct Horse 42")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("format PHC inattendu : %s", hash)
	}
	if strings.Contains(hash, "Correct") {
		t.Fatal("l'empreinte ne doit pas contenir le mot de passe")
	}

	check, err := hasher.Verify(ctx, hash, "Correct Horse 42")
	if err != nil || !check.Match || check.NeedsRehash {
		t.Fatalf("bon mot de passe : (%+v, %v)", check, err)
	}
	for _, wrong := range []string{"correct horse 42", "Correct Horse 43", "", " Correct Horse 42"} {
		if check, err := hasher.Verify(ctx, hash, wrong); err != nil || check.Match {
			t.Errorf("mauvais mot de passe %q accepté : (%+v, %v)", wrong, check, err)
		}
	}
}

func TestPasswordHashesAreSalted(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2())
	first, _ := hasher.Hash(context.Background(), "Meme Mot De Passe 1")
	second, _ := hasher.Hash(context.Background(), "Meme Mot De Passe 1")
	if first == second {
		t.Fatal("deux empreintes du même mot de passe doivent différer (sel aléatoire)")
	}
}

func TestPasswordNeedsRehashWhenParametersChange(t *testing.T) {
	weak := NewPasswordHasher(testArgon2())
	hash, _ := weak.Hash(context.Background(), "Mot De Passe 1234")

	stronger := testArgon2()
	stronger.Time = 2
	check, err := NewPasswordHasher(stronger).Verify(context.Background(), hash, "Mot De Passe 1234")
	if err != nil || !check.Match || !check.NeedsRehash {
		t.Fatalf("(%+v, %v) : le mot de passe doit être valide ET à rehacher", check, err)
	}
}

func TestPasswordVerifyRejectsMalformedHashes(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2())
	for name, encoded := range map[string]string{
		"vide":               "",
		"texte libre":        "motdepasse",
		"mauvais algorithme": "$argon2i$v=19$m=64,t=1,p=1$c2VsMTIzNDU2Nzg5MA$Y2zDqQ",
		"version inconnue":   "$argon2id$v=16$m=64,t=1,p=1$c2VsMTIzNDU2Nzg5MA$Y2zDqQ",
		"paramètres absents": "$argon2id$v=19$$c2VsMTIzNDU2Nzg5MA$Y2zDqQ",
		"paramètre nul":      "$argon2id$v=19$m=0,t=1,p=1$c2VsMTIzNDU2Nzg5MA$Y2zDqQ",
		"sel invalide":       "$argon2id$v=19$m=64,t=1,p=1$!!!$Y2zDqQ",
		"empreinte vide":     "$argon2id$v=19$m=64,t=1,p=1$c2VsMTIzNDU2Nzg5MA$",
		"champ manquant":     "$argon2id$v=19$m=64,t=1,p=1$c2VsMTIzNDU2Nzg5MA",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := hasher.Verify(context.Background(), encoded, "x"); !errors.Is(err, ErrInvalidHash) {
				t.Fatalf("erreur = %v, attendu ErrInvalidHash", err)
			}
		})
	}
}

func TestVerifyDummyNeverMatches(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2())
	if err := hasher.VerifyDummy(context.Background(), dummyPassword); err != nil {
		t.Fatalf("VerifyDummy : %v", err)
	}
	if err := hasher.VerifyDummy(context.Background(), "autre"); err != nil {
		t.Fatalf("VerifyDummy : %v", err)
	}
}

func TestHasherCapsConcurrentComputations(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2()) // 2 emplacements

	releaseFirst, err := hasher.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	releaseSecond, err := hasher.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := hasher.acquire(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("un troisième calcul doit attendre puis abandonner : %v", err)
	}

	releaseFirst()
	release, err := hasher.acquire(context.Background())
	if err != nil {
		t.Fatalf("un emplacement libéré doit être réutilisable : %v", err)
	}
	release()
	releaseSecond()
}

func TestHasherHonoursCancelledContext(t *testing.T) {
	params := testArgon2()
	params.MaxConcurrent = 1
	hasher := NewPasswordHasher(params)
	release, _ := hasher.acquire(context.Background())
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := hasher.Hash(ctx, "Mot De Passe 1234"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Hash avec contexte annulé : %v", err)
	}
}

func TestHasherIsSafeForConcurrentUse(t *testing.T) {
	hasher := NewPasswordHasher(testArgon2())
	hash, _ := hasher.Hash(context.Background(), "Mot De Passe 1234")

	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			check, err := hasher.Verify(context.Background(), hash, "Mot De Passe 1234")
			if err == nil && !check.Match {
				err = errors.New("le bon mot de passe a été refusé")
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestNewTokenIsRandomWellFormedAndHashed(t *testing.T) {
	seen := make(map[string]bool)
	for range 500 {
		raw, hash, err := newToken()
		if err != nil {
			t.Fatal(err)
		}
		if !isWellFormedToken(raw) {
			t.Fatalf("jeton mal formé : %q", raw)
		}
		if len(hash) != 32 || string(hash) != string(hashToken(raw)) {
			t.Fatal("l'empreinte doit être le SHA-256 déterministe du jeton")
		}
		if seen[raw] {
			t.Fatalf("jeton en double : %q", raw)
		}
		seen[raw] = true
	}
}

func TestIsWellFormedToken(t *testing.T) {
	valid, _, _ := newToken()
	tests := map[string]bool{
		valid:                          true,
		"":                             false,
		valid[:42]:                     false,
		valid + "A":                    false,
		strings.Repeat("!", 43):        false,
		strings.Repeat("A", 43) + "\n": false,
		strings.Repeat("a", 5000):      false,
	}
	for token, want := range tests {
		if got := isWellFormedToken(token); got != want {
			t.Errorf("isWellFormedToken(%.20q) = %v, attendu %v", token, got, want)
		}
	}
}

func TestCSRFTokenIsBoundToSession(t *testing.T) {
	csrf := NewCSRF([]byte("secret-serveur-de-test-de-32-octets"))

	first := csrf.Token("session-A")
	if first != csrf.Token("session-A") {
		t.Fatal("le jeton doit être déterministe pour une session")
	}
	if first == csrf.Token("session-B") {
		t.Fatal("deux sessions ne doivent pas partager de jeton")
	}
	if !csrf.Valid("session-A", first) {
		t.Fatal("le jeton de la session doit être accepté")
	}
	if csrf.Valid("session-B", first) {
		t.Error("le jeton d'une autre session doit être refusé")
	}
	if csrf.Valid("session-A", first+"x") || csrf.Valid("session-A", "") || csrf.Valid("", first) {
		t.Error("jeton altéré ou vide accepté")
	}
	if NewCSRF([]byte("un-autre-secret-serveur-de-test!!")).Valid("session-A", first) {
		t.Error("un autre secret ne doit pas valider le jeton")
	}
}

func TestCSRFDoesNotKeepCallerSecret(t *testing.T) {
	secret := []byte("secret-serveur-de-test-de-32-octets")
	csrf := NewCSRF(secret)
	before := csrf.Token("s")
	secret[0] ^= 0xff
	if csrf.Token("s") != before {
		t.Fatal("modifier le tableau d'origine ne doit pas changer le secret du générateur")
	}
}

func TestCookieAttributes(t *testing.T) {
	now := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	expires := now.Add(2 * time.Hour)

	for _, secure := range []bool{true, false} {
		manager := NewCookieManager(config.Session{CookieSecure: secure})
		recorder := httptest.NewRecorder()
		manager.Set(recorder, "jeton", expires, now)

		cookie := recorder.Result().Cookies()[0]
		wantName := "session"
		if secure {
			wantName = "__Host-session"
		}
		if cookie.Name != wantName || manager.Name() != wantName {
			t.Errorf("secure=%v : nom = %q", secure, cookie.Name)
		}
		if !cookie.HttpOnly || cookie.Secure != secure || cookie.SameSite != http.SameSiteLaxMode ||
			cookie.Path != "/" || cookie.Domain != "" {
			t.Errorf("secure=%v : attributs = %+v", secure, cookie)
		}
		if cookie.MaxAge != 7200 || !cookie.Expires.Equal(expires) {
			t.Errorf("secure=%v : durée = %d, expiration = %v", secure, cookie.MaxAge, cookie.Expires)
		}

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(cookie)
		if token, ok := manager.Read(request); !ok || token != "jeton" {
			t.Errorf("Read = (%q, %v)", token, ok)
		}
	}
}

func TestCookieClearAndReadAbsent(t *testing.T) {
	manager := NewCookieManager(config.Session{CookieSecure: true})
	recorder := httptest.NewRecorder()
	manager.Clear(recorder)

	cookie := recorder.Result().Cookies()[0]
	if cookie.MaxAge >= 0 || cookie.Value != "" || !cookie.HttpOnly || !cookie.Secure {
		t.Errorf("le cookie effacé doit expirer et garder ses attributs de sécurité : %+v", cookie)
	}
	if _, ok := manager.Read(httptest.NewRequest(http.MethodGet, "/", nil)); ok {
		t.Error("Read sans cookie doit retourner false")
	}
}

func TestDeviceLabel(t *testing.T) {
	tests := []struct{ name, userAgent, want string }{
		{"Chrome macOS", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36", "Chrome · macOS"},
		{"Safari iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1", "Safari · iOS"},
		{"Firefox Linux", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0", "Firefox · Linux"},
		{"Edge Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0", "Edge · Windows"},
		{"Chrome Android", "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36", "Chrome · Android"},
		{"Chrome iOS", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0.0.0 Mobile/15E148 Safari/604.1", "Chrome · iOS"},
		{"Opera", "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 OPR/111.0.0.0", "Opera · Windows"},
		{"client HTTP sans navigateur", "curl/8.7.1", unknownDevice},
		{"vide", "", unknownDevice},
		{"système seul", "Dalvik/2.1.0 (Linux; U; Android 14)", "Android"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := deviceLabel(test.userAgent); got != test.want {
				t.Errorf("deviceLabel = %q, attendu %q", got, test.want)
			}
		})
	}
}

func TestDeviceLabelDoesNotKeepRawUserAgent(t *testing.T) {
	label := deviceLabel("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/126.0.6478.127 Safari/537.36")
	if strings.ContainsAny(label, "0123456789/;") {
		t.Errorf("l'étiquette ne doit pas contenir de version ni de détail : %q", label)
	}
}
