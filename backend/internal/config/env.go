package config

// Noms des variables d'environnement lues par le backend, toutes préfixées par HEROS_.
// Chacune doit figurer dans backend/.env.example (un test le vérifie).
const (
	envEnvironment = "HEROS_ENV"
	envLogLevel    = "HEROS_LOG_LEVEL"
	envLogFormat   = "HEROS_LOG_FORMAT"
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

	envRateContactPerIP         = "HEROS_RATE_CONTACT_PER_IP"
	envRateNewsletterPerIP      = "HEROS_RATE_NEWSLETTER_PER_IP"
	envRateNewsletterPerEmail   = "HEROS_RATE_NEWSLETTER_PER_EMAIL"
	envRateEmailChangePerUser   = "HEROS_RATE_EMAIL_CHANGE_PER_USER"
	envRateAvatarPerUser        = "HEROS_RATE_AVATAR_PER_USER"
	envRateMFAVerifyPerUser     = "HEROS_RATE_MFA_VERIFY_PER_USER"
	envRatePasswordCheckPerUser = "HEROS_RATE_PASSWORD_CHECK_PER_USER"
	envRateExportPerUser        = "HEROS_RATE_EXPORT_PER_USER"
	envRateHeroLocationPerUser  = "HEROS_RATE_HERO_LOCATION_PER_USER"

	envActivationTokenTTL    = "HEROS_ACTIVATION_TOKEN_TTL"
	envPasswordResetTokenTTL = "HEROS_PASSWORD_RESET_TOKEN_TTL"

	envNewsletterConfirmTTL  = "HEROS_NEWSLETTER_CONFIRM_TTL"
	envEmailChangeConfirmTTL = "HEROS_EMAIL_CHANGE_CONFIRM_TTL"

	envMFAModeratorRequired = "HEROS_MFA_MODERATOR_REQUIRED"
	envMFAIssuer            = "HEROS_MFA_ISSUER"

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
	envMailQueue       = "HEROS_MAIL_QUEUE"

	envRabbitMQURL            = "HEROS_RABBITMQ_URL"
	envRabbitMQConfirmTimeout = "HEROS_RABBITMQ_CONFIRM_TIMEOUT"
	envRabbitMQReconnectDelay = "HEROS_RABBITMQ_RECONNECT_DELAY"

	envOMDbAPIKey  = "HEROS_OMDB_API_KEY"
	envOMDbBaseURL = "HEROS_OMDB_BASE_URL"
	envOMDbTimeout = "HEROS_OMDB_TIMEOUT"

	envContactRecipient = "HEROS_CONTACT_RECIPIENT"

	envAvatarDir          = "HEROS_AVATAR_DIR"
	envAvatarMaxBytes     = "HEROS_AVATAR_MAX_BYTES"
	envAvatarMaxDimension = "HEROS_AVATAR_MAX_DIMENSION"

	envPublicBaseURL     = "HEROS_PUBLIC_BASE_URL"
	envActivationPath    = "HEROS_ACTIVATION_PATH"
	envPasswordResetPath = "HEROS_PASSWORD_RESET_PATH"

	envNewsletterConfirmPath     = "HEROS_NEWSLETTER_CONFIRM_PATH"
	envNewsletterUnsubscribePath = "HEROS_NEWSLETTER_UNSUBSCRIBE_PATH"
	envEmailChangeConfirmPath    = "HEROS_EMAIL_CHANGE_CONFIRM_PATH"

	envCleanupInterval  = "HEROS_CLEANUP_INTERVAL"
	envCleanupTimeout   = "HEROS_CLEANUP_TIMEOUT"
	envSessionRetention = "HEROS_SESSION_RETENTION"
	envTokenRetention   = "HEROS_TOKEN_RETENTION"
	envCommandRetention = "HEROS_COMMAND_RETENTION"

	envEventsPollInterval = "HEROS_EVENTS_POLL_INTERVAL"
	envEventsBatchSize    = "HEROS_EVENTS_BATCH_SIZE"
	envEventsHeartbeat    = "HEROS_EVENTS_HEARTBEAT"

	envEventsMaxStreamsPerUser = "HEROS_EVENTS_MAX_STREAMS_PER_USER"
	envEventsStreamMaxAge      = "HEROS_EVENTS_STREAM_MAX_AGE"
	envEventsStreamBuffer      = "HEROS_EVENTS_STREAM_BUFFER"

	envDashboardStatsTTL = "HEROS_DASHBOARD_STATS_TTL"
	envDashboardWarmup   = "HEROS_DASHBOARD_WARMUP"

	envLocationTTL   = "HEROS_LOCATION_TTL"
	envAlertRadiusKm = "HEROS_ALERT_RADIUS_KM"
)

// EnvVarNames retourne la liste de toutes les variables d'environnement reconnues.
// Elle sert à vérifier que .env.example reste synchronisé avec le code.
func EnvVarNames() []string {
	return []string{
		envEnvironment, envLogLevel, envLogFormat, envAutoMigrate,
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
		envRateContactPerIP, envRateNewsletterPerIP, envRateNewsletterPerEmail, envRateEmailChangePerUser,
		envRateAvatarPerUser, envRateMFAVerifyPerUser, envRatePasswordCheckPerUser, envRateExportPerUser, envRateHeroLocationPerUser,
		envActivationTokenTTL, envPasswordResetTokenTTL, envNewsletterConfirmTTL, envEmailChangeConfirmTTL,
		envMFAModeratorRequired, envMFAIssuer,
		envCSRFSecret, envCryptoKeys, envCryptoActiveK,
		envMailDriver, envMailFromAddress, envMailFromName, envSMTPHost, envSMTPPort,
		envSMTPUsername, envSMTPPassword, envSMTPTLSMode, envSMTPTimeout, envMailWorkers,
		envMailQueueSize, envMailMaxAttempts, envMailRetryDelay, envMailSendTimeout, envMailQueue,
		envRabbitMQURL, envRabbitMQConfirmTimeout, envRabbitMQReconnectDelay,
		envOMDbAPIKey, envOMDbBaseURL, envOMDbTimeout,
		envContactRecipient, envAvatarDir, envAvatarMaxBytes, envAvatarMaxDimension,
		envPublicBaseURL, envActivationPath, envPasswordResetPath, envNewsletterConfirmPath,
		envNewsletterUnsubscribePath, envEmailChangeConfirmPath,
		envCleanupInterval, envCleanupTimeout, envSessionRetention, envTokenRetention, envCommandRetention,
		envEventsPollInterval, envEventsBatchSize, envEventsHeartbeat,
		envEventsMaxStreamsPerUser, envEventsStreamMaxAge, envEventsStreamBuffer,
		envDashboardStatsTTL, envDashboardWarmup,
		envLocationTTL, envAlertRadiusKm,
	}
}
