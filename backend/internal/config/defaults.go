package config

import "time"

// Valeurs par défaut de la configuration. C'est le seul endroit du backend où l'on trouve des
// durées, des tailles ou des seuils codés en dur : le reste du code les lit depuis Config.
// Chaque valeur peut être surchargée par une variable d'environnement (voir env.go).
const (
	defaultHTTPAddr           = ":8080"
	defaultReadHeaderTimeout  = 5 * time.Second
	defaultReadTimeout        = 15 * time.Second
	defaultWriteTimeout       = 30 * time.Second
	defaultIdleTimeout        = 60 * time.Second
	defaultShutdownTimeout    = 20 * time.Second
	defaultMaxBodyBytes       = 1 << 20 // 1 Mio
	defaultHealthcheckTimeout = 3 * time.Second

	defaultDBMaxOpenConns    = 25
	defaultDBMaxIdleConns    = 5
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBConnMaxIdleTime = 10 * time.Minute
	defaultDBPingTimeout     = 5 * time.Second
	defaultDBSlowQuery       = 200 * time.Millisecond

	defaultMemberSessionIdle  = 2 * time.Hour
	defaultMemberSessionMax   = 7 * 24 * time.Hour
	defaultRememberMeIdle     = 30 * 24 * time.Hour
	defaultRememberMeMax      = 30 * 24 * time.Hour
	defaultModeratorIdle      = 15 * time.Minute
	defaultModeratorMax       = 8 * time.Hour
	defaultSessionTouchPeriod = time.Minute

	defaultArgon2Time          = 3
	defaultArgon2MemoryKiB     = 64 * 1024
	defaultArgon2Threads       = 2
	defaultArgon2KeyLength     = 32
	defaultArgon2SaltLength    = 16
	defaultArgon2MaxConcurrent = 4

	defaultPasswordMinLength = 10
	defaultPasswordMaxLength = 128

	defaultLockoutMaxFailures = 5
	defaultLockoutBaseDelay   = time.Minute
	defaultLockoutMaxDelay    = time.Hour

	defaultMaxTrackedRateKeys = 100_000

	defaultActivationTokenTTL    = 24 * time.Hour
	defaultPasswordResetTokenTTL = time.Hour

	defaultNewsletterConfirmTTL = 48 * time.Hour
	// Le lien de confirmation de la nouvelle adresse d'un compte : même durée que celui de la newsletter.
	defaultEmailChangeConfirmTTL = 48 * time.Hour

	defaultMailFromAddress = "no-reply@heros-sos.local"
	defaultMailFromName    = "Heros-SOS"
	defaultSMTPHost        = "localhost"
	defaultSMTPPort        = 1025 // port de Mailpit en développement
	defaultSMTPTimeout     = 10 * time.Second
	defaultMailWorkers     = 4
	defaultMailQueueSize   = 256
	defaultMailMaxAttempts = 3
	defaultMailRetryDelay  = 2 * time.Second
	defaultMailSendTimeout = 30 * time.Second
	defaultOMDbBaseURL     = "https://www.omdbapi.com/"
	defaultOMDbTimeout     = 5 * time.Second

	// Le « Bureau » : destinataire des messages du formulaire de contact.
	defaultContactRecipient = "contact@heros-sos.local"

	// Photos de profil : le frontend les recadre en 256 px et refuse au-delà de 2 Mio.
	defaultAvatarDir          = "data/avatars"
	defaultAvatarMaxBytes     = 2 << 20 // 2 Mio
	defaultAvatarMaxDimension = 1024

	defaultPublicBaseURL     = "http://localhost:3000"
	defaultActivationPath    = "/activation"
	defaultPasswordResetPath = "/reinitialisation-mot-de-passe"

	defaultNewsletterConfirmPath     = "/newsletter/confirmation"
	defaultNewsletterUnsubscribePath = "/newsletter/desinscription"
	defaultEmailChangeConfirmPath    = "/changement-email/confirmation"

	// Double authentification (TOTP) : nom affiché dans l'application d'authentification, et nombre de
	// codes de secours remis à l'activation.
	defaultMFAIssuer        = "Heros-SOS"
	defaultMFARecoveryCodes = 10

	defaultCleanupInterval  = 15 * time.Minute
	defaultSessionRetention = 7 * 24 * time.Hour
	defaultTokenRetention   = 7 * 24 * time.Hour
)

// Règles de limitation de débit par défaut (maximum de requêtes par fenêtre).
var (
	defaultRateLoginPerIP            = RateRule{Max: 20, Window: 15 * time.Minute}
	defaultRateLoginPerIdentifier    = RateRule{Max: 5, Window: 15 * time.Minute}
	defaultRateRegisterPerIP         = RateRule{Max: 5, Window: time.Hour}
	defaultRatePasswordResetPerIP    = RateRule{Max: 10, Window: time.Hour}
	defaultRatePasswordResetPerKey   = RateRule{Max: 3, Window: time.Hour}
	defaultRateIncidentCreatePerUser = RateRule{Max: 10, Window: time.Hour}

	defaultRateContactPerIP       = RateRule{Max: 5, Window: time.Hour}
	defaultRateNewsletterPerIP    = RateRule{Max: 10, Window: time.Hour}
	defaultRateNewsletterPerEmail = RateRule{Max: 3, Window: time.Hour}
	defaultRateEmailChangePerUser = RateRule{Max: 5, Window: time.Hour}
	defaultRateAvatarPerUser      = RateRule{Max: 20, Window: time.Hour}
	// Un code TOTP a un million de valeurs : cinq essais par fenêtre de cinq minutes rendent leur
	// énumération hors de portée, sans gêner une faute de frappe.
	defaultRateMFAVerifyPerUser = RateRule{Max: 5, Window: 5 * time.Minute}
)

// Plancher de sécurité imposé en production pour le hachage des mots de passe
// (recommandation OWASP : 19 Mio de mémoire, 2 itérations).
const (
	productionMinArgon2MemoryKiB = 19 * 1024
	productionMinArgon2Time      = 2
)

// Taille attendue d'une clé maîtresse de chiffrement (cascade à 512 bits) et du secret CSRF.
const (
	MasterKeyLength = 64
	minCSRFSecret   = 32
)
