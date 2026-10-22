package hero

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/avatar"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

const (
	tableHeroProfiles    = "hero_profiles"
	aliasUniqueConstrain = "hero_profiles_alias_key"
)

type profileRecord struct {
	UserID             uuid.UUID `gorm:"primaryKey"`
	Alias              string
	Bio                string
	RequestMessage     string
	VerificationStatus string
	ReviewedBy         *uuid.UUID
	ReviewedAt         *time.Time
	RejectionReason    *string
	LocationConsentAt  *time.Time
	CreatedAt          time.Time
}

func (profileRecord) TableName() string { return tableHeroProfiles }

type publicProfileRow struct {
	UserID             uuid.UUID `gorm:"column:user_id"`
	Alias              string    `gorm:"column:alias"`
	AvatarPath         *string   `gorm:"column:avatar_path"`
	Bio                string    `gorm:"column:bio"`
	AverageRating      float64   `gorm:"column:average_rating"`
	InterventionsCount int64     `gorm:"column:interventions_count"`
}

// toPublicProfile est une conversion : si PublicProfile et la ligne lue divergent, la compilation échoue.
func (r publicProfileRow) toPublicProfile() PublicProfile { return PublicProfile(r) }

// directoryRow : le profil est un champ exporté (et non un type embarqué anonyme), sans quoi GORM
// l'ignorerait.
type directoryRow struct {
	Profile            publicProfileRow `gorm:"embedded"`
	Email              string           `gorm:"column:email"`
	AccountStatus      string           `gorm:"column:account_status"`
	PendingEmail       string           `gorm:"column:pending_email"`
	RegisteredAt       time.Time        `gorm:"column:registered_at"`
	LastInterventionAt *time.Time       `gorm:"column:last_intervention_at"`
}

func (r directoryRow) toDirectoryEntry() DirectoryEntry {
	return DirectoryEntry{
		PublicProfile: r.Profile.toPublicProfile(), Email: r.Email, AccountStatus: user.Status(r.AccountStatus),
		PendingEmail: r.PendingEmail, RegisteredAt: r.RegisteredAt, LastInterventionAt: r.LastInterventionAt,
	}
}

