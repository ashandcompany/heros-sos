package film

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
)

const (
	tableFilms      = "films"
	imdbUniqueIndex = "films_imdb_id_key"
)

type filmRecord struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	IMDBID      *string
	VideoURL    *string
	Title       string
	ReleaseYear *int16
	PosterURL   *string
	Plot        string
	Raw         *string `gorm:"type:jsonb"`
	FetchedAt   *time.Time
	CreatedAt   time.Time
}

func (filmRecord) TableName() string { return tableFilms }

func toFilm(record filmRecord) Film {
	film := Film{
		ID: record.ID, Title: record.Title, ReleaseYear: record.ReleaseYear,
		Plot: record.Plot, FetchedAt: record.FetchedAt, CreatedAt: record.CreatedAt,
	}
	if record.IMDBID != nil {
		film.IMDBID = *record.IMDBID
	}
	if record.VideoURL != nil {
		film.VideoURL = *record.VideoURL
	}
	if record.PosterURL != nil {
		film.PosterURL = *record.PosterURL
	}
	if record.Raw != nil {
		film.Raw = *record.Raw
	}
	return film
}

func toRecord(film Film) filmRecord {
	record := filmRecord{
		ID: film.ID, Title: film.Title, ReleaseYear: film.ReleaseYear, Plot: film.Plot,
		FetchedAt: film.FetchedAt, CreatedAt: film.CreatedAt,
	}
	if film.IMDBID != "" {
		record.IMDBID = &film.IMDBID
	}
	if film.VideoURL != "" {
		record.VideoURL = &film.VideoURL
	}
	if film.PosterURL != "" {
		record.PosterURL = &film.PosterURL
	}
	if film.Raw != "" {
		record.Raw = &film.Raw
	}
	return record
}

type publicRow struct {
	ID           uuid.UUID
	IMDBID       *string
	VideoURL     *string
	Title        string
	ReleaseYear  *int16
	PosterURL    *string
	Plot         string
	Raw          *string
	FetchedAt    *time.Time
	CreatedAt    time.Time
	HeroID       uuid.UUID
	HeroAlias    string
	Status       string
	ReviewedBy   *uuid.UUID
	ReviewedAt   *time.Time
	AddedAt      time.Time
	VillainID    uuid.UUID
	VillainAlias string
	CreatedBy    *uuid.UUID
}

func (r publicRow) film() Film {
	return toFilm(filmRecord{
		ID: r.ID, IMDBID: r.IMDBID, VideoURL: r.VideoURL, Title: r.Title,
		ReleaseYear: r.ReleaseYear, PosterURL: r.PosterURL, Plot: r.Plot,
		Raw: r.Raw, FetchedAt: r.FetchedAt, CreatedAt: r.CreatedAt,
	})
}

type Repository interface {
	CreateFilm(ctx context.Context, film *Film) error
	FindFilm(ctx context.Context, id uuid.UUID) (*Film, error)
	FindFilmByIMDBID(ctx context.Context, imdbID string) (*Film, error)
	AddHeroFilm(ctx context.Context, heroID, filmID uuid.UUID, now time.Time) error
	AddApprovedHeroFilm(ctx context.Context, heroID, filmID, moderatorID uuid.UUID, now time.Time) error
	ReviewHeroFilm(ctx context.Context, heroID, filmID, moderatorID uuid.UUID, status string, now time.Time) error
	DeleteHeroFilm(ctx context.Context, heroID, filmID uuid.UUID) error
	CreateVillainFilm(ctx context.Context, villainID, filmID uuid.UUID, actorID *uuid.UUID, now time.Time) error
	DeleteVillainFilm(ctx context.Context, villainID, filmID uuid.UUID) error
	ListHeroFilms(ctx context.Context, filter HeroFilmFilter, limit, offset int) ([]HeroFilm, int64, error)
	ListPublicFilms(ctx context.Context, filter PublicFilmFilter, limit, offset int) ([]PublicFilm, int64, error)
	ListFilmHeroes(ctx context.Context) ([]FilmHero, error)
	ListVillainFilms(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]VillainFilm, int64, error)
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) CreateFilm(ctx context.Context, film *Film) error {
	record := toRecord(*film)
	if err := r.conn(ctx).Create(&record).Error; err != nil {
		if database.IsUniqueViolation(err, imdbUniqueIndex) {
			return fmt.Errorf("%w: %v", ErrIMDBExists, err)
		}
		return fmt.Errorf("film : création : %w", err)
	}
	*film = toFilm(record)
	return nil
}

