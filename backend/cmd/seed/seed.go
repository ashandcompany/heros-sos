package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/film"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// dataTables sont vidées par -reset. schema_migrations n'en fait pas partie.
var dataTables = []string{
	"users", "hero_profiles", "sessions", "user_tokens", "audit_logs",
	"incident_types", "incidents", "interventions", "intervention_heroes",
	"villains", "villain_specialties", "incident_villains",
	"events", "event_publication", "processed_commands",
	"hero_reviews", "hero_review_replies",
	"films", "hero_films", "villain_films",
	"email_change_requests", "newsletter_subscriptions", "articles",
}

type seeder struct {
	db           *gorm.DB
	users        user.Repository
	passwordHash string
	catalog      film.Catalog // nil sans clé OMDb

	now time.Time
	rnd *rand.Rand

	adminID    uuid.UUID
	userIDs    map[string]uuid.UUID // clé de la personne → compte
	typeIDs    map[string]int16
	villainIDs map[string]uuid.UUID
	filmIDs    map[string]uuid.UUID
	posters    map[string]film.Title // titres enrichis par OMDb
	stats      stats
}

type stats struct {
	Moderators, Heroes, HeroRequests, Citizens, Types, Villains, Films int
	Incidents, Interventions, Reviews, Replies, Articles, Subscribers  int
	PostersFetched                                                     int
}

