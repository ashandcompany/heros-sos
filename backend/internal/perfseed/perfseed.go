// Package perfseed ajoute des incidents en masse à une base déjà remplie par cmd/seed, pour mesurer
// les performances à grand volume (5 millions d'incidents dans le sujet). Les lignes sont produites par
// PostgreSQL (INSERT … SELECT sur generate_series) : aucun aller-retour par ligne, et une exécution
// reproductible, sans tirage aléatoire.
package perfseed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// DefaultIncidents est le volume du sujet : 5 millions d'incidents.
	DefaultIncidents = 5_000_000
	// DefaultBatchSize borne la taille d'une instruction : chaque lot est une transaction, et un échec
	// ou une interruption ne laisse jamais un lot à moitié écrit.
	DefaultBatchSize = 100_000
	// DefaultSpan étale les incidents sur cinq ans, du plus ancien au plus récent.
	DefaultSpan = 5 * 365 * 24 * time.Hour

	minIncidents = 1_000
	// maxExistingIncidents : au-delà, la base n'est plus « fraîche » (cmd/seed en crée une quarantaine).
	maxExistingIncidents = 10_000
)

var (
	// ErrMissingReference indique qu'il manque des comptes ou des vilains : lancer d'abord cmd/seed.
	ErrMissingReference = errors.New("perfseed : données de référence absentes")
	// ErrNotFresh indique que la base contient déjà beaucoup d'incidents.
	ErrNotFresh = errors.New("perfseed : la base contient déjà des incidents")
	// ErrSizeLimit indique que la taille maximale de la base est atteinte.
	ErrSizeLimit = errors.New("perfseed : taille maximale de la base atteinte")
)

// Options règle la génération.
type Options struct {
	// Incidents est le nombre d'incidents à créer (DefaultIncidents si nul).
	Incidents int64
	// BatchSize est la taille d'un lot (DefaultBatchSize si nul).
	BatchSize int64
	// Span est la durée sur laquelle les incidents sont étalés (DefaultSpan si nul).
	Span time.Duration
	// Now est la date du dernier incident (maintenant si nulle).
	Now time.Time
	// Append autorise l'ajout à une base qui contient déjà des incidents.
	Append bool
	// MaxDatabaseBytes arrête la génération, entre deux lots, quand la base dépasse cette taille. Zéro :
	// pas de limite. C'est le garde-fou contre un disque qui se remplit.
	MaxDatabaseBytes int64
	// Progress est appelée après chaque lot.
	Progress func(Progress)
}

// Progress décrit l'avancement de la génération.
type Progress struct {
	Done, Total   int64
	Elapsed       time.Duration
	DatabaseBytes int64
}

// reference compte ce que la génération réutilise.
type reference struct {
	Types, Citizens, Heroes, Villains int64
}

// Load ajoute les incidents, avec leurs interventions, leurs héros et leurs vilains. La répartition
// reproduit celle d'une vraie base : les incidents sont insérés dans l'ordre chronologique, 92 % des
// incidents clos réussissent, quelques-uns seulement (les plus récents) sont encore ouverts, et les
// interventions en cours se répartissent sur les héros qui n'en ont aucune.
func Load(ctx context.Context, db *gorm.DB, opts Options) error {
	opts = opts.withDefaults()
	if opts.Incidents < minIncidents {
		return fmt.Errorf("perfseed : au moins %d incidents sont nécessaires", minIncidents)
	}
	conn, err := db.DB()
	if err != nil {
		return fmt.Errorf("perfseed : accès au pool : %w", err)
	}
	if err := checkReference(ctx, conn); err != nil {
		return err
	}
	if !opts.Append {
		if err := checkFresh(ctx, conn); err != nil {
			return err
		}
	}

	step := opts.Span.Microseconds() / opts.Incidents
	start := opts.Now.Add(-opts.Span)
	began := time.Now()
	for done := int64(0); done < opts.Incidents; {
		size, err := databaseSize(ctx, conn)
		if err != nil {
			return err
		}
		if opts.MaxDatabaseBytes > 0 && size > opts.MaxDatabaseBytes {
			return fmt.Errorf("%w : %d incidents créés sur %d", ErrSizeLimit, done, opts.Incidents)
		}
		high := min(done+opts.BatchSize, opts.Incidents)
		var created int64
		if err := conn.QueryRowContext(ctx, loadBatchSQL, done, high, opts.Incidents, start, step).Scan(&created); err != nil {
			return fmt.Errorf("perfseed : lot %d à %d : %w", done, high, err)
		}
		done = high
		if opts.Progress != nil {
			opts.Progress(Progress{Done: done, Total: opts.Incidents, Elapsed: time.Since(began), DatabaseBytes: size})
		}
	}
	return nil
}

func (o Options) withDefaults() Options {
	if o.Incidents == 0 {
		o.Incidents = DefaultIncidents
	}
	if o.BatchSize == 0 {
		o.BatchSize = DefaultBatchSize
	}
	if o.Span == 0 {
		o.Span = DefaultSpan
	}
	if o.Now.IsZero() {
		o.Now = time.Now().UTC()
	}
	return o
}