func (r *gormRepository) FindFilm(ctx context.Context, id uuid.UUID) (*Film, error) {
	var record filmRecord
	err := r.conn(ctx).Where("id = ?", id).Take(&record).Error
	return filmResult(record, err)
}

func (r *gormRepository) FindFilmByIMDBID(ctx context.Context, imdbID string) (*Film, error) {
	var record filmRecord
	err := r.conn(ctx).Where("imdb_id = ?", imdbID).Take(&record).Error
	return filmResult(record, err)
}

func filmResult(record filmRecord, err error) (*Film, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("film : lecture : %w", err)
	}
	result := toFilm(record)
	return &result, nil
}

func (r *gormRepository) AddHeroFilm(ctx context.Context, heroID, filmID uuid.UUID, now time.Time) error {
	const query = `
INSERT INTO hero_films (hero_id, film_id, status, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (hero_id, film_id) DO UPDATE SET
    status = EXCLUDED.status,
    reviewed_by = NULL,
    reviewed_at = NULL,
    created_at = EXCLUDED.created_at
WHERE hero_films.status = ?`
	result := r.conn(ctx).Exec(query, heroID, filmID, StatusPending, now, StatusRejected)
	if result.Error != nil {
		return fmt.Errorf("film : association au héros : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyLinked
	}
	return nil
}

func (r *gormRepository) AddApprovedHeroFilm(ctx context.Context, heroID, filmID, moderatorID uuid.UUID, now time.Time) error {
	link := map[string]any{
		"hero_id": heroID, "film_id": filmID, "status": StatusApproved,
		"reviewed_by": moderatorID, "reviewed_at": now, "created_at": now,
	}
	result := r.conn(ctx).Table("hero_films").Clauses(clause.OnConflict{DoNothing: true}).Create(&link)
	if result.Error != nil {
		return fmt.Errorf("film : association modérée au héros : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyLinked
	}
	return nil
}

func (r *gormRepository) ReviewHeroFilm(ctx context.Context, heroID, filmID, moderatorID uuid.UUID, status string, now time.Time) error {
	result := r.conn(ctx).Table("hero_films").Where(
		"hero_id = ? AND film_id = ? AND status <> ?", heroID, filmID, status,
	).Updates(map[string]any{"status": status, "reviewed_by": moderatorID, "reviewed_at": now})
	if result.Error != nil {
		return fmt.Errorf("film : modération : %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var exists bool
	if err := r.conn(ctx).Raw("SELECT EXISTS (SELECT 1 FROM hero_films WHERE hero_id = ? AND film_id = ?)", heroID, filmID).Scan(&exists).Error; err != nil {
		return fmt.Errorf("film : vérification du lien : %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrAlreadyReview
}

func (r *gormRepository) DeleteHeroFilm(ctx context.Context, heroID, filmID uuid.UUID) error {
	result := r.conn(ctx).Table("hero_films").Where("hero_id = ? AND film_id = ?", heroID, filmID).Delete(map[string]any{})
	if result.Error != nil {
		return fmt.Errorf("film : suppression du lien héros : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) CreateVillainFilm(ctx context.Context, villainID, filmID uuid.UUID, actorID *uuid.UUID, now time.Time) error {
	link := map[string]any{"villain_id": villainID, "film_id": filmID, "created_by": actorID, "created_at": now}
	result := r.conn(ctx).Table("villain_films").Clauses(clause.OnConflict{DoNothing: true}).Create(&link)
	if result.Error != nil {
		return fmt.Errorf("film : association au vilain : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyLinked
	}
	return nil
}

func (r *gormRepository) DeleteVillainFilm(ctx context.Context, villainID, filmID uuid.UUID) error {
	result := r.conn(ctx).Table("villain_films").Where("villain_id = ? AND film_id = ?", villainID, filmID).Delete(map[string]any{})
	if result.Error != nil {
		return fmt.Errorf("film : suppression du lien vilain : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) ListHeroFilms(ctx context.Context, filter HeroFilmFilter, limit, offset int) ([]HeroFilm, int64, error) {
	query := r.conn(ctx).Table("films").Joins("JOIN hero_films ON hero_films.film_id = films.id").Joins(
		"JOIN hero_profiles ON hero_profiles.user_id = hero_films.hero_id",
	)
	if filter.HeroID != nil {
		query = query.Where("hero_films.hero_id = ?", *filter.HeroID)
	}
	if filter.Status != "" && filter.Status != "all" {
		query = query.Where("hero_films.status = ?", filter.Status)
	} else if filter.Status == "" && filter.HeroID == nil {
		query = query.Where("hero_films.status = ?", StatusApproved)
	}
	if filter.Title != "" {
		query = query.Where("films.title ILIKE ?", "%"+escapeLike(filter.Title)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("film : comptage des films de héros : %w", err)
	}
	var rows []publicRow
	selectColumns := `films.id, films.imdb_id, films.video_url, films.title, films.release_year,
films.poster_url, films.plot, films.raw, films.fetched_at, films.created_at,
hero_films.hero_id, hero_profiles.alias AS hero_alias, hero_films.status,
hero_films.reviewed_by, hero_films.reviewed_at, hero_films.created_at AS added_at`
	if err := query.Select(selectColumns).Order("hero_films.created_at DESC, films.id").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("film : liste des films de héros : %w", err)
	}
	items := make([]HeroFilm, len(rows))
	for index, row := range rows {
		items[index] = HeroFilm{
			Film: row.film(), HeroID: row.HeroID, HeroAlias: row.HeroAlias,
			Status: row.Status, ReviewedBy: row.ReviewedBy, ReviewedAt: row.ReviewedAt, AddedAt: row.AddedAt,
		}
	}
	return items, total, nil
}

// publicLinks : rattachements approuvés à des héros dont le profil est validé (les seuls dont les
// films sont visibles des citoyens).
const publicLinks = `FROM films
JOIN hero_films ON hero_films.film_id = films.id AND hero_films.status = ?
JOIN hero_profiles ON hero_profiles.user_id = hero_films.hero_id AND hero_profiles.verification_status = ?`

type publicFilmRow struct {
	ID          uuid.UUID
	IMDBID      *string
	VideoURL    *string
	Title       string
	ReleaseYear *int16
	PosterURL   *string
	Plot        string
	CreatedAt   time.Time
	AddedAt     time.Time
	// Heroes : tableau JSON [{id, alias}].
	Heroes string
}

func (r *gormRepository) ListPublicFilms(ctx context.Context, filter PublicFilmFilter, limit, offset int) ([]PublicFilm, int64, error) {
	conditions := []string{"TRUE"}
	args := []any{StatusApproved, string(hero.StatusApproved)}
	if filter.HeroID != nil {
		// Sous-requête plutôt que filtre sur la jointure : le film garde la liste de tous ses héros.
		conditions = append(conditions, `EXISTS (SELECT 1 FROM hero_films linked
WHERE linked.film_id = films.id AND linked.hero_id = ? AND linked.status = ?)`)
		args = append(args, *filter.HeroID, StatusApproved)
	}
	if filter.Title != "" {
		conditions = append(conditions, "films.title ILIKE ?")
		args = append(args, "%"+escapeLike(filter.Title)+"%")
	}
	from := publicLinks + "\nWHERE " + strings.Join(conditions, " AND ")

	var total int64
	if err := r.conn(ctx).Raw("SELECT COUNT(DISTINCT films.id) "+from, args...).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("film : comptage de la vidéothèque publique : %w", err)
	}
	var rows []publicFilmRow
	query := `SELECT films.id, films.imdb_id, films.video_url, films.title, films.release_year,
films.poster_url, films.plot, films.created_at, MAX(hero_films.created_at) AS added_at,
json_agg(json_build_object('id', hero_films.hero_id, 'alias', hero_profiles.alias)
    ORDER BY hero_profiles.alias, hero_films.hero_id)::text AS heroes
` + from + `
GROUP BY films.id
ORDER BY added_at DESC, films.id
LIMIT ? OFFSET ?`
	if err := r.conn(ctx).Raw(query, append(args, limit, offset)...).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("film : vidéothèque publique : %w", err)
	}
	items := make([]PublicFilm, len(rows))
	for index, row := range rows {
		var heroes []HeroRef
		if err := json.Unmarshal([]byte(row.Heroes), &heroes); err != nil {
			return nil, 0, fmt.Errorf("film : héros du film %s : %w", row.ID, err)
		}
		items[index] = PublicFilm{
			Film: toFilm(filmRecord{
				ID: row.ID, IMDBID: row.IMDBID, VideoURL: row.VideoURL, Title: row.Title,
				ReleaseYear: row.ReleaseYear, PosterURL: row.PosterURL, Plot: row.Plot, CreatedAt: row.CreatedAt,
			}),
			Heroes: heroes, AddedAt: row.AddedAt,
		}
	}
	return items, total, nil
}

func (r *gormRepository) ListFilmHeroes(ctx context.Context) ([]FilmHero, error) {
	var rows []struct {
		ID        uuid.UUID
		Alias     string
		FilmCount int64
	}
	query := `SELECT hero_films.hero_id AS id, hero_profiles.alias, COUNT(*) AS film_count
` + publicLinks + `
GROUP BY hero_films.hero_id, hero_profiles.alias
ORDER BY hero_profiles.alias, hero_films.hero_id`
	if err := r.conn(ctx).Raw(query, StatusApproved, string(hero.StatusApproved)).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("film : héros de la vidéothèque : %w", err)
	}
	items := make([]FilmHero, len(rows))
	for index, row := range rows {
		items[index] = FilmHero{HeroRef: HeroRef{ID: row.ID, Alias: row.Alias}, FilmCount: row.FilmCount}
	}
	return items, nil
}

// escapeLike neutralise les caractères joker de LIKE (l'échappement par défaut de PostgreSQL
// est l'antislash).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(s)
}

func (r *gormRepository) ListVillainFilms(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]VillainFilm, int64, error) {
	query := r.conn(ctx).Table("films").Joins("JOIN villain_films ON villain_films.film_id = films.id").Joins(
		"JOIN villains ON villains.id = villain_films.villain_id",
	).Where("villain_films.villain_id = ?", villainID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("film : comptage des films de vilains : %w", err)
	}
	var rows []publicRow
	selectColumns := `films.id, films.imdb_id, films.video_url, films.title, films.release_year,
films.poster_url, films.plot, films.raw, films.fetched_at, films.created_at,
villain_films.villain_id, villains.alias AS villain_alias, villain_films.created_by,
villain_films.created_at AS added_at`
	if err := query.Select(selectColumns).Order("villain_films.created_at DESC, films.id").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("film : liste des films de vilains : %w", err)
	}
	items := make([]VillainFilm, len(rows))
	for index, row := range rows {
		items[index] = VillainFilm{
			Film: row.film(), VillainID: row.VillainID, VillainAlias: row.VillainAlias,
			CreatedBy: row.CreatedBy, AddedAt: row.AddedAt,
		}
	}
	return items, total, nil
}
