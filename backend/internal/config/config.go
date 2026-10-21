// Package config centralise tous les paramètres du backend : lecture des variables
// d'environnement, valeurs par défaut et validation. Aucun autre paquet ne lit l'environnement.
package config

import (
	"errors"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"time"
)

// Environment désigne le contexte d'exécution. Les règles de validation sont plus strictes
// en production.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
)

// MailDriver choisit comment les mails sont envoyés.
type MailDriver string

const (
	MailDriverLog  MailDriver = "log"  // écrit les mails dans les logs (développement uniquement)
	MailDriverSMTP MailDriver = "smtp" // envoie via un serveur SMTP
)

// TLSMode décrit le chiffrement de la connexion SMTP.
type TLSMode string

const (
	TLSModeNone     TLSMode = "none"     // aucun chiffrement (Mailpit en développement)
	TLSModeStartTLS TLSMode = "starttls" // connexion en clair puis passage en TLS
	TLSModeImplicit TLSMode = "tls"      // TLS dès l'ouverture de la connexion
)

const (
	secureCookieName = "__Host-session"
	plainCookieName  = "session"
)

// RateRule limite à Max requêtes par fenêtre de durée Window.
type RateRule struct {
	Max    int
	Window time.Duration
}

// Config regroupe l'ensemble des paramètres. Elle est construite une fois au démarrage puis
// reste en lecture seule : elle peut être partagée sans verrou entre goroutines.
type Config struct {
	Environment Environment
	LogLevel    slog.Level
	AutoMigrate bool
	HTTP        HTTP
	Database    Database
	Session     Session
	Security    Security
	Crypto      Crypto
	Mail        Mail
	OMDb        OMDb
	App         App
	Jobs        Jobs
}

// OMDb regroupe les réglages du catalogue de films.
type OMDb struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// HTTP regroupe les réglages du serveur.
type HTTP struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	// HealthcheckTimeout borne le délai de `api -healthcheck` (contrôle de santé du conteneur).
	HealthcheckTimeout time.Duration
	// TrustedProxies liste les proxys dont l'en-tête X-Forwarded-For est cru pour
	// déterminer l'adresse IP du client.
	TrustedProxies []netip.Prefix
	// AllowedOrigins liste les origines autorisées à émettre des requêtes qui modifient
	// des données (protection CSRF), par exemple https://heros-sos.example.
	AllowedOrigins []string
}

// Database regroupe les réglages de la connexion PostgreSQL.
type Database struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
	// SlowQueryThreshold : au-delà, une requête est journalisée comme lente (sans son SQL,
	// qui peut contenir des données personnelles).
	SlowQueryThreshold time.Duration
}

// SessionPolicy fixe les durées d'une session : expiration après inactivité et durée maximale.
type SessionPolicy struct {
	IdleTimeout time.Duration
	MaxLifetime time.Duration
}

// Session regroupe les réglages des sessions et du cookie.
type Session struct {
	CookieSecure bool
	// Member s'applique aux citoyens et aux héros, RememberMe à « se souvenir de moi »,
	// Moderator aux modérateurs (jamais de « se souvenir de moi »).
	Member     SessionPolicy
	RememberMe SessionPolicy
	Moderator  SessionPolicy
	// TouchPeriod limite la fréquence de mise à jour de last_seen_at (une écriture au plus
	// par période et par session).
	TouchPeriod time.Duration
}

// CookieName retourne le nom du cookie de session : le préfixe __Host- impose au navigateur
// Secure, Path=/ et l'absence de Domain, mais exige HTTPS.
func (s Session) CookieName() string {
	if s.CookieSecure {
		return secureCookieName
	}
	return plainCookieName
}

// Argon2 regroupe les paramètres de hachage des mots de passe (argon2id).
type Argon2 struct {
	Time          uint32
	MemoryKiB     uint32
	Threads       uint8
	KeyLength     uint32
	SaltLength    uint32
	MaxConcurrent int // nombre maximal de hachages simultanés (chacun consomme MemoryKiB)
}

// PasswordPolicy fixe les longueurs acceptées pour un mot de passe.
type PasswordPolicy struct {
	MinLength int
	MaxLength int
}

