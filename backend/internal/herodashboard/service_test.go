package herodashboard

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
)

// fakeRepo compte ses appels et peut ralentir ou bloquer le calcul des statistiques d'un héros.
type fakeRepo struct {
	heroes                                []uuid.UUID
	globalCalls, personalCalls, enemyCall atomic.Int64
	running, maxRunning                   atomic.Int64
	delay                                 time.Duration
	failFor                               uuid.UUID
	// gate, s'il existe, bloque Personal jusqu'à sa fermeture.
	gate chan struct{}
	// started reçoit un signal quand un calcul Personal commence.
	started chan struct{}
}

func (r *fakeRepo) Global(context.Context) (Global, error) {
	r.globalCalls.Add(1)
	return Global{Interventions: 7}, nil
}

func (r *fakeRepo) Personal(ctx context.Context, heroID uuid.UUID) (Personal, error) {
	r.personalCalls.Add(1)
	now := r.running.Add(1)
	defer r.running.Add(-1)
	for {
		peak := r.maxRunning.Load()
		if now <= peak || r.maxRunning.CompareAndSwap(peak, now) {
			break
		}
	}
	if r.started != nil {
		r.started <- struct{}{}
	}
	if r.gate != nil {
		<-r.gate
	}
	time.Sleep(r.delay)
	if heroID == r.failFor {
		return Personal{}, errors.New("base indisponible")
	}
	return Personal{Interventions: 3}, nil
}

func (r *fakeRepo) MainEnemies(context.Context, uuid.UUID, int) ([]Enemy, error) {
	r.enemyCall.Add(1)
	return []Enemy{}, nil
}

func (r *fakeRepo) HeroIDs(context.Context) ([]uuid.UUID, error) { return r.heroes, nil }

type noIncidents struct{}

func (noIncidents) List(context.Context, incident.ListFilter, int, int) (incident.ListResult, error) {
	return incident.ListResult{}, nil
}

func heroIDs(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.New()
	}
	return ids
}

func newTestService(repo Repository, ttl time.Duration) *Service {
	return NewService(repo, noIncidents{}, Options{
		StatsTTL: ttl, Clock: clock.NewFake(testStart), Logger: slog.New(slog.DiscardHandler),
	})
}

func TestWarmFillsTheCacheSoThatNoRequestComputes(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(3)}
	service := newTestService(repo, time.Minute)

	report, err := service.Warm(context.Background())
	if err != nil || report.Heroes != 3 || report.Failed != 0 {
		t.Fatalf("rapport : %+v, %v", report, err)
	}
	if repo.globalCalls.Load() != 1 || repo.personalCalls.Load() != 3 || repo.enemyCall.Load() != 3 {
		t.Fatalf("calculs du préchauffage : global %d, héros %d, ennemis %d", repo.globalCalls.Load(), repo.personalCalls.Load(), repo.enemyCall.Load())
	}
	for _, heroID := range repo.heroes {
		dashboard, err := service.Get(context.Background(), heroID)
		if err != nil || dashboard.Global.Interventions != 7 || dashboard.Personal.Interventions != 3 {
			t.Fatalf("tableau de bord : %+v, %v", dashboard, err)
		}
	}
	if repo.globalCalls.Load() != 1 || repo.personalCalls.Load() != 3 {
		t.Errorf("une requête après le préchauffage a recalculé : global %d, héros %d", repo.globalCalls.Load(), repo.personalCalls.Load())
	}
}

func TestWarmDoesNothingWhenTheCacheIsDisabled(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(3)}
	report, err := newTestService(repo, 0).Warm(context.Background())
	if err != nil || report != (WarmReport{}) {
		t.Fatalf("rapport : %+v, %v", report, err)
	}
	if repo.globalCalls.Load() != 0 || repo.personalCalls.Load() != 0 {
		t.Error("sans cache, le préchauffage ne calcule rien : il serait perdu")
	}
}

func TestWarmBoundsTheNumberOfSimultaneousComputations(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(8), delay: 30 * time.Millisecond}
	report, err := newTestService(repo, time.Minute).Warm(context.Background())
	if err != nil || report.Heroes != 8 {
		t.Fatalf("rapport : %+v, %v", report, err)
	}
	if got := repo.maxRunning.Load(); got < 1 || got > warmConcurrency {
		t.Errorf("%d calculs simultanés, au plus %d autorisés", got, warmConcurrency)
	}
}

func TestOneFailingHeroDoesNotStopTheOthers(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(3)}
	repo.failFor = repo.heroes[1]
	service := newTestService(repo, time.Minute)

	report, err := service.Warm(context.Background())
	if err != nil || report.Heroes != 2 || report.Failed != 1 {
		t.Fatalf("rapport : %+v, %v", report, err)
	}
	before := repo.personalCalls.Load()
	for _, index := range []int{0, 2} {
		if _, err := service.Get(context.Background(), repo.heroes[index]); err != nil {
			t.Fatal(err)
		}
	}
	if repo.personalCalls.Load() != before {
		t.Error("les héros préchauffés ne doivent pas être recalculés")
	}
}

// Un héros qui ouvre son tableau de bord pendant le préchauffage partage le calcul en cours : un seul calcul.
func TestARequestDuringWarmupSharesTheRunningComputation(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(1), gate: make(chan struct{}), started: make(chan struct{}, 4)}
	service := newTestService(repo, time.Minute)
	warmed := make(chan WarmReport, 1)
	go func() {
		report, _ := service.Warm(context.Background())
		warmed <- report
	}()
	select {
	case <-repo.started:
	case <-time.After(2 * time.Second):
		t.Fatal("le calcul du héros ne démarre pas")
	}

	served := make(chan error, 1)
	go func() {
		_, err := service.Get(context.Background(), repo.heroes[0])
		served <- err
	}()
	time.Sleep(50 * time.Millisecond)
	close(repo.gate)

	if err := <-served; err != nil {
		t.Fatal(err)
	}
	if report := <-warmed; report.Heroes != 1 {
		t.Fatalf("rapport : %+v", report)
	}
	if got := repo.personalCalls.Load(); got != 1 {
		t.Errorf("%d calculs des statistiques du héros, attendu 1 (partagé)", got)
	}
}

func TestWarmStopsWhenTheContextIsCancelled(t *testing.T) {
	repo := &fakeRepo{heroes: heroIDs(6), gate: make(chan struct{}), started: make(chan struct{}, 8)}
	t.Cleanup(func() { close(repo.gate) })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := newTestService(repo, time.Minute).Warm(ctx)
		done <- err
	}()
	<-repo.started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("erreur : %v, attendu context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("le préchauffage ne s'arrête pas à l'annulation")
	}
	if got := repo.personalCalls.Load(); got > warmConcurrency {
		t.Errorf("%d calculs lancés après l'annulation", got)
	}
}
