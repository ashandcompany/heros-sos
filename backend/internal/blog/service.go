package blog

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const (
	entityType = "article"
	// AuthorName signe les articles : le blog parle au nom de l'équipe, pas d'un modérateur.
	AuthorName = "Équipe Heros-SOS"
	// slugAttempts borne les tentatives quand un slug est pris entre le contrôle et l'écriture.
	slugAttempts = 20
)

// Service porte les règles du blog.
type Service struct {
	repo  Repository
	audit audit.Recorder
	tx    database.Transactor
	clock clock.Clock
}

// NewService crée le service.
func NewService(repo Repository, recorder audit.Recorder, tx database.Transactor, clk clock.Clock) *Service {
	return &Service{repo: repo, audit: recorder, tx: tx, clock: clk}
}

// Input est la saisie d'un article.
type Input struct {
	Title   string
	Excerpt string
	Content string
	Status  string
}

type cleanInput struct {
	title, excerpt, content string
	status                  Status
}

func validate(in Input) (cleanInput, error) {
	var fields validation.Collector
	clean := cleanInput{title: strings.TrimSpace(in.Title), excerpt: strings.TrimSpace(in.Excerpt), status: Status(in.Status)}

	switch n := utf8.RuneCountInString(clean.title); {
	case n == 0:
		fields.Add("title", validation.CodeRequired)
	case n < MinTitleLength:
		fields.Add("title", validation.CodeTooShort)
	case n > MaxTitleLength:
		fields.Add("title", validation.CodeTooLong)
	}
	switch n := utf8.RuneCountInString(clean.excerpt); {
	case n == 0:
		fields.Add("excerpt", validation.CodeRequired)
	case n > MaxExcerptLength:
		fields.Add("excerpt", validation.CodeTooLong)
	}
	if !clean.status.Valid() {
		fields.Add("status", validation.CodeInvalid)
	}
	if len(in.Content) > MaxContentBytes {
		fields.Add("content", validation.CodeTooLong)
	} else {
		var textLength int
		clean.content, textLength = SanitizeHTML(in.Content)
		if textLength < MinContentTextLength {
			fields.Add("content", validation.CodeTooShort)
		}
	}
	if err := fields.Err(); err != nil {
		return cleanInput{}, err
	}
	return clean, nil
}

// Create enregistre un article. Le slug dérive du titre et reçoit un suffixe numérique s'il est pris.
func (s *Service) Create(ctx context.Context, authorID uuid.UUID, in Input, ip netip.Addr) (*Article, error) {
	clean, err := validate(in)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	article := &Article{
		ID: ids.New(), Title: clean.title, Excerpt: clean.excerpt, Content: clean.content,
		Status: clean.status, AuthorID: &authorID, AuthorName: AuthorName, CreatedAt: now, UpdatedAt: now,
	}
	if clean.status == StatusPublished {
		article.PublishedAt = &now
	}
	err = s.withUniqueSlug(ctx, article, article.Title, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, article); err != nil {
			return err
		}
		return s.record(ctx, authorID, article.ID, audit.ActionArticleCreated, ip)
	})
	if err != nil {
		return nil, err
	}
	return article, nil
}

// Update remplace le contenu d'un article. La date de première publication est conservée ; le
// slug ne change plus une fois l'article publié (les liens partagés restent valides).
func (s *Service) Update(ctx context.Context, actorID, id uuid.UUID, in Input, ip netip.Addr) (*Article, error) {
	clean, err := validate(in)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	updated := *current
	updated.Title, updated.Excerpt, updated.Content, updated.Status = clean.title, clean.excerpt, clean.content, clean.status
	updated.UpdatedAt = now
	if clean.status == StatusPublished && updated.PublishedAt == nil {
		updated.PublishedAt = &now
	}
	rename := clean.title != current.Title && current.PublishedAt == nil
	save := func(ctx context.Context) error {
		if err := s.repo.Update(ctx, &updated); err != nil {
			return err
		}
		return s.record(ctx, actorID, id, audit.ActionArticleUpdated, ip)
	}
	if rename {
		err = s.withUniqueSlug(ctx, &updated, clean.title, save)
	} else {
		err = s.tx.WithinTx(ctx, save)
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete supprime définitivement un article.
func (s *Service) Delete(ctx context.Context, actorID, id uuid.UUID, ip netip.Addr) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Delete(ctx, id); err != nil {
			return err
		}
		return s.record(ctx, actorID, id, audit.ActionArticleDeleted, ip)
	})
}

// Get retourne un article, brouillon compris (back-office).
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Article, error) {
	return s.repo.Find(ctx, id)
}

// GetPublished retourne un article publié ; un brouillon est introuvable.
func (s *Service) GetPublished(ctx context.Context, slug string) (*Article, error) {
	return s.repo.FindPublishedBySlug(ctx, slug)
}

// ListPublished retourne une page d'articles publiés.
func (s *Service) ListPublished(ctx context.Context, limit, offset int) ([]Article, int64, error) {
	return s.repo.ListPublished(ctx, limit, offset)
}

// ListAll retourne une page de tous les articles, éventuellement filtrés par statut.
func (s *Service) ListAll(ctx context.Context, status string, limit, offset int) ([]Article, int64, error) {
	if status != "" && !Status(status).Valid() {
		return nil, 0, &validation.Error{Fields: map[string]string{"status": validation.CodeInvalid}}
	}
	return s.repo.ListAll(ctx, Status(status), limit, offset)
}

// withUniqueSlug choisit un slug libre pour article puis exécute write dans une transaction. Si
// un autre article prend le slug entre-temps, la transaction échoue et on retente avec le suivant.
func (s *Service) withUniqueSlug(ctx context.Context, article *Article, title string, write func(context.Context) error) error {
	base := Slugify(title)
	for attempt := 0; attempt < slugAttempts; attempt++ {
		candidate := slugCandidate(base, attempt)
		taken, err := s.repo.SlugTaken(ctx, candidate, article.ID)
		if err != nil {
			return err
		}
		if taken {
			continue
		}
		article.Slug = candidate
		err = s.tx.WithinTx(ctx, write)
		if errors.Is(err, ErrSlugTaken) {
			continue
		}
		return err
	}
	return ErrSlugTaken
}

func (s *Service) record(ctx context.Context, actorID, id uuid.UUID, action string, ip netip.Addr) error {
	return s.audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: entityType, EntityID: &id, IP: ip,
	})
}
