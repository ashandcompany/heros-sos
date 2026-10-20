package film

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
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
	ListHeroFilms(ctx context.Context, heroID *uuid.UUID, status string, limit, offset int) ([]HeroFilm, int64, error)
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

func (r *gormRepository) ListHeroFilms(ctx context.Context, heroID *uuid.UUID, status string, limit, offset int) ([]HeroFilm, int64, error) {
	query := r.conn(ctx).Table("films").Joins("JOIN hero_films ON hero_films.film_id = films.id").Joins(
		"JOIN hero_profiles ON hero_profiles.user_id = hero_films.hero_id",
	)
	if heroID != nil {
		query = query.Where("hero_films.hero_id = ?", *heroID)
	}
	if status != "" && status != "all" {
		query = query.Where("hero_films.status = ?", status)
	} else if status == "" && heroID == nil {
		query = query.Where("hero_films.status = ?", StatusApproved)
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
