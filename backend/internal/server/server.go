// Package server assemble l'application : dépendances, routes, chaîne de middlewares, tâches de
// fond et arrêt propre.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/account"
	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/avatar"
	"github.com/ashandcompany/heros-sos/backend/internal/blog"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/contact"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/film"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/herodashboard"
	"github.com/ashandcompany/heros-sos/backend/internal/herolocation"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/idempotency"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
	"github.com/ashandcompany/heros-sos/backend/internal/intervention"
	"github.com/ashandcompany/heros-sos/backend/internal/interventiontype"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/moderation"
	"github.com/ashandcompany/heros-sos/backend/internal/newsletter"
	"github.com/ashandcompany/heros-sos/backend/internal/ratelimit"
	"github.com/ashandcompany/heros-sos/backend/internal/realtime"
	"github.com/ashandcompany/heros-sos/backend/internal/reputation"
	"github.com/ashandcompany/heros-sos/backend/internal/signedlink"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

// components regroupe les dépendances partagées par les fonctions d'enregistrement des routes.
type components struct {
	cfg     config.Config
	logger  *slog.Logger
	db      *gorm.DB
	clock   clock.Clock
	tx      database.Transactor
	users   user.Repository
	audit   audit.Recorder
	events  *events.Store
	shipper *events.Shipper
	// idempotent rend une route d'écriture rejouable sans risque (voir le paquet idempotency).
	idempotent *idempotency.Middleware
	// hub, tailer et streamEvents diffusent le journal aux flux temps réel (voir le paquet realtime).
	hub          *realtime.Hub
	tailer       *realtime.Tailer
	streamEvents *realtime.Handler
	mailer       mail.Mailer

	health              healthHandler
	blogHandler         *blog.Handler
	filmHandler         *film.Handler
	authService         *auth.Service
	authMiddleware      *auth.Middleware
	authHandler         *auth.Handler
	heroHandler         *hero.Handler
	dashboardHandler    *herodashboard.Handler
	dashboardService    *herodashboard.Service
	locationService     *herolocation.Service
	locationHandler     *herolocation.Handler
	moderationHandler   *moderation.Handler
	incidentHandler     *incident.Handler
	interventionHandler *intervention.Handler
	reputationHandler   *reputation.Handler
	villainHandler      *villain.Handler
	typeHandler         *interventiontype.Handler

	accountService    *account.Service
	accountHandler    *account.Handler
	avatarHandler     *avatar.Handler
	contactHandler    *contact.Handler
	newsletterHandler *newsletter.Handler
	newsletterService *newsletter.Service
}

// Server est l'application prête à servir.
type Server struct {
	cfg        config.Config
	logger     *slog.Logger
	db         *gorm.DB
	router     *Router
	handler    http.Handler
	httpServer *http.Server
	mailQueue  mailQueue // nil quand un Mailer est injecté (tests)
	components *components
}

// mailQueue est la file d'envoi des mails, en mémoire (mail.Queue) ou dans RabbitMQ (mail.Publisher) : l'arrêt du
// serveur la vide ou la ferme.
type mailQueue interface {
	Close(ctx context.Context) error
}

type options struct {
	clock  clock.Clock
	mailer mail.Mailer
}

// Option personnalise l'assemblage, principalement pour les tests.
type Option func(*options)

// WithClock remplace l'horloge système.
func WithClock(clk clock.Clock) Option { return func(o *options) { o.clock = clk } }

// WithMailer remplace l'envoi de mails : le Mailer est alors appelé directement, sans file
// d'envoi asynchrone, ce qui rend les tests déterministes.
func WithMailer(mailer mail.Mailer) Option { return func(o *options) { o.mailer = mailer } }

// New ouvre la base, applique les migrations si la configuration le demande, puis assemble
// l'application.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger, opts ...Option) (*Server, error) {
	if cfg.AutoMigrate {
		if err := database.Migrate(cfg.Database.URL); err != nil {
			return nil, err
		}
	}
	db, err := database.Open(ctx, cfg.Database, logger)
	if err != nil {
		return nil, err
	}
	server, err := NewWithDB(cfg, logger, db, opts...)
	if err != nil {
		_ = database.Close(db)
		return nil, err
	}
	return server, nil
}

