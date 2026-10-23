// Package config centralise tous les paramètres du backend : lecture des variables
// d'environnement, valeurs par défaut et validation. Aucun autre paquet ne lit l'environnement.
package config

import (
	"errors"
	"log/slog"
	"math"
	"net/netip"
	"net/url"
	"os"
	"time"
)

// Environment désigne le contexte d'exécution. Les règles de validation sont plus strictes
// en production.
type Environment string

// Environnements connus ; la production active des contrôles de démarrage plus stricts.
const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
)

// LogFormat choisit la forme des lignes de journal.
type LogFormat string

// Formats de journal.
const (
	LogFormatJSON LogFormat = "json" // une ligne JSON par entrée, pour Filebeat et Logstash (défaut en production)
	LogFormatText LogFormat = "text" // clé=valeur lisible dans un terminal (défaut ailleurs)
)

// MailDriver choisit comment les mails sont envoyés.
type MailDriver string

// Pilotes d'envoi de mails.
const (
	MailDriverLog  MailDriver = "log"  // écrit les mails dans les logs (développement uniquement)
	MailDriverSMTP MailDriver = "smtp" // envoie via un serveur SMTP
)

// MailQueue choisit où attendent les mails avant leur envoi.
type MailQueue string

// Files d'attente des mails.
const (
	// MailQueueMemory est une file bornée dans le processus de l'API : simple, mais les mails en attente
	// sont perdus si l'API s'arrête brutalement.
	MailQueueMemory MailQueue = "memory"
	// MailQueueRabbitMQ dépose chaque mail dans une file durable de RabbitMQ ; le worker `mailworker` les envoie.
	MailQueueRabbitMQ MailQueue = "rabbitmq"
)

// TLSMode décrit le chiffrement de la connexion SMTP.
type TLSMode string

// Modes de chiffrement de la connexion SMTP.
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
	// LogFormat est le format des journaux : JSON (lu par le monitoring) ou texte lisible.
	LogFormat   LogFormat
	AutoMigrate bool
	HTTP        HTTP
	Database    Database
	Session     Session
	Security    Security
	Crypto      Crypto
	Mail        Mail
	OMDb        OMDb
	Avatar      Avatar
	App         App
	Jobs        Jobs
	Events      Events
	Dashboard   Dashboard
	Proximity   Proximity
}

// OMDb regroupe les réglages du catalogue de films.
type OMDb struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// Avatar regroupe les réglages des photos de profil.
type Avatar struct {
	// Dir est le dossier où les photos sont écrites : un volume inscriptible, car l'API tourne
	// sinon avec un système de fichiers en lecture seule.
	Dir string
	// MaxBytes borne la taille du fichier reçu.
	MaxBytes int64
	// MaxDimension borne le côté de l'image, en pixels (protège contre les images « bombes »).
	MaxDimension int
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
	ContactPerIP          RateRule
	NewsletterPerIP       RateRule
	NewsletterPerEmail    RateRule
	EmailChangePerUser    RateRule
	AvatarPerUser         RateRule
	MFAVerifyPerUser      RateRule
	// PasswordCheckPerUser borne les actions qui redemandent le mot de passe actuel (changement du mot de
	// passe, suppression du compte).
	PasswordCheckPerUser RateRule
	HeroLocationPerUser  RateRule
	MaxTrackedKeys       int
}

// Tokens regroupe la durée de vie des jetons à usage unique envoyés par mail.
type Tokens struct {
	ActivationTTL    time.Duration
	PasswordResetTTL time.Duration
	// NewsletterConfirmTTL est la validité du lien de confirmation d'un abonnement (signé, sans
	// jeton en base).
	NewsletterConfirmTTL time.Duration
	// EmailChangeConfirmTTL est la validité du lien envoyé à la nouvelle adresse d'un compte pour la
	// confirmer (signé, sans jeton en base).
	EmailChangeConfirmTTL time.Duration
}

