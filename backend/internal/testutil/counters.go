package testutil

import (
	"testing"

	"gorm.io/gorm"
)

// counterChecks compare chaque compteur de lecture à un recalcul depuis les tables. Les lignes de
// compteur à zéro sont ignorées : elles restent après qu'un incident (ou une participation) a disparu.
var counterChecks = []struct{ name, expected, stored string }{
	{
		name: "incident_stats",
		expected: `SELECT moderation, coalesce(status, 'reported') AS status, type_id, count(*) AS incidents, coalesce(sum(citizens_rescued), 0)::bigint AS rescued
			FROM incidents GROUP BY moderation, coalesce(status, 'reported'), type_id`,
		stored: `SELECT moderation, status, type_id, incidents, citizens_rescued FROM incident_stats WHERE incidents <> 0 OR citizens_rescued <> 0`,
	},
	{
		name: "reporter_stats",
		expected: `SELECT reporter_id, coalesce(status, 'reported') AS status, count(*) AS incidents
			FROM incidents WHERE reporter_id IS NOT NULL GROUP BY reporter_id, coalesce(status, 'reported')`,
		stored: `SELECT reporter_id, status, incidents FROM reporter_stats WHERE incidents <> 0`,
	},
	{
		// Recalculé depuis les incidents eux-mêmes, pas depuis incident_villains.incident_approved : une copie
		// périmée de la modération serait vue ici.
		name: "villain_stats",
		expected: `SELECT iv.villain_id, count(*) AS incidents
			FROM incident_villains iv JOIN incidents i ON i.id = iv.incident_id
			WHERE i.moderation = 'approved' GROUP BY iv.villain_id`,
		stored: `SELECT villain_id, incidents FROM villain_stats WHERE incidents <> 0`,
	},
	{
		// Colonne dénormalisée plutôt que compteur, mais même contrôle : la date de résolution d'un incident est la fin
		// de sa dernière intervention (la plus récemment démarrée) s'il est réussi, recalculée ici depuis les
		// interventions et non lue dans les triggers.
		name: "incidents.resolved_at",
		expected: `SELECT i.id, CASE WHEN i.status = 'succeeded' THEN (
				SELECT n.ended_at FROM interventions n WHERE n.incident_id = i.id
				ORDER BY n.started_at DESC NULLS LAST, n.id DESC LIMIT 1) END AS resolved_at
			FROM incidents i`,
		stored: `SELECT id, resolved_at FROM incidents`,
	},
	{
		// Idem : les interventions réussies d'un incident approuvé, pas intervention_heroes.resolved_at.
		name: "hero_stats",
		expected: `SELECT ih.hero_id, count(*) AS interventions,
				count(*) FILTER (WHERE n.status = 'succeeded' AND i.moderation = 'approved') AS resolved
			FROM intervention_heroes ih
			JOIN interventions n ON n.id = ih.intervention_id
			JOIN incidents i ON i.id = n.incident_id
			WHERE ih.participation = 'confirmed' GROUP BY ih.hero_id`,
		stored: `SELECT hero_id, interventions, resolved FROM hero_stats WHERE interventions <> 0 OR resolved <> 0`,
	},
}

// AssertReadCounters échoue si un compteur de lecture (incident_stats, villain_stats, hero_stats) diffère de
// ce que donnerait un comptage direct des tables. Les tests qui écrivent des incidents ou des interventions
// l'appellent après leurs écritures : un chemin d'écriture qui contournerait les triggers serait vu ici.
func AssertReadCounters(t testing.TB, db *gorm.DB) {
	t.Helper()
	for _, check := range counterChecks {
		var differing int64
		query := `SELECT count(*) FROM (
			(` + check.expected + ` EXCEPT ` + check.stored + `)
			UNION ALL
			(` + check.stored + ` EXCEPT ` + check.expected + `)) AS differences`
		if err := db.Raw(query).Scan(&differing).Error; err != nil {
			t.Fatalf("compteur %s : %v", check.name, err)
		}
		if differing != 0 {
			t.Errorf("compteur %s : %d lignes diffèrent du comptage des tables", check.name, differing)
		}
	}
}
