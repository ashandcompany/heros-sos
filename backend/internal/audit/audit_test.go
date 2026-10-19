package audit_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

var startAt = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)

func TestRecordStoresTheEntry(t *testing.T) {
	db := testutil.NewDB(t)
	store := audit.NewStore(db, clock.NewFake(startAt))
	entityID := ids.New()

	err := store.Record(context.Background(), audit.Entry{
		Action: audit.ActionUserDeactivated, EntityType: "user", EntityID: &entityID,
		Details: map[string]any{"reason": "spam", "count": 3},
		IP:      netip.MustParseAddr("203.0.113.7"),
	})
	if err != nil {
		t.Fatal(err)
	}

	var row struct {
		Action, EntityType, Details, IPAddress string
		ActorID                                *uuid.UUID
		CreatedAt                              time.Time
	}
	if err := db.Raw(`SELECT action, entity_type, details::text AS details, host(ip_address) AS ip_address, actor_id, created_at FROM audit_logs`).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Action != "user.deactivate" || row.EntityType != "user" || row.IPAddress != "203.0.113.7" || row.ActorID != nil {
		t.Errorf("ligne = %+v", row)
	}
	if row.Details != `{"count": 3, "reason": "spam"}` {
		t.Errorf("détails = %s", row.Details)
	}
	if !row.CreatedAt.Equal(startAt) {
		t.Errorf("date = %v : elle doit venir de l'horloge injectée", row.CreatedAt)
	}
}

func TestRecordWithoutDetailsOrIPStoresEmptyObjectAndNull(t *testing.T) {
	db := testutil.NewDB(t)
	store := audit.NewStore(db, clock.NewFake(startAt))

	if err := store.Record(context.Background(), audit.Entry{Action: "test.minimal"}); err != nil {
		t.Fatal(err)
	}
	var row struct {
		Details string
		NoIP    bool
	}
	if err := db.Raw(`SELECT details::text AS details, ip_address IS NULL AS no_ip FROM audit_logs`).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Details != "{}" || !row.NoIP {
		t.Errorf("ligne = %+v", row)
	}
}

func TestRecordParticipatesInTheCallerTransaction(t *testing.T) {
	db := testutil.NewDB(t)
	store := audit.NewStore(db, clock.NewFake(startAt))
	transactor := database.NewTransactor(db)
	boom := errors.New("l'action échoue")

	err := transactor.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := store.Record(ctx, audit.Entry{Action: "test.rolled_back"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("erreur = %v", err)
	}

	var total int64
	db.Table("audit_logs").Count(&total)
	if total != 0 {
		t.Fatalf("lignes = %d : une action annulée ne doit laisser aucune trace d'audit", total)
	}
}
