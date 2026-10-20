// Package server assemble l'application : dépendances, routes, chaîne de middlewares, tâches de
// fond et arrêt propre.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/hero"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
	"github.com/ashandcompany/heros-sos/backend/internal/intervention"
	"github.com/ashandcompany/heros-sos/backend/internal/interventiontype"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/moderation"
	"github.com/ashandcompany/heros-sos/backend/internal/reputation"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/villain"
)

// components regroupe les dépendances partagées par les fonctions d'enregistrement des routes.
type components struct {
	cfg    config.Config
	logger *slog.Logger
	db     *gorm.DB
	clock  clock.Clock
	tx     database.Transactor
	users  user.Repository
	audit  audit.Recorder
	mailer mail.Mailer

	health              healthHandler
	authService         *auth.Service
	authMiddleware      *auth.Middleware
	authHandler         *auth.Handler
	heroHandler         *hero.Handler
	moderationHandler   *moderation.Handler
	incidentHandler     *incident.Handler
	interventionHandler *intervention.Handler
	reputationHandler   *reputation.Handler
	villainHandler      *villain.Handler
	typeHandler         *interventiontype.Handler
}

// Server est l'application prête à servir.
type Server struct {
	cfg        config.Config
	logger     *slog.Logger
	db         *gorm.DB
	router     *Router
	handler    http.Handler
	httpServer *http.Server
	mailQueue  *mail.Queue // nil quand un Mailer est injecté (tests)
	components *components
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
	if cfg.IsProduction() && len(cfg.HTTP.TrustedProxies) == 0 {
		logger.Warn("aucun proxy de confiance configuré (HEROS_TRUSTED_PROXIES) : derrière un reverse proxy, " +
			"tous les clients partageraient l'adresse IP du proxy et les mêmes limites de débit")
	}

	mailer, queue, err := buildMailer(cfg, logger, settings)
	if err != nil {
		return nil, err
	}
	mailService, err := mail.NewService(mailer, cfg.Mail.FromName, cfg.Security.Tokens)
	if err != nil {
		return nil, err
	}

	c := &components{
		cfg: cfg, logger: logger, db: db, clock: settings.clock, mailer: mailer,
		tx:     database.NewTransactor(db),
		users:  user.NewRepository(db, sealer),
		audit:  audit.NewStore(db, settings.clock),
		health: healthHandler{db: db, pingTimeout: cfg.Database.PingTimeout},
	}

	sessions := auth.NewSessionService(auth.NewSessionStore(db), c.clock, cfg.Session, logger)
	cookies := auth.NewCookieManager(cfg.Session)
	csrf := auth.NewCSRF(cfg.Security.CSRFSecret)

	c.authService = auth.NewService(auth.ServiceDeps{
		Users: c.users, Tokens: auth.NewTokenStore(db), Sessions: sessions,
		Hasher: auth.NewPasswordHasher(cfg.Security.Argon2), Mailer: mailService,
		Audit: c.audit, Tx: c.tx, Clock: c.clock, Security: cfg.Security, App: cfg.App, Logger: logger,
	})
	c.authMiddleware = auth.NewMiddleware(sessions, cookies, csrf, cfg.HTTP.AllowedOrigins, logger)
	c.authHandler = auth.NewHandler(auth.HandlerDeps{
		Service: c.authService, Cookies: cookies, CSRF: csrf, Clock: c.clock, Logger: logger,
		HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits,
	})
	heroProfiles := hero.NewRepository(db)
	c.heroHandler = hero.NewHandler(
		hero.NewService(heroProfiles, c.users, c.audit, c.tx, c.clock), logger, cfg.HTTP)
	c.moderationHandler = moderation.NewHandler(
		moderation.NewService(c.users, sessions, c.audit, c.tx, c.clock), logger)

	types := interventiontype.NewRepository(db)
	villains := villain.NewRepository(db)
	c.typeHandler = interventiontype.NewHandler(
		interventiontype.NewService(types, c.audit, c.tx, c.clock), logger, cfg.HTTP)
	c.villainHandler = villain.NewHandler(
		villain.NewService(villains, c.audit, c.tx, c.clock, types), logger, cfg.HTTP)
	c.incidentHandler = incident.NewHandler(incident.HandlerDeps{
		Service: incident.NewService(incident.ServiceDeps{
			Repo: incident.NewRepository(db), Types: types, Villains: villains, Users: c.users,
			Notifier: mailService, Audit: c.audit, Tx: c.tx, Clock: c.clock, Logger: logger,
		}),
		Logger: logger, HTTP: cfg.HTTP, Limits: cfg.Security.RateLimits, Clock: c.clock,
	})
	c.interventionHandler = intervention.NewHandler(intervention.NewService(intervention.ServiceDeps{
		Repo: intervention.NewRepository(db), Users: c.users, Heroes: heroProfiles,
		Audit: c.audit, Tx: c.tx, Clock: c.clock, App: cfg.App, Notifier: mailService, Logger: logger,
	}), logger, cfg.HTTP)
	c.reputationHandler = reputation.NewHandler(reputation.NewService(reputation.ServiceDeps{
		Repo: reputation.NewRepository(db), Audit: c.audit, Tx: c.tx, Clock: c.clock,
	}), logger, cfg.HTTP)
	router := newRouter()
	registerRoutes(router, c)

	handler := httpx.Chain(router,
		httpx.RequestID,
		httpx.WithClientIP(cfg.HTTP.TrustedProxies),
		httpx.AccessLog(logger),
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

// buildMailer choisit l'expéditeur : celui injecté (tests), sinon SMTP ou journal selon la
// configuration, placé derrière la file d'envoi asynchrone.
func buildMailer(cfg config.Config, logger *slog.Logger, settings options) (mail.Mailer, *mail.Queue, error) {
	if settings.mailer != nil {
		return settings.mailer, nil, nil
	}

	var transport mail.Mailer
	switch cfg.Mail.Driver {
	case config.MailDriverSMTP:
		transport = mail.NewSMTPMailer(cfg.Mail.SMTP, mail.Sender{Address: cfg.Mail.FromAddress, Name: cfg.Mail.FromName}, clock.System{})
	case config.MailDriverLog:
		transport = mail.NewLogMailer(logger)
	default:
		return nil, nil, fmt.Errorf("server : driver de mail inconnu %q", cfg.Mail.Driver)
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

func (s *Server) cleanup(ctx context.Context) {
	c := s.components
	c.authHandler.SweepLimiters()
	c.incidentHandler.SweepLimiters()
	sessions, tokens, err := c.authService.Cleanup(ctx, s.cfg.Jobs.SessionRetention, s.cfg.Jobs.TokenRetention)
	if err != nil {
		s.logger.ErrorContext(ctx, "nettoyage périodique en échec", "error", err)
		return
	}
	if sessions > 0 || tokens > 0 {
		s.logger.InfoContext(ctx, "nettoyage périodique", "sessions_supprimees", sessions, "jetons_supprimes", tokens)
	}
}