// NewWithDB assemble l'application sur une connexion existante.
func NewWithDB(cfg config.Config, logger *slog.Logger, db *gorm.DB, opts ...Option) (*Server, error) {
	settings := options{clock: clock.System{}}
	for _, opt := range opts {
		opt(&settings)
	}

	sealer, err := crypto.NewSealer(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		return nil, err
	}
	// Avant la file de mails : une erreur ici ne doit pas laisser de goroutines d'envoi derrière elle.
	links, err := signedlink.NewSigner(cfg.Crypto.Keys, cfg.Crypto.ActiveKeyID)
	if err != nil {
		return nil, err
	}
	avatars, err := avatar.NewDiskStore(cfg.Avatar.Dir)
	if err != nil {
		return nil, err
	}
	if cfg.IsProduction() && len(cfg.HTTP.TrustedProxies) == 0 {
		logger.Warn("aucun proxy de confiance configuré (HEROS_TRUSTED_PROXIES) : derrière un reverse proxy, " +
			"tous les clients partageraient l'adresse IP du proxy et les mêmes limites de débit")
	}

	mailer, queue, err := buildMailer(cfg, logger, settings)
	if err != nil {
		return nil, err
	}
	mailService, err := mail.NewService(mailer, cfg.Mail.FromName, cfg.Security.Tokens, mail.WithLogoURL(cfg.App.MailLogoURL()))
	if err != nil {
		return nil, err
	}

	c := &components{
		cfg: cfg, logger: logger, db: db, clock: settings.clock, mailer: mailer,
		tx:     database.NewTransactor(db),
		users:  user.NewRepository(db, sealer),
		audit:  audit.NewStore(db, settings.clock),
		events: events.NewStore(db, settings.clock),
		health: healthHandler{db: db, pingTimeout: cfg.Database.PingTimeout},
	}

	sessions := auth.NewSessionService(auth.NewSessionStore(db), c.clock, cfg.Session, cfg.Security.MFA, logger)
	cookies := auth.NewCookieManager(cfg.Session)
	csrf := auth.NewCSRF(cfg.Security.CSRFSecret)

	mfaStore := auth.NewMFAStore(db, sealer)
	hasher := auth.NewPasswordHasher(cfg.Security.Argon2)
	c.authService = auth.NewService(auth.ServiceDeps{
		Users: c.users, Tokens: auth.NewTokenStore(db), MFA: mfaStore, Sessions: sessions,
		Hasher: hasher, Mailer: mailService,
		Audit: c.audit, Tx: c.tx, Clock: c.clock, Security: cfg.Security, App: cfg.App, Logger: logger,
	})
	mfaService := auth.NewMFAService(auth.MFADeps{
		Store: mfaStore, Sessions: sessions, Users: c.users, Audit: c.audit, Tx: c.tx, Clock: c.clock,
		Security: cfg.Security,
	})
	c.authMiddleware = auth.NewMiddleware(sessions, cookies, csrf, cfg.HTTP.AllowedOrigins, logger)
	c.authHandler = auth.NewHandler(auth.HandlerDeps{
		Service: c.authService, Cookies: cookies, MFA: mfaService, CSRF: csrf, Clock: c.clock, Logger: logger,
		HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits,
	})
	heroProfiles := hero.NewRepository(db)
	c.heroHandler = hero.NewHandler(
		hero.NewService(heroProfiles, c.users, c.audit, c.tx, c.clock), logger, cfg.HTTP, APIPrefix+avatarRoutePrefix)
	c.moderationHandler = moderation.NewHandler(
		moderation.NewService(c.users, sessions, c.audit, c.tx, c.clock), moderation.NewSummaryStore(db), logger)

	types := interventiontype.NewRepository(db)
	c.blogHandler = blog.NewHandler(
		blog.NewService(blog.NewRepository(db), c.audit, c.tx, c.clock), logger, cfg.HTTP)
	villains := villain.NewRepository(db)
	c.filmHandler = film.NewHandler(film.NewService(film.ServiceDeps{
		Repo: film.NewRepository(db), Catalog: film.NewOMDbClient(cfg.OMDb.BaseURL, cfg.OMDb.APIKey, cfg.OMDb.Timeout),
		Heroes: heroProfiles, Villains: villains, Audit: c.audit, Tx: c.tx, Clock: c.clock,
	}), logger, cfg.HTTP)
	c.typeHandler = interventiontype.NewHandler(
		interventiontype.NewService(types, c.audit, c.tx, c.clock), logger, cfg.HTTP)
	c.villainHandler = villain.NewHandler(
		villain.NewService(villains, c.audit, c.tx, c.clock, types), logger, cfg.HTTP)
	c.locationService = herolocation.NewService(db, c.clock, cfg.Proximity.LocationTTL, cfg.Proximity.AlertRadiusKm)
	limits := cfg.Security.RateLimits
	c.locationHandler = herolocation.NewHandler(c.locationService, logger, cfg.HTTP,
		ratelimit.New(limits.HeroLocationPerUser, limits.MaxTrackedKeys, c.clock))
	incidentService := incident.NewService(incident.ServiceDeps{
		Repo: incident.NewRepository(db), Types: types, Villains: villains, Users: c.users, Heroes: c.locationService,
		Notifier: mailService, Audit: c.audit, Events: c.events, Tx: c.tx, Clock: c.clock, Logger: logger,
	})
	c.incidentHandler = incident.NewHandler(incident.HandlerDeps{
		Service: incidentService,
		Logger:  logger, HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits, Clock: c.clock,
	})
	c.dashboardService = herodashboard.NewService(
		herodashboard.NewRepository(db), incidentService,
		herodashboard.Options{StatsTTL: cfg.Dashboard.StatsTTL, Clock: c.clock, Logger: logger})
	c.dashboardHandler = herodashboard.NewHandler(c.dashboardService, logger)
	c.interventionHandler = intervention.NewHandler(intervention.NewService(intervention.ServiceDeps{
		Repo: intervention.NewRepository(db), Users: c.users, Heroes: heroProfiles,
		Audit: c.audit, Events: c.events, Tx: c.tx, Clock: c.clock, App: cfg.App, Notifier: mailService, Logger: logger,
	}), logger, cfg.HTTP)
	c.reputationHandler = reputation.NewHandler(reputation.NewService(reputation.ServiceDeps{
		Repo: reputation.NewRepository(db), Audit: c.audit, Tx: c.tx, Clock: c.clock,
	}), logger, cfg.HTTP)
	c.newsletterService = newsletter.NewService(newsletter.ServiceDeps{
		Repo: newsletter.NewRepository(db), Issues: newsletter.NewIssueRepository(db), Audit: c.audit, Tx: c.tx,
		Links: links, Mailer: mailService, App: cfg.App,
		ConfirmTTL: cfg.Security.Tokens.NewsletterConfirmTTL, Clock: c.clock, Logger: logger,
	})
	c.newsletterHandler = newsletter.NewHandler(newsletter.HandlerDeps{
		Service: c.newsletterService, Logger: logger, HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits, Clock: c.clock,
	})
	c.accountService = account.NewService(account.ServiceDeps{
		Users: c.users, Heroes: heroProfiles, Changes: account.NewEmailChangeRepository(db),
		Newsletter: c.newsletterService, Avatars: avatars, Mailer: mailService, Links: links, App: cfg.App,
		EmailChangeConfirmTTL: cfg.Security.Tokens.EmailChangeConfirmTTL,
		Privacy:               account.NewPrivacyRepository(db),
		Identity:              identityChecker{c.authService, mfaService}, Hasher: hasher,
		Audit: c.audit, Tx: c.tx, Clock: c.clock,
		AvatarMaxDimension: cfg.Avatar.MaxDimension, Logger: logger,
	})
	c.accountHandler = account.NewHandler(account.HandlerDeps{
		Service: c.accountService,
		Logger:  logger, HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits, Clock: c.clock,
		Avatar: cfg.Avatar, AvatarURLPrefix: APIPrefix + avatarRoutePrefix,
	})
	c.avatarHandler = avatar.NewHandler(avatars, logger)
	c.contactHandler = contact.NewHandler(contact.HandlerDeps{
		Service: contact.NewService(mailService, cfg.Mail.ContactRecipient),
		Logger:  logger, HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits, Clock: c.clock,
	})
	c.shipper = events.NewShipper(c.events, db, logger, cfg.Events)
	c.hub = realtime.NewHub(cfg.Events.StreamBuffer, cfg.Events.MaxStreamsPerUser)
	c.tailer = realtime.NewTailer(c.events, db, c.hub, logger, cfg.Events)
	c.streamEvents = realtime.NewHandler(c.hub, c.events, db, logger, cfg.Events, cfg.HTTP)
	c.idempotent = idempotency.New(idempotency.Deps{
		DB: db, Tx: c.tx, Clock: c.clock, Logger: logger, MaxBodyBytes: cfg.HTTP.MaxBodyBytes,
	})
	router := newRouter()
	registerRoutes(router, c)

	handler := httpx.Chain(router,
		httpx.RequestID,
		httpx.WithClientIP(cfg.HTTP.TrustedProxies),
		httpx.AccessLog(logger, probeRoutes...),
		httpx.Recover(logger),
		httpx.SecurityHeaders(cfg.IsProduction()),
		c.authMiddleware.OriginCheck,
		c.authMiddleware.Authenticate,
	)

	return &Server{
		cfg: cfg, logger: logger, db: db, router: router, handler: handler, mailQueue: queue, components: c,
		httpServer: &http.Server{
			Addr:              cfg.HTTP.Addr,
			Handler:           handler,
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
	}, nil
}

// buildMailer choisit l'expéditeur : celui injecté (tests), sinon une file d'envoi asynchrone selon la
// configuration. Avec la file en mémoire, l'API envoie elle-même (SMTP ou journal) ; avec RabbitMQ, elle ne
// fait que déposer les mails, et le worker `mailworker` les envoie.
func buildMailer(cfg config.Config, logger *slog.Logger, settings options) (mail.Mailer, mailQueue, error) {
	if settings.mailer != nil {
		return settings.mailer, nil, nil
	}

	if cfg.Mail.Queue == config.MailQueueRabbitMQ {
		publisher := mail.NewPublisher(cfg.Mail, logger)
		return publisher, publisher, nil
	}
	transport, err := mail.NewTransport(cfg.Mail, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("server : %w", err)
	}
	queue := mail.NewQueue(transport, cfg.Mail, logger)
	return queue, queue, nil
}

// Handler retourne le handler HTTP complet (middlewares inclus).
func (s *Server) Handler() http.Handler { return s.handler }

// DB retourne la connexion à la base (fermée par l'arrêt de Run).
func (s *Server) DB() *gorm.DB { return s.db }

// Routes retourne les routes enregistrées.
func (s *Server) Routes() []Route { return s.router.Routes() }

// Run sert l'API jusqu'à l'annulation du contexte, puis s'arrête proprement : plus aucune
// nouvelle requête, requêtes en cours terminées, mails en file envoyés, connexions fermées.
// Chaque goroutine appartient à l'errgroup : aucune ne survit à Run.
func (s *Server) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		s.logger.InfoContext(ctx, "serveur démarré", "addr", s.cfg.HTTP.Addr, "environment", s.cfg.Environment)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server : %w", err)
		}
		return nil
	})
	group.Go(func() error {
		s.runJanitor(groupCtx)
		return nil
	})
	group.Go(func() error {
		s.components.shipper.Run(groupCtx)
		return nil
	})
	group.Go(func() error {
		s.components.tailer.Run(groupCtx)
		return nil
	})
	if s.cfg.Dashboard.Warmup {
		group.Go(func() error {
			s.warmDashboard(groupCtx)
			return nil
		})
	}
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return s.shutdown(shutdownCtx)
	})
	return group.Wait()
}

