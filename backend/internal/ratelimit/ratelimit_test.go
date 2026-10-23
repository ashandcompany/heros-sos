package ratelimit_test

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
)

var start = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)

func newLimiter(limit int, window time.Duration, maxKeys int) (*ratelimit.Limiter, *clock.Fake) {
	clk := clock.NewFake(start)
	return ratelimit.New(config.RateRule{Max: limit, Window: window}, maxKeys, clk), clk
}

func TestAllowsUpToMaxThenDenies(t *testing.T) {
	limiter, clk := newLimiter(3, time.Minute, 100)

	for i := 1; i <= 3; i++ {
		if allowed, _ := limiter.Allow("a"); !allowed {
			t.Fatalf("requête %d refusée", i)
		}
	}

	clk.Advance(20 * time.Second)
	allowed, retryAfter := limiter.Allow("a")
	if allowed {
		t.Fatal("la quatrième requête doit être refusée")
	}
	if retryAfter != 40*time.Second {
		t.Errorf("délai = %v, attendu 40s", retryAfter)
	}
}

func TestWindowExpiryRestoresQuota(t *testing.T) {
	limiter, clk := newLimiter(1, time.Minute, 100)
	limiter.Allow("a")
	if allowed, _ := limiter.Allow("a"); allowed {
		t.Fatal("la deuxième requête doit être refusée")
	}

	clk.Advance(time.Minute)
	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("une nouvelle fenêtre doit rétablir le quota")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	limiter, _ := newLimiter(1, time.Minute, 100)
	limiter.Allow("a")
	if allowed, _ := limiter.Allow("b"); !allowed {
		t.Fatal("la clé b ne doit pas être affectée par la clé a")
	}
}

func TestResetForgetsKey(t *testing.T) {
	limiter, _ := newLimiter(1, time.Minute, 100)
	limiter.Allow("a")
	limiter.Reset("a")
	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("après Reset, la clé doit repartir de zéro")
	}
}

func TestSweepReleasesExpiredKeys(t *testing.T) {
	limiter, clk := newLimiter(1, time.Minute, 100)
	for i := range 10 {
		limiter.Allow(strconv.Itoa(i))
	}
	clk.Advance(30 * time.Second)
	limiter.Allow("récente")

	clk.Advance(31 * time.Second)
	limiter.Sweep()

	if got := limiter.Len(); got != 1 {
		t.Fatalf("clés suivies = %d, attendu 1 (seule la plus récente reste)", got)
	}
}

func TestFullTableFailsClosedForNewKeysOnly(t *testing.T) {
	limiter, clk := newLimiter(5, time.Minute, 2)
	limiter.Allow("a")
	limiter.Allow("b")

	if allowed, _ := limiter.Allow("c"); allowed {
		t.Fatal("une clé nouvelle doit être refusée quand la table est pleine")
	}
	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("une clé déjà suivie reste servie")
	}

	clk.Advance(time.Minute)
	if allowed, _ := limiter.Allow("c"); !allowed {
		t.Fatal("après expiration, de la place est libérée pour les nouvelles clés")
	}
	if got := limiter.Len(); got > 2 {
		t.Fatalf("la table dépasse sa borne : %d", got)
	}
}

func TestConcurrentCallsNeverExceedLimit(t *testing.T) {
	const limit, callers = 50, 400
	limiter, _ := newLimiter(limit, time.Hour, 100)

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := limiter.Allow("même-clé"); ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := allowed.Load(); got != limit {
		t.Fatalf("requêtes acceptées = %d, attendu exactement %d", got, limit)
	}
}

func TestConcurrentMixedOperationsAreRaceFree(t *testing.T) {
	limiter, clk := newLimiter(5, time.Second, 64)

	var wg sync.WaitGroup
	for worker := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				key := strconv.Itoa((worker + i) % 32)
				limiter.Allow(key)
				switch i % 20 {
				case 0:
					limiter.Reset(key)
				case 10:
					limiter.Sweep()
				case 15:
					clk.Advance(time.Millisecond * 100)
				}
			}
		}()
	}
	wg.Wait()

	if limiter.Len() > 64 {
		t.Fatalf("la table dépasse sa borne : %d", limiter.Len())
	}
}