// Lockout décrit le verrouillage progressif d'un compte après des échecs de connexion.
type Lockout struct {
	MaxFailures int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// RateLimits regroupe les règles de limitation de débit.
type RateLimits struct {
	LoginPerIP            RateRule
	LoginPerIdentifier    RateRule
	RegisterPerIP         RateRule
	PasswordResetPerIP    RateRule
	PasswordResetPerEmail RateRule
	IncidentCreatePerUser RateRule
	MaxTrackedKeys        int
}

// Tokens regroupe la durée de vie des jetons à usage unique envoyés par mail.
type Tokens struct {
	ActivationTTL    time.Duration
	PasswordResetTTL time.Duration
}

// Security regroupe les réglages de sécurité.
type Security struct {
	Argon2     Argon2
	Password   PasswordPolicy
	Lockout    Lockout
	RateLimits RateLimits
	Tokens     Tokens
	CSRFSecret []byte
}

// Crypto regroupe les clés maîtresses du chiffrement des données personnelles.
// Plusieurs versions coexistent pour permettre la rotation : ActiveKeyID chiffre,
// toutes les clés déchiffrent.
type Crypto struct {
	Keys        map[uint16][]byte
	ActiveKeyID uint16
}

// SMTP regroupe les réglages du serveur d'envoi.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	TLSMode  TLSMode
	Timeout  time.Duration
}

// Mail regroupe les réglages d'envoi des mails et de la file d'envoi asynchrone.
type Mail struct {
	Driver      MailDriver
	FromAddress string
	FromName    string
	SMTP        SMTP
	Workers     int
	QueueSize   int
	MaxAttempts int
	RetryDelay  time.Duration
	SendTimeout time.Duration
}

// App regroupe les réglages de l'application visibles des utilisateurs.
type App struct {
	// PublicBaseURL est l'adresse publique du frontend, utilisée pour les liens des mails.
	PublicBaseURL     *url.URL
	ActivationPath    string
	PasswordResetPath string
}

// ActivationLink construit le lien d'activation de compte envoyé par mail.
func (a App) ActivationLink(token string) string {
	return a.link(a.ActivationPath, token)
}

// PasswordResetLink construit le lien de réinitialisation de mot de passe envoyé par mail.
func (a App) PasswordResetLink(token string) string {
	return a.link(a.PasswordResetPath, token)
}

// HeroReviewLink construit le lien de notation d'un héros après une intervention.
func (a App) HeroReviewLink(interventionID, heroID string) string {
	target := *a.PublicBaseURL
	target.Path = "/profil/interventions"
	target.RawQuery = url.Values{"interventionId": {interventionID}, "heroId": {heroID}}.Encode()
	return target.String()
}

func (a App) link(path, token string) string {
	target := *a.PublicBaseURL
	target.Path = path
	target.RawQuery = url.Values{"token": {token}}.Encode()
	return target.String()
}

// Jobs regroupe les réglages des tâches de fond.
type Jobs struct {
	CleanupInterval  time.Duration
	SessionRetention time.Duration
	TokenRetention   time.Duration
}

// IsProduction indique si l'application tourne en production.
func (c Config) IsProduction() bool {
	return c.Environment == EnvironmentProduction
}

// FromEnv construit la configuration depuis les variables d'environnement du processus.
func FromEnv() (Config, error) {
	return Load(os.Getenv)
}