// shutdown arrête les composants dans l'ordre : HTTP d'abord (plus de nouveaux mails), puis la
// file d'envoi (elle se vide), enfin la base.
func (s *Server) shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "arrêt du serveur")
	// Les flux temps réel d'abord : ce sont des requêtes qui ne se terminent pas, l'arrêt HTTP les attendrait.
	s.components.hub.Close()
	var errs []error
	if err := s.httpServer.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("arrêt HTTP : %w", err))
	}
	if s.mailQueue != nil {
		if err := s.mailQueue.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("vidage de la file de mails : %w", err))
		}
	}
	if err := database.Close(s.db); err != nil {
		errs = append(errs, fmt.Errorf("fermeture de la base : %w", err))
	}
	return errors.Join(errs...)
}

// warmDashboard calcule d'avance les statistiques du tableau de bord des héros, en arrière-plan : le serveur répond
// déjà pendant ce temps, et une requête qui arrive partage le calcul en cours.
func (s *Server) warmDashboard(ctx context.Context) {
	report, err := s.components.dashboardService.Warm(ctx)
	switch {
	case ctx.Err() != nil:
		return // arrêt du serveur
	case err != nil:
		s.logger.WarnContext(ctx, "tableau de bord : préchauffage échoué, les statistiques se calculeront à la première requête", "error", err)
	default:
		s.logger.InfoContext(ctx, "tableau de bord : statistiques préchauffées",
			"heroes", report.Heroes, "failed", report.Failed, "duration_ms", report.Elapsed.Milliseconds())
	}
}

