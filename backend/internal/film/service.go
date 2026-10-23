package film

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

const (
	entityType = "hero_film"
	// titleQueryMaxLength borne la recherche par titre dans les listes de films.
	titleQueryMaxLength = 100
)

var imdbIDPattern = regexp.MustCompile(`^tt[0-9]{7,12}$`)

type HeroFinder interface {
	Find(ctx context.Context, userID uuid.UUID) (*hero.Profile, error)
}

type VillainFinder interface {
	Find(ctx context.Context, id uuid.UUID) (*villain.Villain, error)
}

type ServiceDeps struct {
	Repo     Repository
	Catalog  Catalog
	Heroes   HeroFinder
	Villains VillainFinder
	Audit    audit.Recorder
	Tx       database.Transactor
	Clock    clock.Clock
}

type Service struct{ ServiceDeps }

func NewService(deps ServiceDeps) *Service { return &Service{ServiceDeps: deps} }

type FilmInput struct {
	IMDBID      string
	FilmID      uuid.UUID
	Title       string
	Description string
	VideoURL    string
}

func (s *Service) Search(ctx context.Context, query string) ([]Title, error) {
	query, code := validation.RequiredText(query, 100)
	if code != "" {
		return nil, &validation.Error{Fields: map[string]string{"q": code}}
	}
	return s.Catalog.Search(ctx, query)
}

func (s *Service) FindTitle(ctx context.Context, imdbID string) (*Title, error) {
	if !imdbIDPattern.MatchString(strings.TrimSpace(imdbID)) {
		return nil, ErrTitleNotFound
	}
	title, _, err := s.Catalog.Get(ctx, imdbID)
	return title, err
}

func (s *Service) AddToHero(ctx context.Context, actorID uuid.UUID, role user.Role, heroID uuid.UUID, in FilmInput, ip netip.Addr) (*Film, error) {
	if role != user.RoleHero {
		return nil, ErrNotAllowed
	}
	heroID = actorID
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return nil, err
	}
	film, err := s.resolveFilm(ctx, in)
	if err != nil {
		return nil, err
	}
	addedAt := s.Clock.Now()
	if err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.AddHeroFilm(ctx, heroID, film.ID, addedAt); err != nil {
			return err
		}
		return s.record(ctx, actorID, audit.ActionHeroFilmAdded, entityType, film.ID, ip, map[string]any{
			"heroId": heroID.String(), "status": StatusPending,
		})
	}); err != nil {
		return nil, err
	}
	film.CreatedAt = addedAt
	return film, nil
}

func (s *Service) AddApprovedToHero(ctx context.Context, moderatorID, heroID uuid.UUID, in FilmInput, ip netip.Addr) (*Film, error) {
	if err := s.requireApprovedHero(ctx, heroID); err != nil {
		return nil, err
	}
	film, err := s.resolveFilm(ctx, in)
	if err != nil {
		return nil, err
	}
	addedAt := s.Clock.Now()
	if err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.AddApprovedHeroFilm(ctx, heroID, film.ID, moderatorID, addedAt); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, audit.ActionHeroFilmApproved, entityType, film.ID, ip,
			map[string]any{"heroId": heroID.String(), "status": StatusApproved})
	}); err != nil {
		return nil, err
	}
	film.CreatedAt = addedAt
	return film, nil
}

func (s *Service) AddToVillain(ctx context.Context, moderatorID, villainID uuid.UUID, in FilmInput, ip netip.Addr) (*Film, error) {
	if _, err := s.Villains.Find(ctx, villainID); err != nil {
		return nil, err
	}
	film, err := s.resolveFilm(ctx, in)
	if err != nil {
		return nil, err
	}
	addedAt := s.Clock.Now()
	if err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.CreateVillainFilm(ctx, villainID, film.ID, &moderatorID, addedAt); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, audit.ActionVillainFilmLinked, "villain_film", film.ID, ip,
			map[string]any{"villainId": villainID.String()})
	}); err != nil {
		return nil, err
	}
	film.CreatedAt = addedAt
	return film, nil
}

func (s *Service) Moderate(ctx context.Context, moderatorID, heroID, filmID uuid.UUID, status string, ip netip.Addr) error {
	if status != StatusApproved && status != StatusRejected {
		return &validation.Error{Fields: map[string]string{"status": validation.CodeInvalid}}
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.ReviewHeroFilm(ctx, heroID, filmID, moderatorID, status, s.Clock.Now()); err != nil {
			return err
		}
		action := audit.ActionHeroFilmApproved
		if status == StatusRejected {
			action = audit.ActionHeroFilmRejected
		}
		return s.record(ctx, moderatorID, action, entityType, filmID, ip,
			map[string]any{"heroId": heroID.String(), "status": status})
	})
}

func (s *Service) DeleteHeroFilm(ctx context.Context, actorID, heroID, filmID uuid.UUID, role user.Role, ip netip.Addr) error {
	if role == user.RoleHero {
		heroID = actorID
	} else if role != user.RoleModerator {
		return ErrNotAllowed
	}
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.DeleteHeroFilm(ctx, heroID, filmID); err != nil {
			return err
		}
		return s.record(ctx, actorID, audit.ActionHeroFilmDeleted, entityType, filmID, ip,
			map[string]any{"heroId": heroID.String()})
	})
}

