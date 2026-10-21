package config

// Noms des variables d'environnement lues par le backend, toutes préfixées par HEROS_.
// Chacune doit figurer dans backend/.env.example (un test le vérifie).
const (
	envEnvironment = "HEROS_ENV"
	envLogLevel    = "HEROS_LOG_LEVEL"
	envAutoMigrate = "HEROS_AUTO_MIGRATE"

	envHTTPAddr              = "HEROS_HTTP_ADDR"
	envHTTPReadHeaderTimeout = "HEROS_HTTP_READ_HEADER_TIMEOUT"
	envHTTPReadTimeout       = "HEROS_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout      = "HEROS_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout       = "HEROS_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout   = "HEROS_HTTP_SHUTDOWN_TIMEOUT"
	envHTTPMaxBodyBytes      = "HEROS_HTTP_MAX_BODY_BYTES"
	envHealthcheckTimeout    = "HEROS_HEALTHCHECK_TIMEOUT"
	envTrustedProxies        = "HEROS_TRUSTED_PROXIES"
	envAllowedOrigins        = "HEROS_ALLOWED_ORIGINS"

	envDatabaseURL             = "HEROS_DATABASE_URL"
	envDatabaseMaxOpenConns    = "HEROS_DATABASE_MAX_OPEN_CONNS"
	envDatabaseMaxIdleConns    = "HEROS_DATABASE_MAX_IDLE_CONNS"
	envDatabaseConnMaxLifetime = "HEROS_DATABASE_CONN_MAX_LIFETIME"
	envDatabaseConnMaxIdleTime = "HEROS_DATABASE_CONN_MAX_IDLE_TIME"
	envDatabasePingTimeout     = "HEROS_DATABASE_PING_TIMEOUT"
	envDatabaseSlowQuery       = "HEROS_DATABASE_SLOW_QUERY_THRESHOLD"

	envCookieSecure         = "HEROS_COOKIE_SECURE"
	envSessionMemberIdle    = "HEROS_SESSION_MEMBER_IDLE"
	envSessionMemberMax     = "HEROS_SESSION_MEMBER_MAX"
	envSessionRememberIdle  = "HEROS_SESSION_REMEMBER_IDLE"
	envSessionRememberMax   = "HEROS_SESSION_REMEMBER_MAX"
	envSessionModeratorIdle = "HEROS_SESSION_MODERATOR_IDLE"
	envSessionModeratorMax  = "HEROS_SESSION_MODERATOR_MAX"
	envSessionTouchPeriod   = "HEROS_SESSION_TOUCH_PERIOD"

	envArgon2Time          = "HEROS_ARGON2_TIME"
	envArgon2MemoryKiB     = "HEROS_ARGON2_MEMORY_KIB"
	envArgon2Threads       = "HEROS_ARGON2_THREADS"
	envArgon2MaxConcurrent = "HEROS_ARGON2_MAX_CONCURRENT"

	envPasswordMinLength = "HEROS_PASSWORD_MIN_LENGTH"
	envPasswordMaxLength = "HEROS_PASSWORD_MAX_LENGTH"

	envLockoutMaxFailures = "HEROS_LOCKOUT_MAX_FAILURES"
	envLockoutBaseDelay   = "HEROS_LOCKOUT_BASE_DELAY"
	envLockoutMaxDelay    = "HEROS_LOCKOUT_MAX_DELAY"

	envRateLoginPerIP            = "HEROS_RATE_LOGIN_PER_IP"
	envRateLoginPerIdentifier    = "HEROS_RATE_LOGIN_PER_IDENTIFIER"
	envRateRegisterPerIP         = "HEROS_RATE_REGISTER_PER_IP"
	envRatePasswordResetPerIP    = "HEROS_RATE_PASSWORD_RESET_PER_IP"
	envRatePasswordResetPerKey   = "HEROS_RATE_PASSWORD_RESET_PER_EMAIL"
	envRateIncidentCreatePerUser = "HEROS_RATE_INCIDENT_CREATE_PER_USER"
	envMaxTrackedRateKeys        = "HEROS_RATE_MAX_TRACKED_KEYS"

	envActivationTokenTTL    = "HEROS_ACTIVATION_TOKEN_TTL"
	envPasswordResetTokenTTL = "HEROS_PASSWORD_RESET_TOKEN_TTL"

	envCSRFSecret    = "HEROS_CSRF_SECRET"
	envCryptoKeys    = "HEROS_CRYPTO_KEYS"
	envCryptoActiveK = "HEROS_CRYPTO_ACTIVE_KEY_ID"

	envMailDriver      = "HEROS_MAIL_DRIVER"
	envMailFromAddress = "HEROS_MAIL_FROM_ADDRESS"
	envMailFromName    = "HEROS_MAIL_FROM_NAME"
	envSMTPHost        = "HEROS_SMTP_HOST"
	envSMTPPort        = "HEROS_SMTP_PORT"
	envSMTPUsername    = "HEROS_SMTP_USERNAME"
	envSMTPPassword    = "HEROS_SMTP_PASSWORD"
	envSMTPTLSMode     = "HEROS_SMTP_TLS_MODE"
	envSMTPTimeout     = "HEROS_SMTP_TIMEOUT"
	envMailWorkers     = "HEROS_MAIL_WORKERS"
	envMailQueueSize   = "HEROS_MAIL_QUEUE_SIZE"
	envMailMaxAttempts = "HEROS_MAIL_MAX_ATTEMPTS"
	envMailRetryDelay  = "HEROS_MAIL_RETRY_DELAY"
	envMailSendTimeout = "HEROS_MAIL_SEND_TIMEOUT"

	envOMDbAPIKey  = "HEROS_OMDB_API_KEY"
	envOMDbBaseURL = "HEROS_OMDB_BASE_URL"
	envOMDbTimeout = "HEROS_OMDB_TIMEOUT"

	envPublicBaseURL     = "HEROS_PUBLIC_BASE_URL"
	envActivationPath    = "HEROS_ACTIVATION_PATH"
	envPasswordResetPath = "HEROS_PASSWORD_RESET_PATH"

	envCleanupInterval  = "HEROS_CLEANUP_INTERVAL"
	envSessionRetention = "HEROS_SESSION_RETENTION"
	envTokenRetention   = "HEROS_TOKEN_RETENTION"
)

