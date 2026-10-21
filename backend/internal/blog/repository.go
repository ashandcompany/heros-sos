package blog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableArticles    = "articles"
	slugUniqueKey    = "articles_slug_key"
	publishedOrder   = "published_at DESC, id DESC"
	updatedOrder     = "updated_at DESC, id DESC"
	summaryColumns   = "id, title, slug, excerpt, '' AS content, status, author_id, author_name, created_at, updated_at, published_at"
	publishedPredict = "status = 'publie' AND published_at IS NOT NULL"
)

// Repository est le stockage des articles.
type Repository interface {
	Create(ctx context.Context, a *Article) error
	Update(ctx context.Context, a *Article) error
	Delete(ctx context.Context, id uuid.UUID) error
	Find(ctx context.Context, id uuid.UUID) (*Article, error)
	// FindPublishedBySlug ne retourne que les articles publiés : un brouillon est ErrNotFound.
	FindPublishedBySlug(ctx context.Context, slug string) (*Article, error)
	// ListPublished retourne les articles publiés, du plus récent au plus ancien, sans contenu.
	ListPublished(ctx context.Context, limit, offset int) ([]Article, int64, error)
	// ListAll retourne tous les articles, les derniers modifiés d'abord, sans contenu. Un statut
	// non vide restreint la liste à ce statut.
	ListAll(ctx context.Context, status Status, limit, offset int) ([]Article, int64, error)
	SlugTaken(ctx context.Context, slug string, ignoreID uuid.UUID) (bool, error)
}

type record struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Title       string
	Slug        string
	Excerpt     string
	Content     string
	Status      string
	AuthorID    *uuid.UUID
	AuthorName  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

func (record) TableName() string { return tableArticles }

func newRecord(a *Article) record {
	return record{
		ID: a.ID, Title: a.Title, Slug: a.Slug, Excerpt: a.Excerpt, Content: a.Content,
		Status: string(a.Status), AuthorID: a.AuthorID, AuthorName: a.AuthorName,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, PublishedAt: a.PublishedAt,
	}
}

func (r record) toArticle() Article {
	return Article{
		ID: r.ID, Title: r.Title, Slug: r.Slug, Excerpt: r.Excerpt, Content: r.Content,
		Status: Status(r.Status), AuthorID: r.AuthorID, AuthorName: r.AuthorName,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, PublishedAt: r.PublishedAt,
	}
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository crée le dépôt adossé à PostgreSQL.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *gormRepository) Create(ctx context.Context, a *Article) error {
	rec := newRecord(a)
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsUniqueViolation(err, slugUniqueKey) {
			return ErrSlugTaken
		}
		return fmt.Errorf("blog : création : %w", err)
	}
	return nil
}

func (r *gormRepository) Update(ctx context.Context, a *Article) error {
	result := r.conn(ctx).Table(tableArticles).Where("id = ?", a.ID).Updates(map[string]any{
		"title": a.Title, "slug": a.Slug, "excerpt": a.Excerpt, "content": a.Content,
		"status": string(a.Status), "updated_at": a.UpdatedAt, "published_at": a.PublishedAt,
	})
	if result.Error != nil {
		if database.IsUniqueViolation(result.Error, slugUniqueKey) {
			return ErrSlugTaken
		}
		return fmt.Errorf("blog : mise à jour : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.conn(ctx).Where("id = ?", id).Delete(&record{})
	if result.Error != nil {
		return fmt.Errorf("blog : suppression : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) Find(ctx context.Context, id uuid.UUID) (*Article, error) {
	return r.take(r.conn(ctx).Where("id = ?", id))
}

func (r *gormRepository) FindPublishedBySlug(ctx context.Context, slug string) (*Article, error) {
	return r.take(r.conn(ctx).Where("slug = ? AND "+publishedPredict, slug))
}

func (r *gormRepository) take(query *gorm.DB) (*Article, error) {
	var rec record
	err := query.Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("blog : lecture : %w", err)
	}
	article := rec.toArticle()
	return &article, nil
}

func (r *gormRepository) ListPublished(ctx context.Context, limit, offset int) ([]Article, int64, error) {
	return r.list(ctx, publishedPredict, nil, publishedOrder, limit, offset)
}

func (r *gormRepository) ListAll(ctx context.Context, status Status, limit, offset int) ([]Article, int64, error) {
	if status == "" {
		return r.list(ctx, "", nil, updatedOrder, limit, offset)
	}
	return r.list(ctx, "status = ?", []any{string(status)}, updatedOrder, limit, offset)
}

func (r *gormRepository) list(ctx context.Context, where string, args []any, order string, limit, offset int) ([]Article, int64, error) {
	filtered := func() *gorm.DB {
		query := r.conn(ctx).Table(tableArticles)
		if where != "" {
			query = query.Where(where, args...)
		}
		return query
	}
	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("blog : comptage : %w", err)
	}
	var records []record
	if err := filtered().Select(summaryColumns).Order(order).Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("blog : liste : %w", err)
	}
	articles := make([]Article, len(records))
	for i, rec := range records {
		articles[i] = rec.toArticle()
	}
	return articles, total, nil
}

func (r *gormRepository) SlugTaken(ctx context.Context, slug string, ignoreID uuid.UUID) (bool, error) {
	var count int64
	err := r.conn(ctx).Table(tableArticles).Where("slug = ? AND id <> ?", slug, ignoreID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("blog : slug : %w", err)
	}
	return count > 0, nil
}