// Load construit la configuration à partir de getenv (os.Getenv en pratique, une fonction
// de test sinon). Toutes les erreurs de lecture et de validation sont retournées ensemble.
func Load(getenv func(string) string) (Config, error) {
	l := &loader{getenv: getenv}
	environment := l.environment()
	production := environment == EnvironmentProduction

	cryptoKeys := l.cryptoKeys()
	cfg := Config{
		Environment: environment,
		LogLevel:    l.logLevel(),
		AutoMigrate: l.boolean(envAutoMigrate, false),
		HTTP: HTTP{
			Addr:               l.text(envHTTPAddr, defaultHTTPAddr),
			ReadHeaderTimeout:  l.duration(envHTTPReadHeaderTimeout, defaultReadHeaderTimeout),
			ReadTimeout:        l.duration(envHTTPReadTimeout, defaultReadTimeout),
			WriteTimeout:       l.duration(envHTTPWriteTimeout, defaultWriteTimeout),
			IdleTimeout:        l.duration(envHTTPIdleTimeout, defaultIdleTimeout),
			ShutdownTimeout:    l.duration(envHTTPShutdownTimeout, defaultShutdownTimeout),
			MaxBodyBytes:       int64(l.positiveInt(envHTTPMaxBodyBytes, defaultMaxBodyBytes)),
			HealthcheckTimeout: l.duration(envHealthcheckTimeout, defaultHealthcheckTimeout),
			TrustedProxies:     l.trustedProxies(),
			AllowedOrigins:     l.list(envAllowedOrigins),
		},
		Database: Database{
			URL:                l.text(envDatabaseURL, ""),
			MaxOpenConns:       l.positiveInt(envDatabaseMaxOpenConns, defaultDBMaxOpenConns),
			MaxIdleConns:       l.positiveInt(envDatabaseMaxIdleConns, defaultDBMaxIdleConns),
			ConnMaxLifetime:    l.duration(envDatabaseConnMaxLifetime, defaultDBConnMaxLifetime),
			ConnMaxIdleTime:    l.duration(envDatabaseConnMaxIdleTime, defaultDBConnMaxIdleTime),
			PingTimeout:        l.duration(envDatabasePingTimeout, defaultDBPingTimeout),
			SlowQueryThreshold: l.duration(envDatabaseSlowQuery, defaultDBSlowQuery),
		},
		Session: Session{
			CookieSecure: l.boolean(envCookieSecure, production),
			Member: SessionPolicy{
				IdleTimeout: l.duration(envSessionMemberIdle, defaultMemberSessionIdle),
				MaxLifetime: l.duration(envSessionMemberMax, defaultMemberSessionMax),
			},
			RememberMe: SessionPolicy{
				IdleTimeout: l.duration(envSessionRememberIdle, defaultRememberMeIdle),
				MaxLifetime: l.duration(envSessionRememberMax, defaultRememberMeMax),
			},
			Moderator: SessionPolicy{
				IdleTimeout: l.duration(envSessionModeratorIdle, defaultModeratorIdle),
				MaxLifetime: l.duration(envSessionModeratorMax, defaultModeratorMax),
			},
			TouchPeriod: l.duration(envSessionTouchPeriod, defaultSessionTouchPeriod),
		},
		Security: Security{
			Argon2: Argon2{
				Time:          uint32(l.positiveInt(envArgon2Time, defaultArgon2Time)),
				MemoryKiB:     uint32(l.positiveInt(envArgon2MemoryKiB, defaultArgon2MemoryKiB)),
				Threads:       uint8(l.positiveInt(envArgon2Threads, defaultArgon2Threads)),
				KeyLength:     defaultArgon2KeyLength,
				SaltLength:    defaultArgon2SaltLength,
				MaxConcurrent: l.positiveInt(envArgon2MaxConcurrent, defaultArgon2MaxConcurrent),
			},
			Password: PasswordPolicy{
				MinLength: l.positiveInt(envPasswordMinLength, defaultPasswordMinLength),
				MaxLength: l.positiveInt(envPasswordMaxLength, defaultPasswordMaxLength),
			},
			Lockout: Lockout{
				MaxFailures: l.positiveInt(envLockoutMaxFailures, defaultLockoutMaxFailures),
				BaseDelay:   l.duration(envLockoutBaseDelay, defaultLockoutBaseDelay),
				MaxDelay:    l.duration(envLockoutMaxDelay, defaultLockoutMaxDelay),
			},
			RateLimits: RateLimits{
				LoginPerIP:            l.rateRule(envRateLoginPerIP, defaultRateLoginPerIP),
				LoginPerIdentifier:    l.rateRule(envRateLoginPerIdentifier, defaultRateLoginPerIdentifier),
				RegisterPerIP:         l.rateRule(envRateRegisterPerIP, defaultRateRegisterPerIP),
				PasswordResetPerIP:    l.rateRule(envRatePasswordResetPerIP, defaultRatePasswordResetPerIP),
				PasswordResetPerEmail: l.rateRule(envRatePasswordResetPerKey, defaultRatePasswordResetPerKey),
				IncidentCreatePerUser: l.rateRule(envRateIncidentCreatePerUser, defaultRateIncidentCreatePerUser),
				MaxTrackedKeys:        l.positiveInt(envMaxTrackedRateKeys, defaultMaxTrackedRateKeys),
			},
			Tokens: Tokens{
				ActivationTTL:    l.duration(envActivationTokenTTL, defaultActivationTokenTTL),
				PasswordResetTTL: l.duration(envPasswordResetTokenTTL, defaultPasswordResetTokenTTL),
			},
			CSRFSecret: l.base64Secret(envCSRFSecret),
		},
		Crypto: Crypto{
			Keys:        cryptoKeys,
			ActiveKeyID: l.activeKeyID(cryptoKeys),
		},
		Mail: l.mail(production),
		OMDb: OMDb{
			APIKey:  l.text(envOMDbAPIKey, ""),
			BaseURL: l.text(envOMDbBaseURL, defaultOMDbBaseURL),
			Timeout: l.duration(envOMDbTimeout, defaultOMDbTimeout),
		},
		App: l.app(),
		Jobs: Jobs{
			CleanupInterval:  l.duration(envCleanupInterval, defaultCleanupInterval),
			SessionRetention: l.duration(envSessionRetention, defaultSessionRetention),
			TokenRetention:   l.duration(envTokenRetention, defaultTokenRetention),
		},
	}

	l.errs = append(l.errs, cfg.validate())
	return cfg, errors.Join(l.errs...)
}

