// Package realtime diffuse les événements du journal aux navigateurs connectés, en temps réel, par un flux SSE
// (Server-Sent Events : une réponse HTTP qui reste ouverte, le serveur y écrit à chaque événement).
//
// Le journal (paquet events) fait foi : le flux n'en est qu'un reflet. Un lecteur qui perd la connexion, ou qui
// est trop lent, est déconnecté ; le navigateur se reconnecte seul en redonnant le numéro du dernier événement
// reçu (en-tête Last-Event-ID), et le serveur rejoue ce qu'il a manqué depuis le journal. Rien n'est perdu, et
// rien n'est gardé en mémoire au-delà de la file d'attente de chaque lecteur.
package realtime

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/events"
)

// ErrTooManyStreams indique qu'un utilisateur a atteint son nombre maximal de flux ouverts.
var ErrTooManyStreams = errors.New("realtime : trop de flux ouverts")

// StreamEvent est un événement du journal prêt à être diffusé : le déclarant de l'incident y est ajouté, pour
// que chaque lecteur sache sans requête si l'événement le concerne.
type StreamEvent struct {
	events.Recorded
	ReporterID *uuid.UUID
}

// Hub répartit les événements entre les flux ouverts.
type Hub struct {
	mu         sync.Mutex
	closed     bool
	buffer     int
	maxPerUser int
	streams    map[*Subscription]struct{}
	perUser    map[uuid.UUID]int
}

// NewHub crée le hub. buffer est le nombre d'événements gardés en attente par lecteur ; maxPerUser, le nombre de
// flux simultanés autorisés par utilisateur.
func NewHub(buffer, maxPerUser int) *Hub {
	return &Hub{
		buffer: buffer, maxPerUser: maxPerUser,
		streams: map[*Subscription]struct{}{}, perUser: map[uuid.UUID]int{},
	}
}

// Subscription est l'abonnement d'un flux. Son canal est fermé quand le lecteur est trop lent, ou que le hub
// s'arrête : le flux se termine alors, et le navigateur se reconnecte.
type Subscription struct {
	C      <-chan StreamEvent
	events chan StreamEvent
	user   uuid.UUID
	hub    *Hub
}

// Subscribe ouvre un abonnement pour l'utilisateur (ErrTooManyStreams s'il en a déjà trop).
func (h *Hub) Subscribe(userID uuid.UUID) (*Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, errors.New("realtime : le serveur s'arrête")
	}
	if h.perUser[userID] >= h.maxPerUser {
		return nil, ErrTooManyStreams
	}
	channel := make(chan StreamEvent, h.buffer)
	subscription := &Subscription{C: channel, events: channel, user: userID, hub: h}
	h.streams[subscription] = struct{}{}
	h.perUser[userID]++
	return subscription, nil
}

// Publish remet l'événement à chaque flux. Un flux dont la file est pleine est fermé plutôt que d'attendre : un
// lecteur lent ne doit jamais ralentir les autres, ni faire grossir la mémoire.
func (h *Hub) Publish(event StreamEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscription := range h.streams {
		select {
		case subscription.events <- event:
		default:
			h.remove(subscription)
		}
	}
}

// Close ferme tous les flux : à l'arrêt du serveur, sans quoi leurs requêtes retarderaient l'arrêt HTTP.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for subscription := range h.streams {
		h.remove(subscription)
	}
}

// Streams retourne le nombre de flux ouverts.
func (h *Hub) Streams() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.streams)
}

// Close met fin à l'abonnement (sans effet s'il est déjà fermé).
func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.hub.remove(s)
}

// remove ferme et retire l'abonnement ; h.mu doit être tenu.
func (h *Hub) remove(subscription *Subscription) {
	if _, open := h.streams[subscription]; !open {
		return
	}
	delete(h.streams, subscription)
	close(subscription.events)
	if h.perUser[subscription.user]--; h.perUser[subscription.user] <= 0 {
		delete(h.perUser, subscription.user)
	}
}