// MFA regroupe les réglages de la double authentification (TOTP).
type MFA struct {
	// ModeratorRequired impose aux modérateurs d'activer la 2FA : sans elle, leur session ne donne accès
	// qu'à l'activation. Vrai par défaut en production, où il ne peut pas être désactivé ; faux en
	// développement pour ne pas gêner une démonstration.
	ModeratorRequired bool
	// Issuer est le nom du service affiché dans l'application d'authentification.
	Issuer string
	// RecoveryCodes est le nombre de codes de secours remis à l'activation.
	RecoveryCodes int
}

// Security regroupe les réglages de sécurité.
type Security struct {
	Argon2     Argon2
	Password   PasswordPolicy
	Lockout    Lockout
	RateLimits RateLimits
	Tokens     Tokens
	CSRFSecret []byte
	MFA        MFA
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

// RabbitMQ regroupe les réglages de la connexion au courtier de messages (file de mails).
type RabbitMQ struct {
	// URL est l'adresse du courtier (amqp://utilisateur:motdepasse@hôte:5672/). Elle porte un mot de passe :
	// elle n'est jamais journalisée, et en production elle vient d'un secret (HEROS_RABBITMQ_URL_FILE).
	URL string
	// ConfirmTimeout borne l'attente de la confirmation du courtier après le dépôt d'un mail.
	ConfirmTimeout time.Duration
	// ReconnectDelay sépare deux tentatives de connexion quand le courtier est injoignable.
	ReconnectDelay time.Duration
}

// Mail regroupe les réglages d'envoi des mails et de la file d'envoi asynchrone.
type Mail struct {
	Driver MailDriver
	// Queue dit où les mails attendent : en mémoire (par défaut) ou dans RabbitMQ.
	Queue MailQueue
	// RabbitMQ n'est lu que si Queue vaut MailQueueRabbitMQ.
	RabbitMQ    RabbitMQ
	FromAddress string
	FromName    string
	SMTP        SMTP
	Workers     int
	QueueSize   int
	MaxAttempts int
	RetryDelay  time.Duration
	SendTimeout time.Duration
	// ContactRecipient reçoit les messages du formulaire de contact.
	ContactRecipient string
}

// App regroupe les réglages de l'application visibles des utilisateurs.
type App struct {
	// PublicBaseURL est l'adresse publique du frontend, utilisée pour les liens des mails.
	PublicBaseURL     *url.URL
	ActivationPath    string
	PasswordResetPath string
	// NewsletterConfirmPath est la page du frontend qui confirme un abonnement (`?token=…`).
	NewsletterConfirmPath string
	// NewsletterUnsubscribePath est la page du frontend qui désabonne de la newsletter (`?token=…`).
	NewsletterUnsubscribePath string
	// EmailChangeConfirmPath est la page du frontend qui confirme la nouvelle adresse d'un compte
	// (`?token=…`).
	EmailChangeConfirmPath string
}

// MailLogoPath est l'emplacement, dans le frontend, du logo affiché en tête des mails.
const MailLogoPath = "/mail/logo.png"

// MailLogoURL construit l'adresse absolue du logo des mails.
func (a App) MailLogoURL() string {
	target := *a.PublicBaseURL
	target.Path = MailLogoPath
	return target.String()
}

// ActivationLink construit le lien d'activation de compte envoyé par mail.
func (a App) ActivationLink(token string) string {
	return a.link(a.ActivationPath, token)
}

// PasswordResetLink construit le lien de réinitialisation de mot de passe envoyé par mail.
func (a App) PasswordResetLink(token string) string {
	return a.link(a.PasswordResetPath, token)
}

// NewsletterConfirmLink construit le lien de confirmation d'un abonnement à la newsletter.
func (a App) NewsletterConfirmLink(token string) string {
	return a.link(a.NewsletterConfirmPath, token)
}

// NewsletterUnsubscribeLink construit le lien de désinscription que porte chaque newsletter.
func (a App) NewsletterUnsubscribeLink(token string) string {
	return a.link(a.NewsletterUnsubscribePath, token)
}

// EmailChangeConfirmLink construit le lien qui confirme la nouvelle adresse d'un compte.
func (a App) EmailChangeConfirmLink(token string) string {
	return a.link(a.EmailChangeConfirmPath, token)
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
	CleanupInterval time.Duration
	// CleanupTimeout borne un passage du nettoyage : une requête bloquée ne doit pas immobiliser la tâche, ni
	// garder une connexion du pool indéfiniment.
	CleanupTimeout   time.Duration
	SessionRetention time.Duration
	TokenRetention   time.Duration
	// CommandRetention est la durée de conservation des commandes rejouables (clés d'idempotence).
	CommandRetention time.Duration
}

// Events regroupe les réglages du journal d'événements (voir le paquet events).
type Events struct {
	// PollInterval est la fréquence de lecture des nouveaux événements.
	PollInterval time.Duration
	// BatchSize borne le nombre d'événements lus par passage.
	BatchSize int
	// Heartbeat est le délai entre deux signaux de vie sur un flux ouvert.
	Heartbeat time.Duration
	// MaxStreamsPerUser borne les flux temps réel ouverts en même temps par un utilisateur.
	MaxStreamsPerUser int
	// StreamMaxAge est la durée maximale d'un flux : à son terme le serveur le ferme, et le navigateur se
	// reconnecte (la session est alors revérifiée).
	StreamMaxAge time.Duration
	// StreamBuffer est le nombre d'événements gardés en attente pour un flux lent avant de le fermer.
	StreamBuffer int
}

// Dashboard regroupe les réglages du tableau de bord des héros.
type Dashboard struct {
	// StatsTTL est la fraîcheur des statistiques (globales, et celles de chaque héros) : au-delà, la dernière valeur
	// est encore servie sur-le-champ pendant qu'une nouvelle est calculée en arrière-plan, car ces agrégats parcourent
	// de très grosses tables. Zéro désactive ce cache : chaque appel recalcule.
	StatsTTL time.Duration
	// Warmup calcule d'avance, au démarrage, les statistiques de chaque héros : sans cela, le premier affichage de
	// chacun calcule en direct (environ 11 s à 5 millions d'incidents). Sans effet quand le cache est désactivé.
	Warmup bool
}

// Proximity regroupe les réglages de la localisation des héros.
type Proximity struct {
	// LocationTTL : au-delà, une position n'est plus utilisée pour l'alerte et elle est purgée.
	LocationTTL time.Duration
	// AlertRadiusKm : rayon autour d'un incident dans lequel les héros sont prévenus.
	AlertRadiusKm int
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
		LogFormat:   l.logFormat(production),
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
				Time:          uint32(l.boundedInt(envArgon2Time, defaultArgon2Time, math.MaxUint32)),
				MemoryKiB:     uint32(l.boundedInt(envArgon2MemoryKiB, defaultArgon2MemoryKiB, math.MaxUint32)),
				Threads:       uint8(l.boundedInt(envArgon2Threads, defaultArgon2Threads, math.MaxUint8)),
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
				ContactPerIP:          l.rateRule(envRateContactPerIP, defaultRateContactPerIP),
				NewsletterPerIP:       l.rateRule(envRateNewsletterPerIP, defaultRateNewsletterPerIP),
				NewsletterPerEmail:    l.rateRule(envRateNewsletterPerEmail, defaultRateNewsletterPerEmail),
				EmailChangePerUser:    l.rateRule(envRateEmailChangePerUser, defaultRateEmailChangePerUser),
				AvatarPerUser:         l.rateRule(envRateAvatarPerUser, defaultRateAvatarPerUser),
				MFAVerifyPerUser:      l.rateRule(envRateMFAVerifyPerUser, defaultRateMFAVerifyPerUser),
				PasswordCheckPerUser:  l.rateRule(envRatePasswordCheckPerUser, defaultRatePasswordCheckPerUser),
				HeroLocationPerUser:   l.rateRule(envRateHeroLocationPerUser, defaultRateHeroLocationPerUser),
				MaxTrackedKeys:        l.positiveInt(envMaxTrackedRateKeys, defaultMaxTrackedRateKeys),
			},
			Tokens: Tokens{
				ActivationTTL:    l.duration(envActivationTokenTTL, defaultActivationTokenTTL),
				PasswordResetTTL: l.duration(envPasswordResetTokenTTL, defaultPasswordResetTokenTTL),

				NewsletterConfirmTTL:  l.duration(envNewsletterConfirmTTL, defaultNewsletterConfirmTTL),
				EmailChangeConfirmTTL: l.duration(envEmailChangeConfirmTTL, defaultEmailChangeConfirmTTL),
			},
			CSRFSecret: l.base64Secret(envCSRFSecret),
			MFA: MFA{
				ModeratorRequired: l.boolean(envMFAModeratorRequired, production),
				Issuer:            l.text(envMFAIssuer, defaultMFAIssuer),
				RecoveryCodes:     defaultMFARecoveryCodes,
			},
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
		Avatar: Avatar{
			Dir:          l.text(envAvatarDir, defaultAvatarDir),
			MaxBytes:     int64(l.positiveInt(envAvatarMaxBytes, defaultAvatarMaxBytes)),
			MaxDimension: l.positiveInt(envAvatarMaxDimension, defaultAvatarMaxDimension),
		},
		App: l.app(),
		Jobs: Jobs{
			CleanupInterval:  l.duration(envCleanupInterval, defaultCleanupInterval),
			CleanupTimeout:   l.duration(envCleanupTimeout, defaultCleanupTimeout),
			SessionRetention: l.duration(envSessionRetention, defaultSessionRetention),
			TokenRetention:   l.duration(envTokenRetention, defaultTokenRetention),
			CommandRetention: l.duration(envCommandRetention, defaultCommandRetention),
		},
		Events: Events{
			PollInterval: l.duration(envEventsPollInterval, defaultEventsPollInterval),
			BatchSize:    l.positiveInt(envEventsBatchSize, defaultEventsBatchSize),
			Heartbeat:    l.duration(envEventsHeartbeat, defaultEventsHeartbeat),

			MaxStreamsPerUser: l.positiveInt(envEventsMaxStreamsPerUser, defaultEventsMaxStreamsPerUser),
			StreamMaxAge:      l.duration(envEventsStreamMaxAge, defaultEventsStreamMaxAge),
			StreamBuffer:      l.positiveInt(envEventsStreamBuffer, defaultEventsStreamBuffer),
		},
		Dashboard: Dashboard{
			StatsTTL: l.durationOrZero(envDashboardStatsTTL, defaultDashboardStatsTTL),
			Warmup:   l.boolean(envDashboardWarmup, true),
		},
		Proximity: Proximity{
			LocationTTL:   l.duration(envLocationTTL, defaultLocationTTL),
			AlertRadiusKm: l.positiveInt(envAlertRadiusKm, defaultAlertRadiusKm),
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
	queue := MailQueue(l.text(envMailQueue, string(MailQueueMemory)))
	if queue != MailQueueMemory && queue != MailQueueRabbitMQ {
		l.fail("%s : valeur inconnue %q (memory ou rabbitmq)", envMailQueue, queue)
		queue = MailQueueMemory
	}
	tlsMode := TLSMode(l.text(envSMTPTLSMode, string(defaultTLS)))
	if tlsMode != TLSModeNone && tlsMode != TLSModeStartTLS && tlsMode != TLSModeImplicit {
		l.fail("%s : valeur inconnue %q (none, starttls ou tls)", envSMTPTLSMode, tlsMode)
		tlsMode = defaultTLS
	}

	return Mail{
		Driver: driver,
		Queue:  queue,
		RabbitMQ: RabbitMQ{
			URL:            l.text(envRabbitMQURL, ""),
			ConfirmTimeout: l.duration(envRabbitMQConfirmTimeout, defaultRabbitMQConfirmTimeout),
			ReconnectDelay: l.duration(envRabbitMQReconnectDelay, defaultRabbitMQReconnectDelay),
		},
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

		ContactRecipient: l.text(envContactRecipient, defaultContactRecipient),
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

		NewsletterConfirmPath:     l.text(envNewsletterConfirmPath, defaultNewsletterConfirmPath),
		NewsletterUnsubscribePath: l.text(envNewsletterUnsubscribePath, defaultNewsletterUnsubscribePath),
		EmailChangeConfirmPath:    l.text(envEmailChangeConfirmPath, defaultEmailChangeConfirmPath),
	}
}
