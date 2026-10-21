package newsletter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/newsletter"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

var (
	ctx     = context.Background()
	startAt = time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
)

type fixture struct {
	db    *gorm.DB
	repo  newsletter.Repository
	users user.Repository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{db: db, repo: newsletter.NewRepository(db), users: user.NewRepository(db, sealer)}
}

func (f fixture) newUser(t *testing.T, email string) uuid.UUID {
	t.Helper()
	u := &user.User{
		ID: ids.New(), Email: email, PasswordHash: "$argon2id$x", Role: user.RoleCitizen, Status: user.StatusActive,
		Civility: "madame", FirstName: "Natasha", LastName: "Romanoff", BirthDate: time.Date(1984, 11, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := f.users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	return u.ID
}

type row struct {
	ID                          uuid.UUID
	Status, Source              string
	UserID                      *uuid.UUID
	ConfirmedAt, UnsubscribedAt *time.Time
	CreatedAt                   time.Time
}

func (f fixture) row(t *testing.T, email string) row {
	t.Helper()
	var found row
	result := f.db.Raw(`SELECT id, status, source, user_id, confirmed_at, unsubscribed_at, created_at FROM newsletter_subscriptions WHERE email = ?`, email).Scan(&found)
	if result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("ligne de %s : %d trouvée(s), erreur %v", email, result.RowsAffected, result.Error)
	}
	return found
}

func TestRequestCreatesRelaunchesAndNeverOverwritesAConfirmedSubscription(t *testing.T) {
	f := newFixture(t)
	id, needed, err := f.repo.Request(ctx, ids.New(), "eve@example.org", startAt)
	if err != nil || !needed {
		t.Fatalf("première demande : %v %v", needed, err)
	}
	if got := f.row(t, "eve@example.org"); got.ID != id || got.Status != "pending" || got.Source != "footer" || got.ConfirmedAt != nil {
		t.Fatalf("après la première demande : %+v", got)
	}

	// Une relance garde l'identifiant (le premier lien reste donc valable) et repousse la date de dépôt.
	later := startAt.Add(time.Hour)
	again, needed, err := f.repo.Request(ctx, ids.New(), "eve@example.org", later)
	if err != nil || !needed || again != id {
		t.Fatalf("relance : %v %v %v", again, needed, err)
	}
	if got := f.row(t, "eve@example.org"); !got.CreatedAt.Equal(later) {
		t.Errorf("la relance doit repousser la date de dépôt : %v", got.CreatedAt)
	}

	if err := f.repo.Confirm(ctx, id, later); err != nil {
		t.Fatal(err)
	}
	// Abonné : une nouvelle demande n'écrit rien et n'appelle aucun mail.
	if _, needed, err := f.repo.Request(ctx, ids.New(), "eve@example.org", later.Add(time.Hour)); err != nil || needed {
		t.Fatalf("demande d'un abonné : %v %v", needed, err)
	}
	if got := f.row(t, "eve@example.org"); got.Status != "subscribed" || got.ConfirmedAt == nil || !got.ConfirmedAt.Equal(later) {
		t.Errorf("l'abonnement confirmé doit rester intact : %+v", got)
	}
}

func TestConfirmIsIdempotentAndRefusesWithdrawnOrUnknownRequests(t *testing.T) {
	f := newFixture(t)
	id, _, _ := f.repo.Request(ctx, ids.New(), "eve@example.org", startAt)

	for range 2 {
		if err := f.repo.Confirm(ctx, id, startAt); err != nil {
			t.Fatalf("Confirm : %v", err)
		}
	}
	if err := f.repo.Confirm(ctx, uuid.New(), startAt); !errors.Is(err, newsletter.ErrLinkInvalid) {
		t.Errorf("demande inconnue : %v", err)
	}

	// Retiré par son titulaire : le lien ne le réabonne pas.
	owner := f.newUser(t, "natasha@example.org")
	pending, _, _ := f.repo.Request(ctx, ids.New(), "natasha@example.org", startAt)
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", false, startAt); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.Confirm(ctx, pending, startAt); !errors.Is(err, newsletter.ErrLinkInvalid) {
		t.Errorf("demande retirée : %v", err)
	}
	if got := f.row(t, "natasha@example.org"); got.Status != "unsubscribed" {
		t.Errorf("état = %+v", got)
	}
}