func (s *seeder) run(ctx context.Context, reset bool) (stats, error) {
	s.now = time.Now().UTC().Truncate(time.Second)
	// Graine fixe : deux exécutions produisent les mêmes notes et les mêmes horaires relatifs.
	s.rnd = rand.New(rand.NewPCG(2008, 2026))
	s.userIDs = map[string]uuid.UUID{}
	s.typeIDs = map[string]int16{}
	s.villainIDs = map[string]uuid.UUID{}
	s.filmIDs = map[string]uuid.UUID{}

	// Appels réseau hors transaction.
	s.fetchPosters(ctx)

	err := database.NewTransactor(s.db).WithinTx(ctx, func(ctx context.Context) error {
		if reset {
			if err := s.exec(ctx, "TRUNCATE TABLE "+strings.Join(dataTables, ", ")+" RESTART IDENTITY CASCADE"); err != nil {
				return fmt.Errorf("vidage des tables : %w", err)
			}
		} else if err := s.ensureEmpty(ctx); err != nil {
			return err
		}
		steps := []func(context.Context) error{
			s.seedModerators, s.seedIncidentTypes, s.seedHeroes, s.seedCitizens,
			s.seedFilms, s.seedVillains, s.seedIncidents, s.seedArticles, s.seedNewsletter,
		}
		for _, step := range steps {
			if err := step(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	return s.stats, err
}

func (s *seeder) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, s.db) }

func (s *seeder) exec(ctx context.Context, sql string, args ...any) error {
	return s.conn(ctx).Exec(sql, args...).Error
}

func (s *seeder) ensureEmpty(ctx context.Context) error {
	var filled bool
	err := s.conn(ctx).Raw(`SELECT EXISTS (SELECT 1 FROM users) OR EXISTS (SELECT 1 FROM villains)
		OR EXISTS (SELECT 1 FROM incidents) OR EXISTS (SELECT 1 FROM films)`).Scan(&filled).Error
	if err != nil {
		return fmt.Errorf("lecture de la base : %w", err)
	}
	if filled {
		return errors.New("la base contient déjà des données : relancez avec -reset pour TOUT effacer puis la remplir")
	}
	return nil
}

// daysAgo retourne un instant situé n jours avant maintenant, à une heure de la journée variable.
func (s *seeder) daysAgo(n int) time.Time {
	t := s.now.AddDate(0, 0, -n)
	if n == 0 {
		return t.Add(-time.Duration(s.rnd.IntN(240)+20) * time.Minute)
	}
	return t.Add(-time.Duration(s.rnd.IntN(12*60)) * time.Minute)
}

func (s *seeder) minutes(min, max int) time.Duration {
	return time.Duration(min+s.rnd.IntN(max-min+1)) * time.Minute
}

func email(key string) string { return key + "@" + emailDomain }

// --- Comptes -----------------------------------------------------------------------------------

func (s *seeder) createUser(ctx context.Context, p person, role user.Role, status user.Status, createdAt time.Time) (uuid.UUID, error) {
	birthDate, err := validation.ParseBirthDate(p.BirthDate)
	if err != nil {
		return uuid.Nil, fmt.Errorf("date de naissance de %s : %w", p.Key, err)
	}
	u := &user.User{
		ID: ids.New(), Email: email(p.Key), PasswordHash: s.passwordHash, Role: role, Status: status,
		Civility: p.Civility, FirstName: p.FirstName, LastName: p.LastName, BirthDate: birthDate, Phone: p.Phone,
		Address: user.Address{Street: p.Address.Street, PostalCode: p.Address.PostalCode, City: p.Address.City, Country: p.Address.Country},
	}
	activated := createdAt.Add(s.minutes(5, 120))
	switch status {
	case user.StatusActive:
		u.ActivatedAt = &activated
	case user.StatusDeactivated:
		deactivated := activated.AddDate(0, 0, 10)
		u.ActivatedAt, u.DeactivatedAt = &activated, &deactivated
	}
	if err := s.users.Create(ctx, u); err != nil {
		return uuid.Nil, fmt.Errorf("création du compte %s : %w", u.Email, err)
	}
	if err := s.exec(ctx, "UPDATE users SET created_at = ?, updated_at = ? WHERE id = ?", createdAt, createdAt, u.ID); err != nil {
		return uuid.Nil, err
	}
	s.userIDs[p.Key] = u.ID
	return u.ID, nil
}

func (s *seeder) seedModerators(ctx context.Context) error {
	for i, m := range moderators {
		id, err := s.createUser(ctx, m.person, user.RoleModerator, user.StatusActive, s.now.AddDate(0, 0, -90+i))
		if err != nil {
			return err
		}
		if i == 0 {
			s.adminID = id
		}
		s.stats.Moderators++
	}
	return nil
}

func (s *seeder) seedHeroes(ctx context.Context) error {
	for i, h := range heroes {
		createdAt := s.now.AddDate(0, 0, -85+i)
		role := user.RoleCitizen // une demande en attente ou refusée laisse le compte citoyen
		if h.Status == "approved" {
			role = user.RoleHero
		}
		id, err := s.createUser(ctx, h.person, role, user.StatusActive, createdAt)
		if err != nil {
			return err
		}

		requested := createdAt.Add(s.minutes(60, 600))
		var reviewedBy *uuid.UUID
		var reviewedAt *time.Time
		var rejection *string
		if h.Status != "pending" {
			at := requested.Add(s.minutes(60, 48*60))
			reviewedBy, reviewedAt = &s.adminID, &at
		}
		if h.Status == "rejected" {
			rejection = &h.Rejection
		}
		message := h.RequestMessage
		if message == "" {
			message = "Je souhaite mettre mes capacités au service de Héros SOS."
		}
		if err := s.exec(ctx, `INSERT INTO hero_profiles
			(user_id, alias, bio, request_message, verification_status, reviewed_by, reviewed_at, rejection_reason, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, h.Alias, h.Bio, message, h.Status, reviewedBy, reviewedAt, rejection, requested); err != nil {
			return fmt.Errorf("profil de %s : %w", h.Alias, err)
		}
		if h.Status == "approved" {
			if err := s.exec(ctx, "UPDATE users SET verified_at = ?, verified_by = ? WHERE id = ?", reviewedAt, s.adminID, id); err != nil {
				return err
			}
			s.stats.Heroes++
		} else {
			s.stats.HeroRequests++
		}
	}
	return nil
}

func (s *seeder) seedCitizens(ctx context.Context) error {
	for i, c := range citizens {
		status := user.StatusActive
		if c.Status != "" {
			status = user.Status(c.Status)
		}
		if _, err := s.createUser(ctx, c.person, user.RoleCitizen, status, s.now.AddDate(0, 0, -75+i*3)); err != nil {
			return err
		}
		s.stats.Citizens++
	}
	return nil
}

// --- Catalogues --------------------------------------------------------------------------------

func (s *seeder) seedIncidentTypes(ctx context.Context) error {
	for _, t := range incidentTypes {
		if err := s.exec(ctx, `INSERT INTO incident_types (code, label, endangers_life) VALUES (?, ?, ?)
			ON CONFLICT (code) DO NOTHING`, t.Code, t.Label, t.EndangersLife); err != nil {
			return fmt.Errorf("type %s : %w", t.Code, err)
		}
	}
	var rows []struct {
		ID   int16
		Code string
	}
	if err := s.conn(ctx).Raw("SELECT id, code FROM incident_types").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		s.typeIDs[row.Code] = row.ID
	}
	s.stats.Types = len(rows)
	return nil
}

// fetchPosters complète les films avec l'affiche et le résumé d'OMDb, si une clé est configurée.
// Un échec n'arrête pas le seed : le film garde les textes du seed, sans affiche.
func (s *seeder) fetchPosters(ctx context.Context) {
	s.posters = map[string]film.Title{}
	if s.catalog == nil {
		return
	}
	for _, f := range films {
		title, _, err := s.catalog.Get(ctx, f.IMDbID)
		if err != nil {
			continue
		}
		s.posters[f.IMDbID] = *title
		s.stats.PostersFetched++
	}
}

func (s *seeder) seedFilms(ctx context.Context) error {
	createdAt := s.now.AddDate(0, 0, -80)
	for _, f := range films {
		id := ids.New()
		var poster *string
		var fetchedAt *time.Time
		if omdb, ok := s.posters[f.IMDbID]; ok && omdb.PosterURL != "" {
			poster, fetchedAt = &omdb.PosterURL, &createdAt
		}
		if err := s.exec(ctx, `INSERT INTO films (id, imdb_id, title, release_year, poster_url, plot, fetched_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, f.IMDbID, f.Title, f.Year, poster, f.Plot, fetchedAt, createdAt); err != nil {
			return fmt.Errorf("film %s : %w", f.Title, err)
		}
		s.filmIDs[f.IMDbID] = id
		s.stats.Films++
	}

	for _, h := range heroes {
		heroID := s.userIDs[h.Key]
		for _, imdbID := range h.Films {
			at := s.now.AddDate(0, 0, -70+s.rnd.IntN(30))
			reviewed := at.Add(s.minutes(30, 24*60))
			if err := s.linkHeroFilm(ctx, heroID, imdbID, "approved", &s.adminID, &reviewed, at); err != nil {
				return err
			}
		}
		for _, imdbID := range h.PendingFilms {
			if err := s.linkHeroFilm(ctx, heroID, imdbID, "pending", nil, nil, s.daysAgo(1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) linkHeroFilm(ctx context.Context, heroID uuid.UUID, imdbID, status string, reviewedBy *uuid.UUID, reviewedAt *time.Time, createdAt time.Time) error {
	filmID, ok := s.filmIDs[imdbID]
	if !ok {
		return fmt.Errorf("film inconnu : %s", imdbID)
	}
	return s.exec(ctx, `INSERT INTO hero_films (hero_id, film_id, status, reviewed_by, reviewed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, heroID, filmID, status, reviewedBy, reviewedAt, createdAt)
}

func (s *seeder) seedVillains(ctx context.Context) error {
	for i, v := range villains {
		id := ids.New()
		createdAt := s.now.AddDate(0, 0, -88+i)
		if err := s.exec(ctx, `INSERT INTO villains
			(id, alias, last_name, first_name, main_sector, description, threat_level, status, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, v.Alias, nullable(v.LastName), nullable(v.FirstName), v.Sector, v.Description, v.Threat, v.Status,
			s.adminID, createdAt, createdAt); err != nil {
			return fmt.Errorf("vilain %s : %w", v.Alias, err)
		}
		for _, code := range v.Specialties {
			typeID, ok := s.typeIDs[code]
			if !ok {
				return fmt.Errorf("vilain %s : type inconnu %s", v.Alias, code)
			}
			if err := s.exec(ctx, "INSERT INTO villain_specialties (villain_id, type_id) VALUES (?, ?)", id, typeID); err != nil {
				return err
			}
		}
		for _, imdbID := range v.Films {
			filmID, ok := s.filmIDs[imdbID]
			if !ok {
				return fmt.Errorf("vilain %s : film inconnu %s", v.Alias, imdbID)
			}
			if err := s.exec(ctx, `INSERT INTO villain_films (villain_id, film_id, created_by, created_at)
				VALUES (?, ?, ?, ?)`, id, filmID, s.adminID, createdAt); err != nil {
				return err
			}
		}
		s.villainIDs[v.Alias] = id
		s.stats.Villains++
	}
	return nil
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// --- Incidents et interventions ----------------------------------------------------------------

func (s *seeder) seedIncidents(ctx context.Context) error {
	for _, in := range incidents {
		if err := s.seedIncident(ctx, in); err != nil {
			return fmt.Errorf("incident « %s » : %w", in.Title, err)
		}
		s.stats.Incidents++
	}
	return nil
}

func (s *seeder) seedIncident(ctx context.Context, in incidentSeed) error {
	reporterID, ok := s.userIDs[in.Reporter]
	if !ok {
		return fmt.Errorf("citoyen inconnu : %s", in.Reporter)
	}
	typeID, ok := s.typeIDs[in.Type]
	if !ok {
		return fmt.Errorf("type inconnu : %s", in.Type)
	}
	createdAt := s.daysAgo(in.DaysAgo)
	occurredAt := createdAt.Add(-s.minutes(3, 40))
	status := in.Outcome
	if status == outcomeProposed {
		status = outcomeInProgress
	}

	incidentID := ids.New()
	// Les incidents de démonstration sont approuvés : sans cela (défaut « pending »), les héros ne verraient rien.
	if err := s.exec(ctx, `INSERT INTO incidents
		(id, reporter_id, type_id, title, description, status, priority, moderation, citizens_rescued,
		 street, postal_code, city, country, latitude, longitude, occurred_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'approved', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		incidentID, reporterID, typeID, in.Title, in.Description, status, in.Priority, in.Rescued,
		in.Address.Street, in.Address.PostalCode, in.Address.City, in.Address.Country, in.Lat, in.Lon,
		occurredAt, createdAt, createdAt); err != nil {
		return err
	}
	for _, alias := range in.Villains {
		villainID, ok := s.villainIDs[alias]
		if !ok {
			return fmt.Errorf("vilain inconnu : %s", alias)
		}
		if err := s.exec(ctx, "INSERT INTO incident_villains (incident_id, villain_id, arrested) VALUES (?, ?, ?)",
			incidentID, villainID, in.Outcome == outcomeSucceeded); err != nil {
			return err
		}
	}
	if in.Outcome == outcomeReported {
		return nil
	}
	return s.seedIntervention(ctx, in, incidentID, reporterID, createdAt)
}

func (s *seeder) seedIntervention(ctx context.Context, in incidentSeed, incidentID, reporterID uuid.UUID, reportedAt time.Time) error {
	if len(in.Heroes) == 0 {
		return errors.New("une intervention demande au moins un héros")
	}
	heroIDs := make([]uuid.UUID, len(in.Heroes))
	for i, key := range in.Heroes {
		id, ok := s.userIDs[key]
		if !ok {
			return fmt.Errorf("héros inconnu : %s", key)
		}
		heroIDs[i] = id
	}
	leadID := heroIDs[0]
	startedAt := reportedAt.Add(s.minutes(4, 30))
	if startedAt.After(s.now) {
		startedAt = s.now.Add(-time.Minute)
	}

	closed := in.Outcome == outcomeSucceeded || in.Outcome == outcomeFailed
	interventionStatus := outcomeInProgress
	var endedAt *time.Time
	var proposedStatus *string
	var proposedBy *uuid.UUID
	if closed {
		interventionStatus = in.Outcome
		end := startedAt.Add(s.minutes(45, 6*60))
		endedAt = &end
	}
	if in.Outcome == outcomeProposed {
		succeeded := outcomeSucceeded
		proposedStatus, proposedBy = &succeeded, &leadID
	}

	interventionID := ids.New()
	if err := s.exec(ctx, `INSERT INTO interventions
		(id, incident_id, status, proposed_status, proposed_by, notes, started_at, ended_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		interventionID, incidentID, interventionStatus, proposedStatus, proposedBy, nullable(in.Notes), startedAt, endedAt); err != nil {
		return err
	}

	for i, heroID := range heroIDs {
		role, invitedBy, invitedAt := "lead", (*uuid.UUID)(nil), (*time.Time)(nil)
		joinedAt := startedAt
		if i > 0 {
			invited := startedAt.Add(time.Duration(i) * time.Minute)
			role, invitedBy, invitedAt = "member", &leadID, &invited
			joinedAt = invited.Add(s.minutes(1, 8))
		}
		outcomeValidation := "pending"
		if closed || (in.Outcome == outcomeProposed && i == 0) {
			outcomeValidation = "approved"
		}
		if err := s.exec(ctx, `INSERT INTO intervention_heroes
			(intervention_id, hero_id, role, participation, invited_by, outcome_validation, is_active, invited_at, joined_at)
			VALUES (?, ?, ?, 'confirmed', ?, ?, ?, ?, ?)`,
			interventionID, heroID, role, invitedBy, outcomeValidation, !closed, invitedAt, joinedAt); err != nil {
			return fmt.Errorf("participation de %s : %w", in.Heroes[i], err)
		}
	}
	for i, key := range in.Invited {
		heroID, ok := s.userIDs[key]
		if !ok {
			return fmt.Errorf("héros inconnu : %s", key)
		}
		invited := startedAt.Add(time.Duration(len(heroIDs)+i) * time.Minute)
		if err := s.exec(ctx, `INSERT INTO intervention_heroes
			(intervention_id, hero_id, role, participation, invited_by, outcome_validation, is_active, invited_at)
			VALUES (?, ?, 'member', 'invited', ?, 'pending', FALSE, ?)`,
			interventionID, heroID, leadID, invited); err != nil {
			return fmt.Errorf("invitation de %s : %w", key, err)
		}
	}
	s.stats.Interventions++

	if closed {
		if err := s.exec(ctx, "UPDATE incidents SET updated_at = ? WHERE id = ?", *endedAt, incidentID); err != nil {
			return err
		}
		return s.seedReviews(ctx, in, interventionID, reporterID, heroIDs, *endedAt)
	}
	return nil
}

// seedReviews fait noter les héros d'une intervention clôturée par le citoyen qui l'a signalée.
func (s *seeder) seedReviews(ctx context.Context, in incidentSeed, interventionID, reporterID uuid.UUID, heroIDs []uuid.UUID, endedAt time.Time) error {
	for i, heroID := range heroIDs {
		// Les citoyens ne notent pas toujours tout le monde.
		if i > 0 && s.rnd.IntN(4) == 0 {
			continue
		}
		createdAt := endedAt.Add(s.minutes(30, 36*60))
		if createdAt.After(s.now) {
			continue
		}
		rating, comment := s.pickReview(in.Outcome == outcomeSucceeded)
		status := "published"
		var moderatedBy *uuid.UUID
		var moderatedAt *time.Time
		// Un avis masqué par la modération, pour remplir la file correspondante.
		if in.Heroes[i] == "logan" && in.Outcome == outcomeFailed {
			rating, comment, status = 1, "Il a cassé tout le bar et insulté le serveur. Zéro étoile, et encore.", "hidden"
			at := createdAt.Add(3 * time.Hour)
			moderatedBy, moderatedAt = &s.adminID, &at
		}
		reviewID := ids.New()
		if err := s.exec(ctx, `INSERT INTO hero_reviews
			(id, intervention_id, hero_id, citizen_id, rating, comment, status, moderated_by, moderated_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			reviewID, interventionID, heroID, reporterID, rating, nullable(comment), status, moderatedBy, moderatedAt, createdAt); err != nil {
			return fmt.Errorf("avis : %w", err)
		}
		s.stats.Reviews++

		if status == "published" && s.rnd.IntN(3) == 0 {
			replyAt := createdAt.Add(s.minutes(60, 24*60))
			if replyAt.After(s.now) {
				continue
			}
			if err := s.exec(ctx, `INSERT INTO hero_review_replies (review_id, hero_id, content, status, created_at)
				VALUES (?, ?, ?, 'published', ?)`, reviewID, heroID, heroReplies[s.rnd.IntN(len(heroReplies))], replyAt); err != nil {
				return fmt.Errorf("réponse à un avis : %w", err)
			}
			s.stats.Replies++
		}
	}
	return nil
}

func (s *seeder) pickReview(succeeded bool) (int16, string) {
	if succeeded {
		return int16(4 + s.rnd.IntN(2)), successComments[s.rnd.IntN(len(successComments))]
	}
	return int16(2 + s.rnd.IntN(2)), failureComments[s.rnd.IntN(len(failureComments))]
}

// --- Contenus ----------------------------------------------------------------------------------

func (s *seeder) seedArticles(ctx context.Context) error {
	for _, a := range articles {
		createdAt := s.daysAgo(a.DaysAgo)
		status := "brouillon"
		var publishedAt *time.Time
		if a.Published {
			status, publishedAt = "publie", &createdAt
		}
		if err := s.exec(ctx, `INSERT INTO articles
			(id, title, slug, excerpt, content, status, author_id, author_name, created_at, updated_at, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ids.New(), a.Title, a.Slug, a.Excerpt, a.Content, status, s.adminID,
			moderators[0].FirstName+" "+moderators[0].LastName, createdAt, createdAt, publishedAt); err != nil {
			return fmt.Errorf("article %s : %w", a.Slug, err)
		}
		s.stats.Articles++
	}
	return nil
}

func (s *seeder) seedNewsletter(ctx context.Context) error {
	insert := func(address string, userID *uuid.UUID, source string, daysAgo int) error {
		at := s.daysAgo(daysAgo)
		confirmed := at.Add(s.minutes(2, 90))
		if err := s.exec(ctx, `INSERT INTO newsletter_subscriptions (id, email, user_id, status, source, confirmed_at, created_at)
			VALUES (?, ?, ?, 'subscribed', ?, ?, ?)`, ids.New(), address, userID, source, confirmed, at); err != nil {
			return fmt.Errorf("abonnement %s : %w", address, err)
		}
		s.stats.Subscribers++
		return nil
	}
	for i, c := range citizens {
		if !c.Newsletter {
			continue
		}
		id := s.userIDs[c.Key]
		if err := insert(email(c.Key), &id, "account", 60-i*4); err != nil {
			return err
		}
	}
	for i, local := range []string{"daily.bugle", "kate.bishop", "yelena.belova", "america.chavez"} {
		if err := insert(email(local), nil, "footer", 40-i*7); err != nil {
			return err
		}
	}
	return nil
}

// --- Résumé ------------------------------------------------------------------------------------

func (st stats) print(w io.Writer, password string) {
	fmt.Fprintf(w, `Base remplie :
  %d modérateurs, %d super-héros validés, %d demandes de héros (en attente ou refusées), %d citoyens
  %d types d'incident, %d vilains, %d films (%d affiches OMDb)
  %d incidents, %d interventions, %d avis, %d réponses de héros
  %d articles, %d abonnés à la newsletter

Comptes (mot de passe commun : %s) :
  modérateur  %s
  héros       %s, %s…
  citoyen     %s, %s…
`,
		st.Moderators, st.Heroes, st.HeroRequests, st.Citizens,
		st.Types, st.Villains, st.Films, st.PostersFetched,
		st.Incidents, st.Interventions, st.Reviews, st.Replies,
		st.Articles, st.Subscribers,
		password,
		email(moderators[0].Key),
		email(heroes[0].Key), email(heroes[6].Key),
		email(citizens[0].Key), email(citizens[1].Key),
	)
}
