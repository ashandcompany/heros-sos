package herodashboard

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
)

var testStart = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

func newTestCache(ttl time.Duration) (*snapshotCache[int], *clock.Fake) {
	clk := clock.NewFake(testStart)
	return newSnapshotCache[int](ttl, clk, slog.New(slog.DiscardHandler)), clk
}

// counter retourne un chargeur qui compte ses appels et renvoie le numéro de l'appel.
func counter(calls *atomic.Int64) func(context.Context) (int, error) {
	return func(context.Context) (int, error) { return int(calls.Add(1)), nil }
}

// prime remplit le cache d'une clé et échoue si le premier calcul échoue.
func prime(t *testing.T, cache *snapshotCache[int], calls *atomic.Int64) {
	t.Helper()
	if _, err := cache.get(context.Background(), "k", counter(calls)); err != nil {
		t.Fatal(err)
	}
}

func eventually(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("délai dépassé : %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWithoutATTLEveryCallComputes(t *testing.T) {
	cache, _ := newTestCache(0)
	var calls atomic.Int64
	for want := 1; want <= 3; want++ {
		if got, err := cache.get(context.Background(), "k", counter(&calls)); err != nil || got != want {
			t.Fatalf("appel %d : %d, %v", want, got, err)
		}
	}
}

func TestAFreshValueIsServedWithoutComputing(t *testing.T) {
	cache, clk := newTestCache(30 * time.Second)
	var calls atomic.Int64
	prime(t, cache, &calls)
	clk.Advance(29 * time.Second)
	if got, _ := cache.get(context.Background(), "k", counter(&calls)); got != 1 || calls.Load() != 1 {
		t.Fatalf("valeur %d après %d calculs : une valeur fraîche ne se recalcule pas", got, calls.Load())
	}
	// Une autre clé a sa propre valeur.
	if got, _ := cache.get(context.Background(), "other", counter(&calls)); got != 2 {
		t.Fatalf("autre clé : %d", got)
	}
}

// Une valeur périmée est servie tout de suite, jamais après l'attente d'un recalcul : celui-ci se fait en arrière-plan.
func TestAStaleValueIsServedImmediatelyWhileItRefreshesInTheBackground(t *testing.T) {
	cache, clk := newTestCache(30 * time.Second)
	var calls atomic.Int64
	prime(t, cache, &calls)
	clk.Advance(time.Minute)

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	slow := func(context.Context) (int, error) {
		started <- struct{}{}
		<-release
		return 100, nil
	}
	if got, err := cache.get(context.Background(), "k", slow); err != nil || got != 1 {
		t.Fatalf("la valeur périmée doit être servie sans attendre : %d, %v", got, err)
	}
	<-started
	// Pendant le recalcul, les appels suivants ne lancent pas un second calcul.
	for range 5 {
		if got, _ := cache.get(context.Background(), "k", slow); got != 1 {
			t.Fatalf("pendant le recalcul : %d", got)
		}
	}
	select {
	case <-started:
		t.Fatal("un second calcul a été lancé pour la même clé")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	eventually(t, "la nouvelle valeur est servie", func() bool {
		got, _ := cache.get(context.Background(), "k", slow)
		return got == 100
	})
}

func TestAFailedRefreshKeepsTheOldValueAndTheNextCallRetries(t *testing.T) {
	cache, clk := newTestCache(30 * time.Second)
	var calls atomic.Int64
	prime(t, cache, &calls)
	clk.Advance(time.Minute)

	var attempts atomic.Int64
	failing := func(context.Context) (int, error) {
		attempts.Add(1)
		return 0, errors.New("base indisponible")
	}
	if got, _ := cache.get(context.Background(), "k", failing); got != 1 {
		t.Fatalf("valeur servie = %d", got)
	}
	eventually(t, "le premier recalcul échoue", func() bool { return attempts.Load() == 1 })
	// L'échec n'est ni servi ni mis en cache : l'ancienne valeur reste, et un nouvel appel réessaie.
	eventually(t, "un nouvel appel relance le calcul", func() bool {
		got, err := cache.get(context.Background(), "k", counter(&calls))
		return err == nil && got == 2 && calls.Load() == 2
	})
}

func TestConcurrentFirstCallsShareOneComputation(t *testing.T) {
	cache, _ := newTestCache(time.Minute)
	var calls atomic.Int64
	release := make(chan struct{})
	load := func(context.Context) (int, error) {
		calls.Add(1)
		<-release
		return 7, nil
	}
	var wg sync.WaitGroup
	results := make(chan int, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := cache.get(context.Background(), "k", load)
			if err != nil {
				t.Error(err)
			}
			results <- got
		}()
	}
	eventually(t, "le calcul démarre", func() bool { return calls.Load() >= 1 })
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	close(results)
	for got := range results {
		if got != 7 {
			t.Errorf("résultat = %d", got)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("%d calculs pour 10 appels simultanés, attendu 1", calls.Load())
	}
}

// Un appelant qui abandonne sa requête n'interrompt pas le calcul : le suivant en profite.
func TestAnAbandonedFirstCallStillFillsTheCache(t *testing.T) {
	cache, _ := newTestCache(time.Minute)
	release := make(chan struct{})
	var calls atomic.Int64
	load := func(context.Context) (int, error) {
		calls.Add(1)
		<-release
		return 9, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := cache.get(ctx, "k", load)
		done <- err
	}()
	eventually(t, "le calcul démarre", func() bool { return calls.Load() == 1 })
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("l'appelant abandonné doit recevoir context.Canceled : %v", err)
	}
	close(release)
	eventually(t, "la valeur arrive dans le cache", func() bool {
		got, err := cache.get(context.Background(), "k", load)
		return err == nil && got == 9
	})
	if calls.Load() != 1 {
		t.Errorf("%d calculs, attendu 1", calls.Load())
	}
}

// Un contexte déjà annulé ne lance pas de calcul : il serait détaché de la requête et continuerait pour personne.
func TestACancelledContextDoesNotStartAComputation(t *testing.T) {
	cache, _ := newTestCache(time.Minute)
	var calls atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.get(ctx, "k", counter(&calls)); !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur : %v, attendu context.Canceled", err)
	}
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 0 {
		t.Errorf("%d calculs lancés pour un contexte annulé", calls.Load())
	}
}