// EnvVarNames retourne la liste de toutes les variables d'environnement reconnues.
// Elle sert à vérifier que .env.example reste synchronisé avec le code.
func EnvVarNames() []string {
	return []string{
		envEnvironment, envLogLevel, envAutoMigrate,
		envHTTPAddr, envHTTPReadHeaderTimeout, envHTTPReadTimeout, envHTTPWriteTimeout,
		envHTTPIdleTimeout, envHTTPShutdownTimeout, envHTTPMaxBodyBytes, envHealthcheckTimeout, envTrustedProxies,
		envAllowedOrigins,
		envDatabaseURL, envDatabaseMaxOpenConns, envDatabaseMaxIdleConns,
		envDatabaseConnMaxLifetime, envDatabaseConnMaxIdleTime, envDatabasePingTimeout,
		envDatabaseSlowQuery,
		envCookieSecure, envSessionMemberIdle, envSessionMemberMax, envSessionRememberIdle,
		envSessionRememberMax, envSessionModeratorIdle, envSessionModeratorMax,
		envSessionTouchPeriod,
		envArgon2Time, envArgon2MemoryKiB, envArgon2Threads, envArgon2MaxConcurrent,
		envPasswordMinLength, envPasswordMaxLength,
		envLockoutMaxFailures, envLockoutBaseDelay, envLockoutMaxDelay,
		envRateLoginPerIP, envRateLoginPerIdentifier, envRateRegisterPerIP,
		envRatePasswordResetPerIP, envRatePasswordResetPerKey, envRateIncidentCreatePerUser, envMaxTrackedRateKeys,
		envActivationTokenTTL, envPasswordResetTokenTTL,
		envCSRFSecret, envCryptoKeys, envCryptoActiveK,
		envMailDriver, envMailFromAddress, envMailFromName, envSMTPHost, envSMTPPort,
		envSMTPUsername, envSMTPPassword, envSMTPTLSMode, envSMTPTimeout, envMailWorkers,
		envMailQueueSize, envMailMaxAttempts, envMailRetryDelay, envMailSendTimeout,
		envOMDbAPIKey, envOMDbBaseURL, envOMDbTimeout,
		envPublicBaseURL, envActivationPath, envPasswordResetPath,
		envCleanupInterval, envSessionRetention, envTokenRetention,
	}
}
