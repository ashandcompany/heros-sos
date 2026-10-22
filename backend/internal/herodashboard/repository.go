package herodashboard

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

// Repository calcule les statistiques du tableau de bord.
type Repository interface {
	Global(ctx context.Context) (Global, error)
	Personal(ctx context.Context, heroID uuid.UUID) (Personal, error)
	MainEnemies(ctx context.Context, heroID uuid.UUID, limit int) ([]Enemy, error)
	// HeroIDs liste les héros dont le tableau de bord peut être ouvert : approuvés et actifs (ceux de l'annuaire public).
	HeroIDs(ctx context.Context) ([]uuid.UUID, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

// Les statistiques ne comptent que les incidents approuvés par la modération : un incident en attente ou rejeté
// n'existe pas pour les héros (ils ne le voient pas dans leurs listes), même si une intervention y est liée.
//
// Les citoyens secourus et les incidents par type se lisent dans les compteurs incident_stats (tenus à jour par des
// triggers, migration create_read_counters) : les recalculer parcourait toute la table des incidents. Le nombre
// d'interventions terminées n'a pas de compteur : il parcourt les interventions, ce que le cache du service absorbe.
const globalSQL = `SELECT
	(SELECT COUNT(*) FROM interventions n JOIN incidents i ON i.id = n.incident_id
		WHERE n.status IN ('succeeded', 'failed') AND i.moderation = 'approved') AS interventions,
	(SELECT COALESCE(SUM(citizens_rescued), 0) FROM incident_stats
		WHERE status = 'succeeded' AND moderation = 'approved') AS rescued_citizens,
	(SELECT COUNT(*) FROM villains) AS villains_identified,
	(SELECT COUNT(*) FROM villains WHERE status = 'emprisonne') AS villains_imprisoned`

// incidentsByTypeSQL part du catalogue pour que les types sans incident apparaissent avec un compte de zéro.
const incidentsByTypeSQL = `SELECT t.code, t.label, COALESCE(SUM(s.incidents), 0) AS count
FROM incident_types t
LEFT JOIN incident_stats s ON s.type_id = t.id AND s.moderation = 'approved'
GROUP BY t.id, t.code, t.label
ORDER BY count DESC, t.label`

func (r *gormRepository) Global(ctx context.Context) (Global, error) {
	var global Global
	if err := r.conn(ctx).Raw(globalSQL).Scan(&global).Error; err != nil {
		return Global{}, fmt.Errorf("herodashboard : statistiques globales : %w", err)
	}
	if err := r.conn(ctx).Raw(incidentsByTypeSQL).Scan(&global.IncidentsByType).Error; err != nil {
		return Global{}, fmt.Errorf("herodashboard : incidents par type : %w", err)
	}
	return global, nil
}

// personalSQL joint trois sources indépendantes : les participations confirmées du héros, ses films et ses avis.
const personalSQL = `SELECT
	(SELECT COUNT(*) FROM intervention_heroes ih JOIN interventions n ON n.id = ih.intervention_id
		JOIN incidents i ON i.id = n.incident_id AND i.moderation = 'approved'
		WHERE ih.hero_id = @hero AND ih.participation = 'confirmed' AND n.status = 'succeeded') AS successful_interventions,
	(SELECT COUNT(*) FROM intervention_heroes ih JOIN interventions n ON n.id = ih.intervention_id
		JOIN incidents i ON i.id = n.incident_id AND i.moderation = 'approved'
		WHERE ih.hero_id = @hero AND ih.participation = 'confirmed' AND n.status = 'failed') AS failed_interventions,
	(SELECT COUNT(*) FROM hero_reviews WHERE hero_id = @hero AND status = 'published') AS ratings_count,
	(SELECT COALESCE(AVG(rating), 0) FROM hero_reviews WHERE hero_id = @hero AND status = 'published') AS average_rating,
	(SELECT COUNT(*) FROM hero_films WHERE hero_id = @hero) AS videos,
	(SELECT COUNT(*) FROM hero_films WHERE hero_id = @hero AND status = 'approved') AS published_videos`

func (r *gormRepository) Personal(ctx context.Context, heroID uuid.UUID) (Personal, error) {
	var personal Personal
	if err := r.conn(ctx).Raw(personalSQL, sql.Named("hero", heroID)).Scan(&personal).Error; err != nil {
		return Personal{}, fmt.Errorf("herodashboard : statistiques du héros : %w", err)
	}
	personal.Interventions = personal.SuccessfulInterventions + personal.FailedInterventions
	return personal, nil
}

// mainEnemiesSQL compte un vilain une fois par incident, même si une intervention échouée a été suivie d'une autre.
const mainEnemiesSQL = `SELECT v.id, v.alias, COUNT(DISTINCT iv.incident_id) AS encounters
FROM intervention_heroes ih
JOIN interventions n ON n.id = ih.intervention_id
JOIN incidents i ON i.id = n.incident_id AND i.moderation = 'approved'
JOIN incident_villains iv ON iv.incident_id = i.id
JOIN villains v ON v.id = iv.villain_id
WHERE ih.hero_id = ? AND ih.participation = 'confirmed'
GROUP BY v.id, v.alias
ORDER BY encounters DESC, v.alias
LIMIT ?`

const heroIDsSQL = `SELECT hero_profiles.user_id
FROM hero_profiles
JOIN users ON users.id = hero_profiles.user_id
WHERE hero_profiles.verification_status = 'approved' AND users.role = 'hero' AND users.status = 'active'
ORDER BY hero_profiles.user_id`

func (r *gormRepository) HeroIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := r.conn(ctx).Raw(heroIDsSQL).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("herodashboard : liste des héros : %w", err)
	}
	return ids, nil
}

func (r *gormRepository) MainEnemies(ctx context.Context, heroID uuid.UUID, limit int) ([]Enemy, error) {
	enemies := []Enemy{}
	if err := r.conn(ctx).Raw(mainEnemiesSQL, heroID, limit).Scan(&enemies).Error; err != nil {
		return nil, fmt.Errorf("herodashboard : principaux ennemis : %w", err)
	}
	return enemies, nil
}