// Deux navigateurs qui réclament l'abonnement pendant que le lien est cliqué ne doivent jamais produire
// d'état incohérent (la contrainte « pas d'abonné sans consentement » est en base).
func TestConcurrentRequestsAndConfirmationsEndSubscribed(t *testing.T) {
	f := newFixture(t)
	id, _, err := f.repo.Request(ctx, ids.New(), "eve@example.org", startAt)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 8
	var wg sync.WaitGroup
	for range workers {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, _, err := f.repo.Request(ctx, ids.New(), "eve@example.org", startAt.Add(time.Minute)); err != nil {
				t.Errorf("Request : %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := f.repo.Confirm(ctx, id, startAt.Add(time.Minute)); err != nil {
				t.Errorf("Confirm : %v", err)
			}
		}()
	}
	wg.Wait()

	got := f.row(t, "eve@example.org")
	if got.Status != "subscribed" || got.ConfirmedAt == nil {
		t.Errorf("état final = %+v", got)
	}
	var rows int64
	f.db.Table("newsletter_subscriptions").Where("email = ?", "eve@example.org").Count(&rows)
	if rows != 1 {
		t.Errorf("une seule ligne par adresse : %d", rows)
	}
}

func TestAccountPreferenceSubscribesDirectlyAndKeepsTheOriginalConsentDate(t *testing.T) {
	f := newFixture(t)
	owner := f.newUser(t, "natasha@example.org")

	subscribed, err := f.repo.IsSubscribed(ctx, owner, "natasha@example.org")
	if err != nil || subscribed {
		t.Fatalf("avant : %v %v", subscribed, err)
	}
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", true, startAt); err != nil {
		t.Fatal(err)
	}
	got := f.row(t, "natasha@example.org")
	if got.Status != "subscribed" || got.Source != "account" || got.UserID == nil || *got.UserID != owner || got.ConfirmedAt == nil || !got.ConfirmedAt.Equal(startAt) {
		t.Fatalf("après l'abonnement : %+v", got)
	}

	// Réabonner un abonné ne change pas la date de son consentement.
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", true, startAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if again := f.row(t, "natasha@example.org"); !again.ConfirmedAt.Equal(startAt) || again.ID != got.ID {
		t.Errorf("après le second abonnement : %+v", again)
	}

	// Se retirer puis revenir date un nouveau consentement.
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", false, startAt.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if gone := f.row(t, "natasha@example.org"); gone.Status != "unsubscribed" || gone.UnsubscribedAt == nil {
		t.Errorf("après le retrait : %+v", gone)
	}
	if subscribed, _ := f.repo.IsSubscribed(ctx, owner, "natasha@example.org"); subscribed {
		t.Error("retiré : ne doit plus être abonné")
	}
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", true, startAt.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	back := f.row(t, "natasha@example.org")
	if back.Status != "subscribed" || back.UnsubscribedAt != nil || !back.ConfirmedAt.Equal(startAt.Add(3*time.Hour)) {
		t.Errorf("après le retour : %+v", back)
	}

	// Se retirer d'une liste où l'on n'est pas ne fait rien (et n'échoue pas).
	other := f.newUser(t, "autre@example.org")
	if err := f.repo.SetAccountPreference(ctx, ids.New(), other, "autre@example.org", false, startAt); err != nil {
		t.Errorf("désabonnement sans abonnement : %v", err)
	}
	var rows int64
	f.db.Table("newsletter_subscriptions").Where("email = ?", "autre@example.org").Count(&rows)
	if rows != 0 {
		t.Error("se désabonner sans être abonné ne doit rien créer")
	}
}

func TestIsSubscribedMatchesByAccountOrByAddress(t *testing.T) {
	f := newFixture(t)
	owner := f.newUser(t, "natasha@example.org")

	// Abonnée depuis le pied de page, avant ou sans lien avec son compte : l'adresse suffit.
	id, _, _ := f.repo.Request(ctx, ids.New(), "natasha@example.org", startAt)
	if subscribed, _ := f.repo.IsSubscribed(ctx, owner, "natasha@example.org"); subscribed {
		t.Error("en attente de confirmation : pas abonnée")
	}
	if err := f.repo.Confirm(ctx, id, startAt); err != nil {
		t.Fatal(err)
	}
	if subscribed, _ := f.repo.IsSubscribed(ctx, owner, "NATASHA@example.org"); !subscribed {
		t.Error("l'adresse confirmée (casse ignorée) doit abonner le compte")
	}
	if subscribed, _ := f.repo.IsSubscribed(ctx, uuid.New(), "inconnue@example.org"); subscribed {
		t.Error("un autre compte et une autre adresse ne sont pas abonnés")
	}
}