func (s *Service) DeleteVillainFilm(ctx context.Context, moderatorID, villainID, filmID uuid.UUID, ip netip.Addr) error {
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.DeleteVillainFilm(ctx, villainID, filmID); err != nil {
			return err
		}
		return s.record(ctx, moderatorID, audit.ActionVillainFilmDeleted, "villain_film", filmID, ip,
			map[string]any{"villainId": villainID.String()})
	})
}

func (s *Service) ListHeroFilms(ctx context.Context, filter HeroFilmFilter, limit, offset int) ([]HeroFilm, int64, error) {
	if filter.HeroID != nil {
		if err := s.requireApprovedHero(ctx, *filter.HeroID); err != nil {
			return nil, 0, err
		}
	}
	var fields validation.Collector
	if status := filter.Status; status != "" && status != "all" && status != StatusPending && status != StatusApproved && status != StatusRejected {
		fields.Add("status", validation.CodeInvalid)
	}
	title, code := validation.OptionalText(filter.Title, titleQueryMaxLength)
	fields.Add("q", code)
	if err := fields.Err(); err != nil {
		return nil, 0, err
	}
	filter.Title = title
	return s.Repo.ListHeroFilms(ctx, filter, limit, offset)
}

// ListPublicFilms renvoie la vidéothèque publique, un film par entrée. Un héros inconnu ou non
// validé donne une liste vide.
func (s *Service) ListPublicFilms(ctx context.Context, filter PublicFilmFilter, limit, offset int) ([]PublicFilm, int64, error) {
	title, code := validation.OptionalText(filter.Title, titleQueryMaxLength)
	if code != "" {
		return nil, 0, &validation.Error{Fields: map[string]string{"q": code}}
	}
	filter.Title = title
	return s.Repo.ListPublicFilms(ctx, filter, limit, offset)
}

// ListFilmHeroes renvoie les super-héros ayant au moins un film publié, par pseudo.
func (s *Service) ListFilmHeroes(ctx context.Context) ([]FilmHero, error) {
	return s.Repo.ListFilmHeroes(ctx)
}

func (s *Service) ListVillainFilms(ctx context.Context, villainID uuid.UUID, limit, offset int) ([]VillainFilm, int64, error) {
	if _, err := s.Villains.Find(ctx, villainID); err != nil {
		return nil, 0, err
	}
	return s.Repo.ListVillainFilms(ctx, villainID, limit, offset)
}

func (s *Service) resolveFilm(ctx context.Context, in FilmInput) (*Film, error) {
	in.IMDBID = strings.TrimSpace(in.IMDBID)
	if in.FilmID != uuid.Nil {
		return s.Repo.FindFilm(ctx, in.FilmID)
	}
	if in.IMDBID != "" {
		if !imdbIDPattern.MatchString(in.IMDBID) {
			return nil, &validation.Error{Fields: map[string]string{"imdbId": validation.CodeInvalid}}
		}
		found, err := s.Repo.FindFilmByIMDBID(ctx, in.IMDBID)
		if err == nil {
			return found, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		title, raw, err := s.Catalog.Get(ctx, in.IMDBID)
		if err != nil {
			return nil, err
		}
		if title == nil || title.ID != in.IMDBID {
			return nil, ErrTitleNotFound
		}
		return s.cacheTitle(ctx, *title, raw)
	}
	var fields validation.Collector
	title, code := validation.RequiredText(in.Title, 200)
	fields.Add("title", code)
	description, code := validation.OptionalText(in.Description, 4000)
	fields.Add("description", code)
	videoURL := strings.TrimSpace(in.VideoURL)
	parsed, parseErr := url.Parse(videoURL)
	if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(videoURL) > 2048 {
		fields.Add("videoUrl", validation.CodeInvalid)
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	film := &Film{ID: ids.New(), VideoURL: videoURL, Title: title, Plot: description, CreatedAt: s.Clock.Now()}
	if err := s.Repo.CreateFilm(ctx, film); err != nil {
		return nil, err
	}
	return film, nil
}

func (s *Service) cacheTitle(ctx context.Context, title Title, raw json.RawMessage) (*Film, error) {
	now := s.Clock.Now()
	film := &Film{
		ID: ids.New(), IMDBID: title.ID, Title: title.Title, ReleaseYear: title.ReleaseYear,
		PosterURL: title.PosterURL, Plot: title.Plot, Raw: string(raw), FetchedAt: &now, CreatedAt: now,
	}
	if err := s.Repo.CreateFilm(ctx, film); err != nil {
		if errors.Is(err, ErrIMDBExists) {
			return s.Repo.FindFilmByIMDBID(ctx, title.ID)
		}
		return nil, err
	}
	return film, nil
}

func (s *Service) requireApprovedHero(ctx context.Context, heroID uuid.UUID) error {
	profile, err := s.Heroes.Find(ctx, heroID)
	if err != nil {
		return err
	}
	if profile.Status != hero.StatusApproved {
		return hero.ErrNotFound
	}
	return nil
}

func (s *Service) record(ctx context.Context, actorID uuid.UUID, action, kind string, filmID uuid.UUID, ip netip.Addr, details map[string]any) error {
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: kind, EntityID: &filmID, IP: ip, Details: details,
	})
}
