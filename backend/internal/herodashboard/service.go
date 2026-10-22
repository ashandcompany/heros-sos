package herodashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
)

const (
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
}

// NewService crée le service.
func NewService(repo Repository, incidents IncidentLister, opts Options) *Service {
	return &Service{
		repo: repo, incidents: incidents,
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
		dashboard.Global, err = s.global.get(ctx, "global", s.repo.Global)
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
