// Package clock fournit une horloge injectable. Les services reçoivent une Clock plutôt que
// d'appeler time.Now, ce qui rend les expirations testables sans attendre.
package clock

import (
	"sync"
	"time"
)

// Clock donne l'heure courante, toujours en UTC.
type Clock interface {
	Now() time.Time
}

// System est l'horloge réelle.
type System struct{}

// Now retourne l'heure système en UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// Fake est une horloge contrôlée, sûre pour un usage concurrent. Elle sert aux tests.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// NewFake crée une horloge figée à l'instant donné.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start.UTC()}
}

// Now retourne l'instant courant de l'horloge simulée.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Advance avance l'horloge de la durée donnée.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}
