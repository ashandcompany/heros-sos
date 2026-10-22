package incident

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/perfseed"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

// Les listes et les historiques ne lisent que leur page grâce à des index posés après mesure sur 5 millions
// d'incidents (migration add_incident_perf_indexes). Ce test explique les vraies requêtes et échoue si l'une
// retombe sur un parcours complet de table : un index supprimé, ou une réécriture qui le contourne, y est vu.
//
// Le planificateur choisit un parcours complet sur une petite table même quand un index existe : la base de test
// est donc interrogée avec enable_seqscan = off, ce qui ne laisse un parcours complet que faute d'index utilisable.
func TestReadQueriesUseTheirIndexes(t *testing.T) {
	db := testutil.NewDB(t)
	// Assez de citoyens pour qu'un déclarant ne représente qu'une petite part des incidents, comme en vrai : avec
	// trois, un tiers des incidents seraient les siens et le tri par date suffirait au planificateur.
	testutil.SeedReference(t, db, 200, 4, 3)
	ctx := context.Background()
	if err := perfseed.Load(ctx, db, perfseed.Options{Incidents: 3000}); err != nil {
		t.Fatal(err)
	}
	if err := perfseed.Settle(ctx, db); err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.ExecContext(ctx, `SET enable_seqscan = off`); err != nil {
		t.Fatal(err)
	}

	var reporterID, heroID, villainID uuid.UUID
	for query, target := range map[string]*uuid.UUID{
		`SELECT id FROM users WHERE role = 'citizen' LIMIT 1`: &reporterID,
		`SELECT user_id FROM hero_profiles LIMIT 1`:           &heroID,
		`SELECT id FROM villains LIMIT 1`:                     &villainID,
	} {
		if err := db.Raw(query).Row().Scan(target); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewRepository(db).(*gormRepository)
	listSQL := func(filter ListFilter) (string, []any) {
		statement := repo.filteredQuery(ctx, filter).
			Order("incidents.created_at DESC, incidents.id DESC").Limit(DefaultListLimitForPlans).
			Session(&gorm.Session{DryRun: true}).Find(&[]joinedRecord{}).Statement
		return statement.SQL.String(), statement.Vars
	}
	globalSQL, globalArgs := listSQL(ListFilter{})
	statusSQL, statusArgs := listSQL(ListFilter{Statuses: []Status{StatusReported, StatusInProgress}})
	reporterSQL, reporterArgs := listSQL(ListFilter{ReporterID: &reporterID})

	for _, tc := range []struct {
		name  string
		query string
		args  []any
		// index est celui que le plan doit lire (vide : n'importe lequel, tant qu'il n'y a pas de parcours complet).
		index string
		// full liste les tables qui ne doivent jamais être parcourues en entier.
		full []string
		// ordered : l'index donne déjà l'ordre demandé, aucun tri ne doit apparaître. Sans index d'ordre, PostgreSQL
		// lit un index entier puis trie : ce n'est pas un parcours de table, mais c'est la même lenteur. Réservé aux
		// requêtes dont le plan ne dépend pas de la taille de la base : pour un filtre, quelques dizaines de lignes
		// se trient aussi vite qu'elles se lisent dans l'ordre, et le planificateur le sait.
		ordered bool
	}{
		{"liste sans filtre", globalSQL, globalArgs, "incidents_created_idx", []string{"incidents"}, true},
		{"liste par statut", statusSQL, statusArgs, "", []string{"incidents"}, false},
		{"mes incidents", reporterSQL, reporterArgs, "incidents_reporter_created_idx", []string{"incidents"}, false},
		{"historique d'un héros", positional(resolvedByHeroSQL), []any{heroID, DefaultListLimitForPlans, 0},
			"intervention_heroes_resolved_idx", []string{"intervention_heroes", "interventions"}, false},
		{"historique d'un vilain", positional(byVillainSQL), []any{villainID, DefaultListLimitForPlans, 0},
			"incident_villains_villain_idx", []string{"incident_villains", "incidents"}, false},
		{"dernière intervention d'un incident",
			`SELECT id, ended_at FROM interventions WHERE incident_id = $1 ORDER BY started_at DESC NULLS LAST, id DESC LIMIT 1`,
			[]any{uuid.New()}, "interventions_incident_idx", []string{"interventions"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := explain(ctx, t, conn, tc.query, tc.args...)
			if tc.index != "" && !strings.Contains(plan, tc.index) {
				t.Errorf("le plan doit lire %s :\n%s", tc.index, plan)
			}
			if tc.ordered && strings.Contains(plan, "Sort") {
				t.Errorf("l'ordre demandé doit venir de l'index, sans tri :\n%s", plan)
			}
			for _, table := range tc.full {
				if strings.Contains(plan, "Seq Scan on "+table+" ") || strings.HasSuffix(strings.TrimSpace(plan), "Seq Scan on "+table) {
					t.Errorf("parcours complet de %s :\n%s", table, plan)
				}
			}
		})
	}
}

// DefaultListLimitForPlans est la taille de page des plans expliqués : celle d'une liste ordinaire.
const DefaultListLimitForPlans = 20

// positional remplace les ? de GORM par $1, $2… : EXPLAIN passe par database/sql, pas par GORM.
func positional(query string) string {
	var out strings.Builder
	n := 0
	for _, char := range query {
		if char == '?' {
			n++
			out.WriteString("$" + strconv.Itoa(n))
			continue
		}
		out.WriteRune(char)
	}
	return out.String()
}

func explain(ctx context.Context, t *testing.T, conn *sql.Conn, query string, args ...any) string {
	t.Helper()
	rows, err := conn.QueryContext(ctx, `EXPLAIN (COSTS OFF) `+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN : %v\n%s", err, query)
	}
	defer func() { _ = rows.Close() }()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}
