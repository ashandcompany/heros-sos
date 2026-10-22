package realtime_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/realtime"
)

func streamEvent(seq int64) realtime.StreamEvent {
	return realtime.StreamEvent{Recorded: events.Recorded{Seq: seq, Type: events.IncidentDeclared}}
}

func receive(t *testing.T, subscription *realtime.Subscription) (realtime.StreamEvent, bool) {
	t.Helper()
	select {
	case event, open := <-subscription.C:
		return event, open
	case <-time.After(2 * time.Second):
		t.Fatal("aucun événement reçu")
		return realtime.StreamEvent{}, false
	}
}

func TestHubDeliversEveryEventToEverySubscriberInOrder(t *testing.T) {
	hub := realtime.NewHub(8, 5)
	first, err := hub.Subscribe(ids.New())
	if err != nil {
		t.Fatal(err)
	}
	second, _ := hub.Subscribe(ids.New())

	hub.Publish(streamEvent(1))
	hub.Publish(streamEvent(2))

	for _, subscription := range []*realtime.Subscription{first, second} {
		for want := int64(1); want <= 2; want++ {
			if event, open := receive(t, subscription); !open || event.Seq != want {
				t.Errorf("événement %d : reçu %d (flux ouvert : %v)", want, event.Seq, open)
			}
		}
	}
}

func TestHubLimitsTheStreamsOfOneUser(t *testing.T) {
	hub := realtime.NewHub(8, 2)
	user := uuid.New()
	first, _ := hub.Subscribe(user)
	if _, err := hub.Subscribe(user); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Subscribe(user); !errors.Is(err, realtime.ErrTooManyStreams) {
		t.Fatalf("troisième flux : erreur = %v, attendu ErrTooManyStreams", err)
	}
	if _, err := hub.Subscribe(uuid.New()); err != nil {
		t.Errorf("la limite est par utilisateur : %v", err)
	}

	first.Close()
	if _, err := hub.Subscribe(user); err != nil {
		t.Errorf("une place libérée doit se réutiliser : %v", err)
	}
}

func TestASlowSubscriberIsDroppedWithoutHoldingBackTheOthers(t *testing.T) {
	hub := realtime.NewHub(2, 5)
	slow, _ := hub.Subscribe(ids.New())
	fast, _ := hub.Subscribe(ids.New())

	for seq := int64(1); seq <= 4; seq++ {
		hub.Publish(streamEvent(seq))
		if event, open := receive(t, fast); !open || event.Seq != seq {
			t.Fatalf("le lecteur rapide doit tout recevoir : %d (ouvert : %v)", event.Seq, open)
		}
	}

	// Le lecteur lent n'a rien lu : sa file (2) s'est remplie, il a été fermé au troisième événement.
	received := 0
	for {
		_, open := <-slow.C
		if !open {
			break
		}
		received++
	}
	if received != 2 {
		t.Errorf("le lecteur lent a gardé %d événements, attendu 2 (la taille de sa file)", received)
	}
	if hub.Streams() != 1 {
		t.Errorf("flux ouverts = %d, attendu 1", hub.Streams())
	}
}

func TestHubCloseEndsEveryStreamAndRefusesNewOnes(t *testing.T) {
	hub := realtime.NewHub(8, 5)
	subscription, _ := hub.Subscribe(ids.New())

	hub.Close()
	hub.Close() // sans effet

	if _, open := receive(t, subscription); open {
		t.Error("le flux doit être fermé à l'arrêt du hub")
	}
	if _, err := hub.Subscribe(ids.New()); err == nil {
		t.Error("un hub arrêté ne doit plus accepter de flux")
	}
	subscription.Close() // fermer un flux déjà fermé est sans effet
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