func TestFollowEmailMovesTheSubscriptionToTheNewAddress(t *testing.T) {
	f := newFixture(t)
	owner := f.newUser(t, "ancienne@example.org")
	if err := f.repo.SetAccountPreference(ctx, ids.New(), owner, "ancienne@example.org", true, startAt); err != nil {
		t.Fatal(err)
	}

	if err := f.repo.FollowEmail(ctx, owner, "nouvelle@example.org"); err != nil {
		t.Fatal(err)
	}
	if got := f.row(t, "nouvelle@example.org"); got.Status != "subscribed" || got.UserID == nil || *got.UserID != owner {
		t.Errorf("après le suivi : %+v", got)
	}
	var old int64
	f.db.Table("newsletter_subscriptions").Where("email = ?", "ancienne@example.org").Count(&old)
	if old != 0 {
		t.Error("l'ancienne adresse ne doit plus être dans la liste")
	}

	// Sans abonnement, rien à suivre.
	if err := f.repo.FollowEmail(ctx, f.newUser(t, "sans@example.org"), "autre@example.org"); err != nil {
		t.Errorf("suivi sans abonnement : %v", err)
	}
}

// Si la nouvelle adresse est déjà dans la liste, l'abonnement existant n'est pas écrasé et rien n'échoue :
// le changement d'adresse d'un compte ne doit jamais être bloqué par la newsletter.
func TestFollowEmailNeverFailsOnAnAddressAlreadyInTheList(t *testing.T) {
	f := newFixture(t)
	owner := f.newUser(t, "ancienne@example.org")
	_ = f.repo.SetAccountPreference(ctx, ids.New(), owner, "ancienne@example.org", true, startAt)
	taken, _, _ := f.repo.Request(ctx, ids.New(), "prise@example.org", startAt)

	if err := f.repo.FollowEmail(ctx, owner, "PRISE@example.org"); err != nil {
		t.Fatalf("FollowEmail : %v", err)
	}
	if got := f.row(t, "prise@example.org"); got.ID != taken || got.Status != "pending" {
		t.Errorf("l'abonnement existant ne doit pas changer : %+v", got)
	}
	if got := f.row(t, "ancienne@example.org"); got.Status != "subscribed" {
		t.Errorf("l'abonnement du compte reste tel quel : %+v", got)
	}
}

func TestDeleteStalePendingKeepsEverythingElse(t *testing.T) {
	f := newFixture(t)
	owner := f.newUser(t, "natasha@example.org")

	_, _, _ = f.repo.Request(ctx, ids.New(), "vieille@example.org", startAt.Add(-10*24*time.Hour))
	_, _, _ = f.repo.Request(ctx, ids.New(), "recente@example.org", startAt.Add(-time.Hour))
	oldConfirmed, _, _ := f.repo.Request(ctx, ids.New(), "confirmee@example.org", startAt.Add(-30*24*time.Hour))
	_ = f.repo.Confirm(ctx, oldConfirmed, startAt.Add(-29*24*time.Hour))
	_ = f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", true, startAt.Add(-30*24*time.Hour))
	_ = f.repo.SetAccountPreference(ctx, ids.New(), owner, "natasha@example.org", false, startAt.Add(-20*24*time.Hour))

	deleted, err := f.repo.DeleteStalePending(ctx, startAt.Add(-7*24*time.Hour))
	if err != nil || deleted != 1 {
		t.Fatalf("supprimées = %d, %v", deleted, err)
	}
	remaining := map[string]string{}
	var all []struct{ Email, Status string }
	f.db.Raw(`SELECT email, status FROM newsletter_subscriptions`).Scan(&all)
	for _, r := range all {
		remaining[r.Email] = r.Status
	}
	want := map[string]string{"recente@example.org": "pending", "confirmee@example.org": "subscribed", "natasha@example.org": "unsubscribed"}
	if fmt.Sprint(remaining) != fmt.Sprint(want) {
		t.Errorf("lignes restantes = %v, attendu %v", remaining, want)
	}
}

func TestDatabaseRefusesASubscriberWithoutConsentProof(t *testing.T) {
	f := newFixture(t)
	err := f.db.Exec(`INSERT INTO newsletter_subscriptions (id, email, status, source) VALUES (?, 'x@example.org', 'subscribed', 'footer')`, uuid.New()).Error
	if err == nil {
		t.Error("un abonné sans date de confirmation doit être refusé par la base")
	}
	err = f.db.Exec(`INSERT INTO newsletter_subscriptions (id, email, status, source) VALUES (?, 'y@example.org', 'pending', 'robot')`, uuid.New()).Error
	if err == nil {
		t.Error("une source inconnue doit être refusée par la base")
	}
}
