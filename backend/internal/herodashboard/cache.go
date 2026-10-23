package herodashboard

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
)

const (
	// refreshTimeout borne un calcul : il tourne détaché de la requête qui l'a déclenché.
	refreshTimeout = 2 * time.Minute
	// maxCachedKeys borne la mémoire : au-delà, le cache est vidé (les héros sont peu nombreux, ce plafond n'est
	// qu'une protection).
	maxCachedKeys = 10_000
)

// snapshotCache garde le dernier résultat calculé pour chaque clé. Les statistiques du tableau de bord parcourent
// de très grosses tables (de quelques centaines de millisecondes à plusieurs secondes sur des millions
// d'incidents) : les recalculer à chaque affichage, pour chaque héros, n'est pas tenable, et elles n'ont pas besoin
// d'être exactes à la seconde.
//
// Une valeur plus jeune que ttl est servie telle quelle. Une valeur plus vieille est servie quand même, sur-le-champ,
// pendant qu'une nouvelle se calcule en arrière-plan (au plus un calcul par clé à la fois) : l'affichage n'attend
// jamais un recalcul. Seul le premier appel d'une clé calcule en direct, et les appels simultanés partagent ce
// calcul. Un ttl nul désactive le cache : chaque appel recalcule (les tests, qui veulent voir leurs écritures).
type snapshotCache[V any] struct {
	ttl    time.Duration
	clock  clock.Clock
	logger *slog.Logger

	mu      sync.Mutex
	entries map[string]*snapshot[V]
	group   singleflight.Group
}

type snapshot[V any] struct {
	value      V
	at         time.Time
	refreshing bool
}

func newSnapshotCache[V any](ttl time.Duration, clk clock.Clock, logger *slog.Logger) *snapshotCache[V] {
	return &snapshotCache[V]{ttl: ttl, clock: clk, logger: logger, entries: map[string]*snapshot[V]{}}
}

// enabled dit si le cache garde des valeurs (un ttl nul le désactive).
func (c *snapshotCache[V]) enabled() bool { return c.ttl > 0 }

// get retourne la valeur de la clé, calculée par load au besoin.
func (c *snapshotCache[V]) get(ctx context.Context, key string, load func(context.Context) (V, error)) (V, error) {
	if c.ttl <= 0 {
		return load(ctx)
	}
	c.mu.Lock()
	entry, cached := c.entries[key]
	if cached {
		value := entry.value
		if c.clock.Now().Sub(entry.at) >= c.ttl && !entry.refreshing {
			entry.refreshing = true
			go c.refresh(key, load)
		}
		c.mu.Unlock()
		return value, nil
	}
	c.mu.Unlock()

	// Un contexte déjà annulé ne démarre pas de calcul : il serait détaché de la requête et se poursuivrait pour
	// personne (c'est ce qui arrivait au préchauffage quand le serveur s'arrête).
	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}

	// Premier appel de cette clé : calcul en direct, partagé entre les appels simultanés. Il est détaché de la requête
	// (qui peut être abandonnée) mais l'appelant, lui, n'attend pas plus longtemps que sa requête.
	result := c.group.DoChan(key, func() (any, error) {
		// Un calcul partagé qui vient de se terminer a pu ranger sa valeur entre le moment où cet appel a constaté
		// l'absence de valeur et celui où il est devenu le calcul de la clé : la revérifier évite de calculer deux fois.
		c.mu.Lock()
		if entry, ok := c.entries[key]; ok {
			value := entry.value
			c.mu.Unlock()
			return value, nil
		}
		c.mu.Unlock()
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshTimeout)
		defer cancel()
		value, err := load(loadCtx)
		if err != nil {
			return nil, err
		}
		c.store(key, value)
		return value, nil
	})
	select {
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	case res := <-result:
		if res.Err != nil {
			var zero V
			return zero, res.Err
		}
		return res.Val.(V), nil
	}
}

func (c *snapshotCache[V]) store(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= maxCachedKeys {
		clear(c.entries)
	}
	c.entries[key] = &snapshot[V]{value: value, at: c.clock.Now()}
}

// refresh recalcule une valeur périmée. En cas d'échec, l'ancienne reste servie et le prochain appel réessaie.
func (c *snapshotCache[V]) refresh(key string, load func(context.Context) (V, error)) {
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()
	value, err := load(ctx)
	if err != nil {
		c.logger.Warn("tableau de bord : statistiques non rafraîchies, l'ancienne valeur est conservée", "key", key, "error", err)
		c.mu.Lock()
		if entry, ok := c.entries[key]; ok {
			entry.refreshing = false
		}
		c.mu.Unlock()
		return
	}
	c.store(key, value)
}