func (l *loader) mail(production bool) Mail {
	defaultDriver, defaultTLS := MailDriverLog, TLSModeNone
	if production {
		defaultDriver, defaultTLS = MailDriverSMTP, TLSModeStartTLS
	}

	driver := MailDriver(l.text(envMailDriver, string(defaultDriver)))
	if driver != MailDriverLog && driver != MailDriverSMTP {
		l.fail("%s : valeur inconnue %q (log ou smtp)", envMailDriver, driver)
		driver = defaultDriver
	}
	tlsMode := TLSMode(l.text(envSMTPTLSMode, string(defaultTLS)))
	if tlsMode != TLSModeNone && tlsMode != TLSModeStartTLS && tlsMode != TLSModeImplicit {
		l.fail("%s : valeur inconnue %q (none, starttls ou tls)", envSMTPTLSMode, tlsMode)
		tlsMode = defaultTLS
	}

	return Mail{
		Driver:      driver,
		FromAddress: l.text(envMailFromAddress, defaultMailFromAddress),
		FromName:    l.text(envMailFromName, defaultMailFromName),
		SMTP: SMTP{
			Host:     l.text(envSMTPHost, defaultSMTPHost),
			Port:     l.positiveInt(envSMTPPort, defaultSMTPPort),
			Username: l.text(envSMTPUsername, ""),
			Password: l.text(envSMTPPassword, ""),
			TLSMode:  tlsMode,
			Timeout:  l.duration(envSMTPTimeout, defaultSMTPTimeout),
		},
		Workers:     l.positiveInt(envMailWorkers, defaultMailWorkers),
		QueueSize:   l.positiveInt(envMailQueueSize, defaultMailQueueSize),
		MaxAttempts: l.positiveInt(envMailMaxAttempts, defaultMailMaxAttempts),
		RetryDelay:  l.duration(envMailRetryDelay, defaultMailRetryDelay),
		SendTimeout: l.duration(envMailSendTimeout, defaultMailSendTimeout),
	}
}

func (l *loader) app() App {
	raw := l.text(envPublicBaseURL, defaultPublicBaseURL)
	base, err := url.Parse(raw)
	if err != nil || base.Scheme == "" || base.Host == "" {
		l.fail("%s : URL absolue attendue, reçu %q", envPublicBaseURL, raw)
		base, _ = url.Parse(defaultPublicBaseURL)
	}
	return App{
		PublicBaseURL:     base,
		ActivationPath:    l.text(envActivationPath, defaultActivationPath),
		PasswordResetPath: l.text(envPasswordResetPath, defaultPasswordResetPath),
	}
}