func checkReference(ctx context.Context, conn *sql.DB) error {
	var ref reference
	err := conn.QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM incident_types),
		(SELECT count(*) FROM users WHERE role = 'citizen' AND status = 'active'),
		(SELECT count(*) FROM hero_profiles WHERE verification_status = 'approved'),
		(SELECT count(*) FROM villains)`).Scan(&ref.Types, &ref.Citizens, &ref.Heroes, &ref.Villains)
	if err != nil {
		return fmt.Errorf("perfseed : lecture des données de référence : %w", err)
	}
	for _, missing := range []struct {
		count int64
		label string
	}{
		{ref.Types, "type d'incident"}, {ref.Citizens, "citoyen actif"},
		{ref.Heroes, "super-héros approuvé"}, {ref.Villains, "vilain"},
	} {
		if missing.count == 0 {
			return fmt.Errorf("%w : aucun %s (lancez d'abord cmd/seed)", ErrMissingReference, missing.label)
		}
	}
	return nil
}

func checkFresh(ctx context.Context, conn *sql.DB) error {
	var existing int64
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM incidents`).Scan(&existing); err != nil {
		return fmt.Errorf("perfseed : comptage des incidents : %w", err)
	}
	if existing > maxExistingIncidents {
		return fmt.Errorf("%w (%d) : prévu pour une base fraîche, ajoutez l'option -append pour continuer", ErrNotFresh, existing)
	}
	return nil
}

func databaseSize(ctx context.Context, conn *sql.DB) (int64, error) {
	var size int64
	if err := conn.QueryRowContext(ctx, `SELECT pg_database_size(current_database())`).Scan(&size); err != nil {
		return 0, fmt.Errorf("perfseed : taille de la base : %w", err)
	}
	return size, nil
}

// loadTables sont les tables remplies par Load.
var loadTables = []string{"incidents", "interventions", "intervention_heroes", "incident_villains"}

// Settle met la base dans l'état d'une base installée depuis longtemps : statistiques à jour (le
// planificateur choisit alors ses plans comme en production) et carte de visibilité renseignée (les
// parcours d'index seuls deviennent possibles). À lancer après Load.
func Settle(ctx context.Context, db *gorm.DB) error {
	conn, err := db.DB()
	if err != nil {
		return fmt.Errorf("perfseed : accès au pool : %w", err)
	}
	for _, table := range loadTables {
		if _, err := conn.ExecContext(ctx, `VACUUM (ANALYZE) `+table); err != nil {
			return fmt.Errorf("perfseed : VACUUM de %s : %w", table, err)
		}
	}
	return nil
}

// TableSize est le volume d'une table, index compris.
type TableSize struct {
	Name       string
	Rows       int64
	TotalBytes int64
}

// Sizes retourne le nombre de lignes et la taille de chaque table remplie par Load.
func Sizes(ctx context.Context, db *gorm.DB) ([]TableSize, error) {
	conn, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("perfseed : accès au pool : %w", err)
	}
	sizes := make([]TableSize, 0, len(loadTables))
	for _, table := range loadTables {
		size := TableSize{Name: table}
		err := conn.QueryRowContext(ctx,
			`SELECT (SELECT count(*) FROM `+table+`), pg_total_relation_size($1::regclass)`, table).
			Scan(&size.Rows, &size.TotalBytes)
		if err != nil {
			return nil, fmt.Errorf("perfseed : taille de %s : %w", table, err)
		}
		sizes = append(sizes, size)
	}
	return sizes, nil
}