// resolveAvatarURL transforme la valeur stockée dans users.avatar_path en adresse publique, ou nil si
// elle n'en est pas une. Deux formes existent : le nom d'une photo envoyée par PUT /me/avatar, servie
// sous avatarURLPrefix, et une adresse déjà publique (URL http(s) ou chemin absolu sur le site).
// Tout le reste (chemin de disque, URL piégée) est refusé : le stockage interne ne fuit jamais.
func resolveAvatarURL(avatarURLPrefix string, stored *string) *string {
	if stored == nil {
		return nil
	}
	value := strings.TrimSpace(*stored)
	if value == "" {
		return nil
	}
	if avatar.IsValidName(value) {
		served := avatarURLPrefix + value
		return &served
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil
	}
	if parsed.IsAbs() {
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
			return nil
		}
	} else if parsed.Host != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") ||
		strings.Contains(value, `\`) || path.Clean(parsed.Path) != parsed.Path {
		return nil
	}
	return &value
}

func (r profileRecord) toProfile() Profile {
	profile := Profile{
		UserID: r.UserID, Alias: r.Alias, Bio: r.Bio, RequestMessage: r.RequestMessage,
		Status: VerificationStatus(r.VerificationStatus), ReviewedBy: r.ReviewedBy,
		ReviewedAt: r.ReviewedAt, LocationConsentAt: r.LocationConsentAt, CreatedAt: r.CreatedAt,
	}
	if r.RejectionReason != nil {
		profile.RejectionReason = *r.RejectionReason
	}
	return profile
}

// Repository est le stockage des profils de super-héros.
type Repository interface {
	// Request dépose une demande (ou la redépose après un refus). Il retourne ErrAlreadyPending,
	// ErrAlreadyHero ou ErrAliasTaken quand la demande ne peut pas être enregistrée.
	Request(ctx context.Context, userID uuid.UUID, alias, bio, message string, now time.Time) error
	Find(ctx context.Context, userID uuid.UUID) (*Profile, error)
	List(ctx context.Context, status VerificationStatus, limit, offset int) ([]Profile, int64, error)
	// ListPublic retourne les profils approuvés des comptes actifs, triés selon l'ordre indiqué.
	ListPublic(ctx context.Context, sort PublicSort, limit, offset int) ([]PublicProfile, int64, error)
	// FindPublic retourne un profil uniquement s'il est approuvé et appartient à un compte actif.
	FindPublic(ctx context.Context, userID uuid.UUID) (*PublicProfile, error)
	// ListDirectory retourne les super-héros approuvés, quel que soit l'état de leur compte, par pseudo.
	// Il ne contient que ce qui se lit en SQL : l'identité civile, chiffrée, s'ajoute ensuite.
	ListDirectory(ctx context.Context, limit, offset int) ([]DirectoryEntry, int64, error)
	// Review fait passer une demande de « en attente » à approved ou rejected. Il retourne
	// false si la demande n'était plus en attente : deux modérateurs ne peuvent pas la traiter
	// tous les deux.
	Review(ctx context.Context, userID, reviewer uuid.UUID, to VerificationStatus, reason string, now time.Time) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

// requestSQL insère la demande, ou la remplace si l'utilisateur en avait une refusée. Une demande
// en attente ou acceptée n'est jamais écrasée : la clause WHERE ne laisse passer que « rejected ».
const requestSQL = `
INSERT INTO hero_profiles (user_id, alias, bio, request_message, verification_status, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    alias = EXCLUDED.alias,
    bio = EXCLUDED.bio,
    request_message = EXCLUDED.request_message,
    verification_status = EXCLUDED.verification_status,
    reviewed_by = NULL,
    reviewed_at = NULL,
    rejection_reason = NULL,
    created_at = EXCLUDED.created_at
WHERE hero_profiles.verification_status = ?
RETURNING user_id`

func (r *gormRepository) Request(ctx context.Context, userID uuid.UUID, alias, bio, message string, now time.Time) error {
	var result struct{ UserID uuid.UUID }
	query := r.conn(ctx).Raw(requestSQL, userID, alias, bio, message, string(StatusPending), now, string(StatusRejected)).Scan(&result)
	if query.Error != nil {
		if database.IsUniqueViolation(query.Error, aliasUniqueConstrain) {
			return ErrAliasTaken
		}
		return fmt.Errorf("hero : dépôt de la demande : %w", query.Error)
	}
	if query.RowsAffected > 0 {
		return nil
	}

	// Aucune ligne écrite : l'utilisateur a déjà une demande en attente ou acceptée.
	existing, err := r.Find(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Status == StatusApproved {
		return ErrAlreadyHero
	}
	return ErrAlreadyPending
}

func (r *gormRepository) Find(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var rec profileRecord
	err := r.conn(ctx).Where("user_id = ?", userID).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("hero : lecture du profil : %w", err)
	}
	profile := rec.toProfile()
	return &profile, nil
}

func (r *gormRepository) List(ctx context.Context, status VerificationStatus, limit, offset int) ([]Profile, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Model(&profileRecord{})
		if status != "" {
			query = query.Where("verification_status = ?", string(status))
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : comptage des demandes : %w", err)
	}
	var records []profileRecord
	// Les demandes les plus anciennes d'abord : c'est la file de travail des modérateurs.
	if err := filtered().Order("created_at ASC, user_id ASC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : liste des demandes : %w", err)
	}
	profiles := make([]Profile, len(records))
	for i, rec := range records {
		profiles[i] = rec.toProfile()
	}
	return profiles, total, nil
}

// Les listes de héros calculent leurs statistiques (note moyenne, interventions) APRÈS avoir choisi les
// héros à montrer : le CTE `page` ne contient que la page demandée, et les agrégats ne lisent que les avis et
// les interventions de ces héros (`hero_id IN (SELECT user_id FROM page)`). Agréger d'abord tous les avis puis
// joindre coûtait un balayage complet de hero_reviews et intervention_heroes (avec un tri sur disque) à chaque
// requête : 44 ms pour 2 000 héros et 60 000 avis, contre 1 ms avec l'index intervention_heroes_hero_id_idx.
// Les interventions ne sont pas comptées ici : un héros très actif en a des centaines de milliers, et les compter à
// chaque affichage coûtait plusieurs secondes. Elles se lisent dans hero_stats, tenu à jour par des triggers (voir
// la migration create_read_counters). Seul l'annuaire des modérateurs (interventionsWithLastOfPage) les compte
// encore, avec la date de la dernière.
//
// Seul le tri par note doit connaître la note de tous les héros avant de couper : c'est le seul agrégat qui
// parcourt tous les avis (voir ratingsOfAll). Au-delà de quelques centaines de milliers d'avis, il faudra
// stocker la note moyenne sur hero_profiles plutôt que la recalculer.
const (
	publicColumns = `hero_profiles.user_id, hero_profiles.alias, users.avatar_path, hero_profiles.bio`
	publicFrom    = `FROM hero_profiles JOIN users ON users.id = hero_profiles.user_id`
	// publicWhere : les héros dont le profil est approuvé et le compte actif, les seuls visibles du public.
	publicWhere      = `WHERE hero_profiles.verification_status = 'approved' AND users.role = 'hero' AND users.status = 'active'`
	directoryColumns = publicColumns + `,
		users.email, users.status AS account_status, users.created_at AS registered_at,
		COALESCE(pending_change.new_email, '') AS pending_email`
	directoryFrom = publicFrom + `
		LEFT JOIN email_change_requests AS pending_change
			ON pending_change.user_id = hero_profiles.user_id AND pending_change.status = 'pending'`
	// directoryWhere : tous les héros approuvés, quel que soit l'état de leur compte (un compte désactivé
	// reste dans l'annuaire des modérateurs).
	directoryWhere = `WHERE hero_profiles.verification_status = 'approved' AND users.role = 'hero'`
	orderByAlias   = `hero_profiles.alias, hero_profiles.user_id`

	ratingsOfPage = `
		LEFT JOIN (
			SELECT hero_id, AVG(rating)::double precision AS average_rating
			FROM hero_reviews
			WHERE status = 'published' AND hero_id IN (SELECT user_id FROM page)
			GROUP BY hero_id
		) AS ratings ON ratings.hero_id = page.user_id`
	// ratingsOfAll sert au tri par note : il est joint avant la coupe, aux profils eux-mêmes.
	ratingsOfAll = `
		LEFT JOIN (
			SELECT hero_id, AVG(rating)::double precision AS average_rating
			FROM hero_reviews
			WHERE status = 'published'
			GROUP BY hero_id
		) AS ratings ON ratings.hero_id = hero_profiles.user_id`
	interventionsOfPage = `
		LEFT JOIN (
			SELECT hero_id, interventions AS interventions_count
			FROM hero_stats
			WHERE hero_id IN (SELECT user_id FROM page)
		) AS interventions ON interventions.hero_id = page.user_id`
	// interventionsWithLastOfPage ajoute la date de la dernière intervention (annuaire des modérateurs).
	interventionsWithLastOfPage = `
		LEFT JOIN (
			SELECT intervention_heroes.hero_id, COUNT(*) AS interventions_count,
				MAX(interventions.started_at) AS last_intervention_at
			FROM intervention_heroes
			JOIN interventions ON interventions.id = intervention_heroes.intervention_id
			WHERE intervention_heroes.participation = 'confirmed'
				AND intervention_heroes.hero_id IN (SELECT user_id FROM page)
			GROUP BY intervention_heroes.hero_id
		) AS interventions ON interventions.hero_id = page.user_id`

	ratingColumn        = `COALESCE(ratings.average_rating, 0) AS average_rating`
	interventionsColumn = `COALESCE(interventions.interventions_count, 0) AS interventions_count`
)

// listQuery assemble une requête de liste : le CTE `page` (les héros à montrer), ce qui le complète, puis le tri final.
func listQuery(pageSelect, pageFrom, pageTail, finalColumns, joins, finalTail string) string {
	return "WITH page AS (SELECT " + pageSelect + " " + pageFrom + " " + pageTail + ") SELECT page.*, " +
		finalColumns + " FROM page" + joins + " " + finalTail
}

var (
	publicByAliasSQL = listQuery(publicColumns, publicFrom, publicWhere+" ORDER BY "+orderByAlias+" LIMIT ? OFFSET ?",
		ratingColumn+", "+interventionsColumn, ratingsOfPage+interventionsOfPage, "ORDER BY page.alias, page.user_id")
	publicOneSQL = listQuery(publicColumns, publicFrom, publicWhere+" AND hero_profiles.user_id = ?",
		ratingColumn+", "+interventionsColumn, ratingsOfPage+interventionsOfPage, "")
	// Par note, la page contient déjà la note (calculée pour tous les héros avant la coupe) ; seules les
	// interventions sont comptées pour les héros de la page.
	publicByRatingSQL = listQuery(publicColumns+", "+ratingColumn, publicFrom+ratingsOfAll,
		publicWhere+" ORDER BY average_rating DESC, "+orderByAlias+" LIMIT ? OFFSET ?",
		interventionsColumn, interventionsOfPage, "ORDER BY page.average_rating DESC, page.alias, page.user_id")
	directoryByAliasSQL = listQuery(directoryColumns, directoryFrom, directoryWhere+" ORDER BY "+orderByAlias+" LIMIT ? OFFSET ?",
		ratingColumn+", "+interventionsColumn+", interventions.last_intervention_at",
		ratingsOfPage+interventionsWithLastOfPage, "ORDER BY page.alias, page.user_id")
)

// countPublic et countDirectory comptent sans les statistiques : elles ne changent pas le nombre de lignes.
func (r *gormRepository) countPublic(ctx context.Context) (int64, error) {
	var total int64
	err := r.conn(ctx).Table(tableHeroProfiles).
		Joins("JOIN users ON users.id = hero_profiles.user_id").
		Where("hero_profiles.verification_status = ? AND users.role = ? AND users.status = ?",
			string(StatusApproved), string(user.RoleHero), string(user.StatusActive)).
		Count(&total).Error
	return total, err
}

func (r *gormRepository) countDirectory(ctx context.Context) (int64, error) {
	var total int64
	err := r.conn(ctx).Table(tableHeroProfiles).
		Joins("JOIN users ON users.id = hero_profiles.user_id").
		Where("hero_profiles.verification_status = ? AND users.role = ?", string(StatusApproved), string(user.RoleHero)).
		Count(&total).Error
	return total, err
}

func (r *gormRepository) ListPublic(ctx context.Context, sort PublicSort, limit, offset int) ([]PublicProfile, int64, error) {
	var query string
	switch sort {
	case PublicSortAlias:
		query = publicByAliasSQL
	case PublicSortAverageRating:
		query = publicByRatingSQL
	default:
		return nil, 0, fmt.Errorf("hero : tri public non pris en charge %q", sort)
	}
	total, err := r.countPublic(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("hero : comptage des profils publics : %w", err)
	}
	var rows []publicProfileRow
	if err := r.conn(ctx).Raw(query, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : lecture des profils publics : %w", err)
	}
	profiles := make([]PublicProfile, len(rows))
	for i := range rows {
		profiles[i] = rows[i].toPublicProfile()
	}
	return profiles, total, nil
}

func (r *gormRepository) FindPublic(ctx context.Context, userID uuid.UUID) (*PublicProfile, error) {
	var rows []publicProfileRow
	if err := r.conn(ctx).Raw(publicOneSQL, userID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("hero : lecture du profil public : %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrProfileNotFound
	}
	profile := rows[0].toPublicProfile()
	return &profile, nil
}

func (r *gormRepository) ListDirectory(ctx context.Context, limit, offset int) ([]DirectoryEntry, int64, error) {
	total, err := r.countDirectory(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("hero : comptage de l'annuaire : %w", err)
	}
	var rows []directoryRow
	if err := r.conn(ctx).Raw(directoryByAliasSQL, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("hero : lecture de l'annuaire : %w", err)
	}
	entries := make([]DirectoryEntry, len(rows))
	for i := range rows {
		entries[i] = rows[i].toDirectoryEntry()
	}
	return entries, total, nil
}

func (r *gormRepository) Review(ctx context.Context, userID, reviewer uuid.UUID, to VerificationStatus, reason string, now time.Time) (bool, error) {
	values := map[string]any{
		"verification_status": string(to),
		"reviewed_by":         reviewer,
		"reviewed_at":         now,
		"rejection_reason":    nil,
	}
	if reason != "" {
		values["rejection_reason"] = reason
	}
	result := r.conn(ctx).Table(tableHeroProfiles).
		Where("user_id = ? AND verification_status = ?", userID, string(StatusPending)).
		Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("hero : examen de la demande : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}
