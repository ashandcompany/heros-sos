package perfseed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/perfseed"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

var now = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

// busyHero donne au héros une intervention en cours : perfseed ne doit pas lui en confier une autre.
func busyHero(t *testing.T, db *gorm.DB, heroID uuid.UUID) {
	t.Helper()
	incidentID, interventionID := ids.New(), ids.New()
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO incidents (id, type_id, title, description, status) VALUES (?, 1, 'En cours', 'déjà là', 'in_progress')`, []any{incidentID}},
		{`INSERT INTO interventions (id, incident_id, status, started_at) VALUES (?, ?, 'in_progress', now())`, []any{interventionID, incidentID}},
		{`INSERT INTO intervention_heroes (intervention_id, hero_id, role, participation, is_active) VALUES (?, ?, 'lead', 'confirmed', true)`, []any{interventionID, heroID}},
	} {
		if err := db.Exec(statement.sql, statement.args...).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func count(t *testing.T, db *gorm.DB, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLoadCreatesConsistentIncidents(t *testing.T) {
	db := testutil.NewDB(t)
	heroes := testutil.SeedReference(t, db, 3, 4, 2)
	busyHero(t, db, heroes[0])
	const total = 2000
	span := 10 * 24 * time.Hour

	// Trois lots, dont un plus petit : la numérotation continue d'un lot à l'autre.
	err := perfseed.Load(context.Background(), db, perfseed.Options{Incidents: total, BatchSize: 700, Span: span, Now: now, Append: true})
	if err != nil {
		t.Fatal(err)
	}

	const seeded = "title LIKE 'Incident de charge%'"
	if got := count(t, db, `SELECT count(*) FROM incidents WHERE `+seeded); got != total {
		t.Fatalf("incidents créés = %d, attendu %d", got, total)
	}
	byStatus := func(status string) int64 {
		return count(t, db, `SELECT count(*) FROM incidents WHERE `+seeded+` AND status = ?`, status)
	}
	// Les derniers incidents (un sur 500) sont ouverts : d'abord un « en cours » par héros libre
	// (le quatrième est déjà occupé), les autres sont « signalés ».
	const inProgress = 3
	if got := byStatus("in_progress"); got != inProgress {
		t.Errorf("incidents en cours = %d, attendu %d", got, inProgress)
	}
	reported := byStatus("reported")
	if reported != total/500-inProgress {
		t.Errorf("incidents signalés = %d, attendu %d", reported, total/500-inProgress)
	}
	closed := byStatus("succeeded") + byStatus("failed")
	if share := float64(byStatus("succeeded")) / float64(closed); share < 0.88 || share > 0.96 {
		t.Errorf("part des incidents réussis = %.2f, attendu environ 0,92", share)
	}

	// Une intervention par incident non signalé, avec un seul héros en tête.
	if got := count(t, db, `SELECT count(*) FROM interventions i JOIN incidents n ON n.id = i.incident_id WHERE n.`+seeded); got != total-reported {
		t.Errorf("interventions = %d, attendu %d (tous les incidents sauf les signalés)", got, total-reported)
	}
	if got := count(t, db, `SELECT count(*) FROM (
		SELECT i.id FROM interventions i JOIN incidents n ON n.id = i.incident_id
		LEFT JOIN intervention_heroes h ON h.intervention_id = i.id AND h.role = 'lead'
		WHERE n.`+seeded+` GROUP BY i.id HAVING count(h.hero_id) <> 1) AS broken`); got != 0 {
		t.Errorf("%d interventions n'ont pas exactement un héros en tête", got)
	}
	if got := count(t, db, `SELECT count(*) FROM interventions WHERE status = 'in_progress' AND ended_at IS NOT NULL`); got != 0 {
		t.Errorf("%d interventions en cours ont une date de fin", got)
	}
	if got := count(t, db, `SELECT count(*) FROM interventions WHERE status <> 'in_progress' AND ended_at IS NULL`); got != 0 {
		t.Errorf("%d interventions closes n'ont pas de date de fin", got)
	}
	members := count(t, db, `SELECT count(*) FROM intervention_heroes WHERE role = 'member'`)
	if share := float64(members) / float64(closed); share < 0.2 || share > 0.4 {
		t.Errorf("part des interventions à deux héros = %.2f, attendu environ 0,3", share)
	}
	villains := count(t, db, `SELECT count(*) FROM incident_villains`)
	if share := float64(villains) / total; share < 0.5 || share > 0.7 {
		t.Errorf("part des incidents avec un vilain = %.2f, attendu environ 0,6", share)
	}

	// Dates : étalées sur la durée demandée, jamais dans le futur, et l'événement précède sa déclaration.
	var oldest, newest time.Time
	if err := db.Raw(`SELECT min(created_at), max(created_at) FROM incidents WHERE `+seeded).Row().Scan(&oldest, &newest); err != nil {
		t.Fatal(err)
	}
	if oldest.Before(now.Add(-span)) || oldest.After(now.Add(-span).Add(time.Minute)) || newest.After(now) {
		t.Errorf("dates de %v à %v, attendu entre %v et %v", oldest, newest, now.Add(-span), now)
	}
	if got := count(t, db, `SELECT count(*) FROM incidents WHERE `+seeded+` AND occurred_at >= created_at`); got != 0 {
		t.Errorf("%d incidents ont eu lieu après leur déclaration", got)
	}

	// Colonnes dénormalisées des historiques : fournies par le générateur, cohérentes avec les tables d'origine.
	if got := count(t, db, `SELECT count(*) FROM incident_villains v JOIN incidents n ON n.id = v.incident_id
		WHERE v.incident_created_at <> n.created_at`); got != 0 {
		t.Errorf("%d liens vilain-incident ont une date de création différente de celle de l'incident", got)
	}
	if got := count(t, db, `SELECT count(*) FROM intervention_heroes h JOIN interventions i ON i.id = h.intervention_id
		WHERE (i.status = 'succeeded') <> (h.resolved_at IS NOT NULL) AND h.participation = 'confirmed'`); got != 0 {
		t.Errorf("%d héros ont une date de résolution incohérente avec l'état de leur intervention", got)
	}
	// Les compteurs de lecture se sont mis à jour d'eux-mêmes, lot après lot.
	testutil.AssertReadCounters(t, db)
}

func TestSettleAndSizesDescribeTheLoadedTables(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.SeedReference(t, db, 2, 3, 2)
	ctx := context.Background()
	if err := perfseed.Load(ctx, db, perfseed.Options{Incidents: 1500, Now: now}); err != nil {
		t.Fatal(err)
	}

	if err := perfseed.Settle(ctx, db); err != nil {
		t.Fatal(err)
	}
	sizes, err := perfseed.Sizes(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 4 || sizes[0].Name != "incidents" || sizes[0].Rows != 1500 || sizes[0].TotalBytes == 0 {
		t.Errorf("tailles = %+v", sizes)
	}
}

func TestLoadRefusesMissingReferenceData(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.SeedReference(t, db, 2, 0, 2) // aucun héros

	err := perfseed.Load(context.Background(), db, perfseed.Options{Incidents: 2000, Now: now})
	if !errors.Is(err, perfseed.ErrMissingReference) {
		t.Fatalf("erreur = %v, attendu ErrMissingReference", err)
	}
	if got := count(t, db, `SELECT count(*) FROM incidents`); got != 0 {
		t.Errorf("%d incidents créés malgré l'erreur", got)
	}
}

func TestLoadRefusesAnAlreadyFilledDatabaseUnlessAppending(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.SeedReference(t, db, 2, 3, 2)
	ctx := context.Background()
	if err := perfseed.Load(ctx, db, perfseed.Options{Incidents: 11_000, Now: now}); err != nil {
		t.Fatal(err)
	}

	err := perfseed.Load(ctx, db, perfseed.Options{Incidents: 2000, Now: now})
	if !errors.Is(err, perfseed.ErrNotFresh) {
		t.Fatalf("erreur = %v, attendu ErrNotFresh", err)
	}
	// Les héros libres ont tous reçu une intervention en cours au premier passage : l'ajout n'en
	// crée plus, sans quoi un héros mènerait deux interventions.
	if err := perfseed.Load(ctx, db, perfseed.Options{Incidents: 2000, Now: now, Append: true}); err != nil {
		t.Fatalf("ajout : %v", err)
	}
	if got := count(t, db, `SELECT count(*) FROM incidents`); got != 13_000 {
		t.Errorf("incidents = %d, attendu 13000", got)
	}
}

func TestLoadStopsWhenTheDatabaseIsTooBig(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.SeedReference(t, db, 2, 3, 2)

	err := perfseed.Load(context.Background(), db, perfseed.Options{Incidents: 2000, Now: now, MaxDatabaseBytes: 1})
	if !errors.Is(err, perfseed.ErrSizeLimit) {
		t.Fatalf("erreur = %v, attendu ErrSizeLimit", err)
	}
}

func TestLoadRejectsATinyVolume(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.SeedReference(t, db, 2, 3, 2)

	if err := perfseed.Load(context.Background(), db, perfseed.Options{Incidents: 10, Now: now}); err == nil {
		t.Fatal("un volume ridicule doit être refusé")
	}
}
