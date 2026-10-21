// Package ratelimit limite le nombre de requêtes par clé (adresse IP, identifiant de connexion…)
// pour freiner le bourrage d'identifiants et l'envoi massif de mails.
//
// Le limiteur garde son état en mémoire : il est exact pour une seule instance de l'API. Avec
// plusieurs instances, la limite effective est multipliée par leur nombre.
package ratelimit

import (
	"sync"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
)

type window struct {
	start time.Time
	count int
}

// Limiter applique une règle « au plus Max requêtes par fenêtre de durée Window », en
// fenêtres fixes. Il est sûr pour un usage concurrent.
type Limiter struct {
	rule    config.RateRule
	maxKeys int
	clock   clock.Clock

	mu      sync.Mutex
	windows map[string]*window
}

// New crée un limiteur. maxKeys borne le nombre de clés suivies simultanément, pour qu'un
// attaquant qui varie les clés ne puisse pas épuiser la mémoire.
func New(rule config.RateRule, maxKeys int, clk clock.Clock) *Limiter {
	return &Limiter{rule: rule, maxKeys: maxKeys, clock: clk, windows: make(map[string]*window)}
}

// Allow compte une requête pour la clé. Il retourne false, avec le délai avant la prochaine
// fenêtre, quand la limite est atteinte. Si la table des clés est pleine, les clés nouvelles
// sont refusées : face à une attaque, mieux vaut échouer fermé qu'ouvert.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	current, tracked := l.windows[key]
	if tracked && now.Sub(current.start) >= l.rule.Window {
		delete(l.windows, key)
		tracked = false
	}

	if !tracked {
		if len(l.windows) >= l.maxKeys {
			l.sweepLocked(now)
			if len(l.windows) >= l.maxKeys {
				return false, l.rule.Window
			}
		}
		l.windows[key] = &window{start: now, count: 1}
		return true, 0
	}

	if current.count >= l.rule.Max {
		return false, l.rule.Window - now.Sub(current.start)
	}
	current.count++
	return true, 0
}

// Reset oublie la clé, par exemple après une connexion réussie.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}

// Sweep supprime les fenêtres expirées. Une tâche périodique l'appelle pour libérer la mémoire.
func (l *Limiter) Sweep() {
	now := l.clock.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked(now)
}

func (l *Limiter) sweepLocked(now time.Time) {
	for key, current := range l.windows {
		if now.Sub(current.start) >= l.rule.Window {
			delete(l.windows, key)
		}
	}
}

// Len retourne le nombre de clés suivies (utile aux tests et aux métriques).
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.windows)
}