// loadBatchSQL crée un lot d'incidents [$1, $2) sur un total de $3, le premier à la date $4 et un
// toutes les $5 microsecondes.
//
// Un incident clos (92 % réussis, 8 % échoués) a une intervention terminée : un héros en tête, et un
// deuxième dans trois cas sur dix. Les derniers incidents (un sur 500) sont « signalés », sans
// intervention. Tout à la fin, un incident par héros libre est « en cours » avec une intervention
// active : un héros n'en mène qu'une à la fois. Six incidents sur dix ont un vilain.
//
// Les identifiants sont des UUID v7 comme ceux de l'application. hashint8 donne un entier stable par
// ligne : la même base se régénère à l'identique. incident_created_at et resolved_at sont fournis : les
// triggers qui les remplissent d'habitude ne voient pas les lignes insérées par la même instruction. Les
// compteurs (incident_stats, villain_stats, hero_stats) se mettent à jour d'eux-mêmes, une fois par lot.
const loadBatchSQL = `
WITH ref AS MATERIALIZED (
	SELECT $1::bigint AS lo, $2::bigint AS hi, $3::bigint AS total, $4::timestamptz AS start, $5::bigint AS step,
	       (SELECT array_agg(id ORDER BY id) FROM incident_types) AS types,
	       (SELECT array_agg(id ORDER BY id) FROM users WHERE role = 'citizen' AND status = 'active') AS reporters,
	       (SELECT array_agg(user_id ORDER BY user_id) FROM hero_profiles WHERE verification_status = 'approved') AS heroes,
	       (SELECT coalesce(array_agg(hero_profiles.user_id ORDER BY hero_profiles.user_id), '{}'::uuid[])
	          FROM hero_profiles
	         WHERE hero_profiles.verification_status = 'approved'
	           AND NOT EXISTS (SELECT 1 FROM intervention_heroes ih WHERE ih.hero_id = hero_profiles.user_id AND ih.is_active)) AS free_heroes,
	       (SELECT array_agg(id ORDER BY id) FROM villains) AS villains
), base AS MATERIALIZED (
	SELECT g,
	       uuidv7() AS id,
	       uuidv7() AS intervention_id,
	       (hashint8(g)::bigint & 2147483647) AS h,
	       ref.start + (g * ref.step)::float8 * interval '1 microsecond' AS created_at
	FROM ref, generate_series(ref.lo, ref.hi - 1) AS g
), gen AS MATERIALIZED (
	SELECT base.*,
	       ref.types[1 + (h % cardinality(ref.types))::int] AS type_id,
	       ref.reporters[1 + (g % cardinality(ref.reporters))::int] AS reporter_id,
	       CASE WHEN g >= ref.total - cardinality(ref.free_heroes)
	            THEN ref.free_heroes[1 + (g - (ref.total - cardinality(ref.free_heroes)))::int]
	            ELSE ref.heroes[1 + (g % cardinality(ref.heroes))::int] END AS lead_id,
	       ref.heroes[1 + ((g + 1) % cardinality(ref.heroes))::int] AS member_id,
	       cardinality(ref.heroes) AS hero_count,
	       ref.villains[1 + (h % cardinality(ref.villains))::int] AS villain_id,
	       CASE WHEN g >= ref.total - cardinality(ref.free_heroes) THEN 'in_progress'
	            WHEN g >= ref.total - ref.total / 500 THEN 'reported'
	            WHEN h % 100 < 92 THEN 'succeeded'
	            ELSE 'failed' END AS status,
	       base.created_at + interval '10 minutes' AS started_at,
	       base.created_at + interval '10 minutes' + (30 + h % 300) * interval '1 minute' AS ended_at
	FROM base CROSS JOIN ref
), ins_incidents AS (
	INSERT INTO incidents (id, reporter_id, type_id, title, description, status, priority, citizens_rescued,
		street, postal_code, city, country, latitude, longitude, occurred_at, created_at, updated_at)
	SELECT id, reporter_id, type_id,
	       'Incident de charge n° ' || g,
	       'Signalement généré pour les mesures de performance : ' || md5(g::text) || md5((g + 1)::text),
	       status,
	       CASE WHEN (h / 10) % 10 = 0 THEN 'high' WHEN (h / 10) % 10 = 1 THEN 'low' ELSE 'normal' END,
	       CASE WHEN status = 'succeeded' THEN (h % 7)::smallint ELSE 0::smallint END,
	       'Rue de la Charge ' || (h % 5000),
	       lpad((10000 + h % 80000)::text, 5, '0'),
	       (ARRAY['Paris', 'Lyon', 'Marseille', 'Toulouse', 'Nice', 'Nantes', 'Strasbourg', 'Montpellier', 'Bordeaux', 'Lille'])[1 + ((h / 100) % 10)::int],
	       'FR',
	       (42 + (h % 5000) / 500.0)::numeric(9, 6),
	       (-1 + (h % 7000) / 700.0)::numeric(9, 6),
	       created_at - interval '10 minutes',
	       created_at,
	       CASE WHEN status IN ('succeeded', 'failed') THEN ended_at ELSE created_at END
	FROM gen
), ins_interventions AS (
	INSERT INTO interventions (id, incident_id, status, started_at, ended_at)
	SELECT intervention_id, id, status, started_at, CASE WHEN status <> 'in_progress' THEN ended_at END
	FROM gen
	WHERE status <> 'reported'
), ins_heroes AS (
	INSERT INTO intervention_heroes (intervention_id, hero_id, role, participation, invited_by, outcome_validation,
		is_active, invited_at, joined_at, resolved_at)
	SELECT intervention_id, lead_id, 'lead', 'confirmed', NULL::uuid,
	       CASE WHEN status = 'in_progress' THEN 'pending' ELSE 'approved' END,
	       status = 'in_progress', NULL::timestamptz, started_at,
	       CASE WHEN status = 'succeeded' THEN ended_at END
	FROM gen
	WHERE status <> 'reported'
	UNION ALL
	SELECT intervention_id, member_id, 'member', 'confirmed', lead_id, 'approved',
	       FALSE, started_at + interval '1 minute', started_at + interval '3 minutes',
	       CASE WHEN status = 'succeeded' THEN ended_at END
	FROM gen
	WHERE status IN ('succeeded', 'failed') AND hero_count >= 2 AND h % 10 < 3
), ins_villains AS (
	INSERT INTO incident_villains (incident_id, villain_id, arrested, incident_created_at)
	SELECT id, villain_id, status = 'succeeded' AND h % 3 = 0, created_at
	FROM gen
	WHERE h % 10 < 6
)
SELECT count(*) FROM gen`
