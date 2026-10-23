package herodashboard

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
)

const (
	// globalKey est la clé des statistiques de toute la plateforme dans le cache.
	globalKey = "global"
	// warmConcurrency borne les calculs de statistiques de héros menés en même temps par le préchauffage : chacun
	// parcourt les participations d'un héros (de l'ordre de la dizaine de secondes à 5 millions d'incidents), et le
	// préchauffage ne doit pas saturer la base que les premières requêtes utilisent déjà.
	warmConcurrency = 2
	// mainEnemiesLimit borne la liste des vilains les plus souvent affrontés.
	mainEnemiesLimit = 3
	// recentIncidentsLimit borne la liste des derniers incidents non résolus (signalés, en cours ou échoués).
	recentIncidentsLimit = 5
	// ongoingIncidentsLimit borne les incidents en cours du héros. Un héros n'intervient que sur un incident à la
	// fois : la borne protège seulement d'une base incohérente.
	ongoingIncidentsLimit = 10
)

// IncidentLister lit les incidents (implémenté par incident.Service). Le tableau de bord n'affiche aucune
// pagination : il demande à ne pas calculer le total (ListFilter.SkipTotal).
type IncidentLister interface {
	List(ctx context.Context, filter incident.ListFilter, limit, offset int) (incident.ListResult, error)
}

// Options règle le service.
type Options struct {
	// StatsTTL est la fraîcheur des statistiques mises en cache (voir snapshotCache) ; zéro désactive le cache.
	StatsTTL time.Duration
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Service assemble le tableau de bord. Les statistiques (globales, puis celles de chaque héros) passent par un cache
// qui ne fait jamais attendre un recalcul ; les deux listes d'incidents, qui se lisent dans des index, sont toujours
// à jour.
type Service struct {
	repo      Repository
	incidents IncidentLister
	global    *snapshotCache[Global]
	personal  *snapshotCache[Personal]
	logger    *slog.Logger
}

// NewService crée le service.
func NewService(repo Repository, incidents IncidentLister, opts Options) *Service {
	return &Service{
		repo: repo, incidents: incidents, logger: opts.Logger,
		global:   newSnapshotCache[Global](opts.StatsTTL, opts.Clock, opts.Logger),
		personal: newSnapshotCache[Personal](opts.StatsTTL, opts.Clock, opts.Logger),
	}
}

// Dashboard est le tableau de bord d'un héros.
type Dashboard struct {
	Global             Global
	Personal           Personal
	MyOngoingIncidents []incident.Incident
	RecentIncidents    []incident.Incident
}

// Get assemble le tableau de bord du héros. Les lectures sont indépendantes : elles s'exécutent en parallèle, la
// première erreur annule les autres.
func (s *Service) Get(ctx context.Context, heroID uuid.UUID) (Dashboard, error) {
	var dashboard Dashboard
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		dashboard.Global, err = s.global.get(ctx, globalKey, s.repo.Global)
		return err
	})
	group.Go(func() (err error) {
		dashboard.Personal, err = s.personal.get(ctx, heroID.String(), func(ctx context.Context) (Personal, error) {
			return s.loadPersonal(ctx, heroID)
		})
		return err
	})
	group.Go(func() (err error) {
		filter := incident.ListFilter{ActiveHeroID: &heroID, ApprovedOnly: true, SkipTotal: true}
		page, err := s.incidents.List(ctx, filter, ongoingIncidentsLimit, 0)
		dashboard.MyOngoingIncidents = page.Items
		return err
	})
	group.Go(func() (err error) {
		// Un incident échoué redevient disponible (une nouvelle intervention peut le reprendre) : il n'est pas résolu.
		filter := incident.ListFilter{ApprovedOnly: true, SkipTotal: true, Statuses: []incident.Status{
			incident.StatusReported, incident.StatusInProgress, incident.StatusFailed,
		}}
		page, err := s.incidents.List(ctx, filter, recentIncidentsLimit, 0)
		dashboard.RecentIncidents = page.Items
		return err
	})
	if err := group.Wait(); err != nil {
		return Dashboard{}, err
	}
	return dashboard, nil
}

// WarmReport résume un préchauffage.
type WarmReport struct {
	// Heroes est le nombre de héros dont les statistiques sont maintenant en cache, Failed celui des échecs.
	Heroes, Failed int
	// Elapsed est la durée du préchauffage.
	Elapsed time.Duration
}

// Warm calcule d'avance les statistiques globales puis celles de chaque héros, pour que personne n'attende le calcul à
// froid : sans cela, le premier affichage de chaque héros après un démarrage calcule en direct (environ 11 s à
// 5 millions d'incidents). Il passe par les mêmes calculs que Get : une requête qui arrive pendant le préchauffage
// partage le calcul en cours au lieu de le refaire. Il ne fait rien quand le cache est désactivé. L'échec d'un héros
// n'arrête pas les autres : il sera calculé à sa première requête. Warm s'arrête quand ctx est annulé.
func (s *Service) Warm(ctx context.Context) (WarmReport, error) {
	if !s.global.enabled() {
		return WarmReport{}, nil
	}
	started := time.Now()
	if _, err := s.global.get(ctx, globalKey, s.repo.Global); err != nil {
		return WarmReport{}, fmt.Errorf("herodashboard : préchauffage des statistiques globales : %w", err)
	}
	heroIDs, err := s.repo.HeroIDs(ctx)
	if err != nil {
		return WarmReport{}, err
	}
	var report WarmReport
	var mu sync.Mutex
	var group errgroup.Group
	group.SetLimit(warmConcurrency)
	for _, heroID := range heroIDs {
		if ctx.Err() != nil {
			break
		}
		group.Go(func() error {
			// L'annulation a pu arriver pendant l'attente d'une place : le héros suivant ne démarre pas, et ce n'est
			// pas un échec.
			if ctx.Err() != nil {
				return nil
			}
			_, err := s.personal.get(ctx, heroID.String(), func(ctx context.Context) (Personal, error) {
				return s.loadPersonal(ctx, heroID)
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				report.Failed++
				s.logger.Warn("tableau de bord : préchauffage d'un héros échoué", "hero_id", heroID, "error", err)
				return nil
			}
			report.Heroes++
			return nil
		})
	}
	_ = group.Wait() // les erreurs sont comptées dans le rapport : aucune goroutine ne retourne d'erreur
	report.Elapsed = time.Since(started)
	return report, ctx.Err()
}

// loadPersonal calcule les statistiques d'un héros : ses compteurs et ses principaux ennemis.
func (s *Service) loadPersonal(ctx context.Context, heroID uuid.UUID) (Personal, error) {
	var (
		personal Personal
		enemies  []Enemy
	)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		personal, err = s.repo.Personal(ctx, heroID)
		return err
	})
	group.Go(func() (err error) {
		enemies, err = s.repo.MainEnemies(ctx, heroID, mainEnemiesLimit)
		return err
	})
	if err := group.Wait(); err != nil {
		return Personal{}, err
	}
	personal.MainEnemies = enemies
	return personal, nil
}