// runJanitor exécute périodiquement les tâches de nettoyage jusqu'à l'annulation du contexte.
func (s *Server) runJanitor(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Jobs.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup(ctx)
		}
	}
}

// purgeTask est une purge périodique : ce qu'elle supprime, et les messages de son journal.
type purgeTask struct {
	run            func(ctx context.Context) (int64, error)
	failureMessage string
	successMessage string
	countAttribute string
}

// cleanup fait un passage de nettoyage, borné par HEROS_CLEANUP_TIMEOUT : une requête bloquée ne doit pas
// immobiliser la tâche. Un échec n'empêche pas les purges suivantes.
func (s *Server) cleanup(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Jobs.CleanupTimeout)
	defer cancel()

	c := s.components
	c.authHandler.SweepLimiters()
	c.incidentHandler.SweepLimiters()
	c.accountHandler.SweepLimiters()
	c.newsletterHandler.SweepLimiters()
	c.contactHandler.SweepLimiters()
	c.locationHandler.SweepLimiters()

	tokenRetention := s.cfg.Jobs.TokenRetention
	for _, task := range []purgeTask{
		// Une adresse jamais confirmée n'est pas conservée plus longtemps qu'un jeton périmé.
		{
			run:            func(ctx context.Context) (int64, error) { return c.newsletterService.PurgeStale(ctx, tokenRetention) },
			failureMessage: "purge des abonnements non confirmés en échec",
			successMessage: "abonnements non confirmés purgés", countAttribute: "abonnements_supprimes",
		},
		// Une commande trop ancienne ne sera plus rejouée : sa clé n'a plus lieu d'être gardée.
		{
			run: func(ctx context.Context) (int64, error) {
				return c.idempotent.PurgeOlderThan(ctx, s.cfg.Jobs.CommandRetention)
			},
			failureMessage: "purge des commandes rejouables en échec",
			successMessage: "commandes rejouables purgées", countAttribute: "commandes_supprimees",
		},
		// Une demande de changement d'adresse jamais confirmée ne peut plus aboutir une fois son lien expiré.
		{
			run: func(ctx context.Context) (int64, error) {
				return c.accountService.PurgeStaleEmailChanges(ctx, tokenRetention)
			},
			failureMessage: "purge des changements d'adresse non confirmés en échec",
			successMessage: "changements d'adresse non confirmés purgés", countAttribute: "demandes_supprimees",
		},
		// Une position périmée n'est plus utilisée pour l'alerte : on ne la garde pas.
		{
			run:            c.locationService.PurgeStale,
			failureMessage: "purge des positions de héros périmées en échec",
			successMessage: "positions de héros périmées purgées", countAttribute: "positions_supprimees",
		},
	} {
		s.purge(ctx, task)
	}

	sessions, tokens, err := c.authService.Cleanup(ctx, s.cfg.Jobs.SessionRetention, tokenRetention)
	if err != nil {
		s.logger.ErrorContext(ctx, "nettoyage périodique en échec", "error", err)
		return
	}
	if sessions > 0 || tokens > 0 {
		s.logger.InfoContext(ctx, "nettoyage périodique", "sessions_supprimees", sessions, "jetons_supprimes", tokens)
	}
}

func (s *Server) purge(ctx context.Context, task purgeTask) {
	purged, err := task.run(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, task.failureMessage, "error", err)
		return
	}
	if purged > 0 {
		s.logger.InfoContext(ctx, task.successMessage, task.countAttribute, purged)
	}
}

// identityChecker redemande le mot de passe (auth.Service) puis, si la 2FA est activée, un code
// (auth.MFAService) : c'est account.IdentityChecker, avant la suppression d'un compte.
type identityChecker struct {
	passwords *auth.Service
	codes     *auth.MFAService
}

func (c identityChecker) CheckPassword(ctx context.Context, userID uuid.UUID, password string) error {
	return c.passwords.CheckPassword(ctx, userID, password)
}

func (c identityChecker) CheckCodeIfEnrolled(ctx context.Context, userID uuid.UUID, code string, ip netip.Addr) error {
	return c.codes.CheckCodeIfEnrolled(ctx, userID, code, ip)
}
