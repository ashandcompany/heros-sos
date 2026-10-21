package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
)

// NormalizeOrigin met une origine (schéma, hôte et port éventuel) sous sa forme canonique :
// minuscules et sans barre finale. Il retourne false si la valeur n'est pas une origine.
func NormalizeOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", false
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), true
}

// validate vérifie la cohérence de la configuration. Les contrôles les plus stricts ne
// s'appliquent qu'en production.
func (c Config) validate() error {
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	production := c.IsProduction()

	c.validateDatabase(fail, production)
	c.validateSecrets(fail)
	c.validateSessions(fail, production)
	c.validateMail(fail, production)
	if c.OMDb.BaseURL == "" {
		fail("%s est obligatoire", envOMDbBaseURL)
	} else if parsed, err := url.Parse(c.OMDb.BaseURL); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		fail("%s : URL HTTPS absolue attendue", envOMDbBaseURL)
	}
	if production && strings.TrimSpace(c.OMDb.APIKey) == "" {
		fail("%s est obligatoire en production", envOMDbAPIKey)
	}

	if c.Security.Password.MinLength > c.Security.Password.MaxLength {
		fail("%s ne peut pas dépasser %s", envPasswordMinLength, envPasswordMaxLength)
	}
	if c.Security.Lockout.BaseDelay > c.Security.Lockout.MaxDelay {
		fail("%s ne peut pas dépasser %s", envLockoutBaseDelay, envLockoutMaxDelay)
	}
	if production {
		if c.Security.Argon2.MemoryKiB < productionMinArgon2MemoryKiB || c.Security.Argon2.Time < productionMinArgon2Time {
			fail("argon2id trop faible pour la production (mémoire ≥ %d Kio, itérations ≥ %d)",
				productionMinArgon2MemoryKiB, productionMinArgon2Time)
		}
		if c.App.PublicBaseURL.Scheme != "https" {
			fail("%s doit utiliser https en production", envPublicBaseURL)
		}
	}

	for _, origin := range c.HTTP.AllowedOrigins {
		normalized, ok := NormalizeOrigin(origin)
		switch {
		case !ok:
			fail("%s : origine invalide %q (exemple : https://heros-sos.example)", envAllowedOrigins, origin)
		case production && !strings.HasPrefix(normalized, "https://"):
			fail("%s : l'origine %q doit utiliser https en production", envAllowedOrigins, origin)
		}
	}
	if production && len(c.HTTP.AllowedOrigins) == 0 {
		fail("%s est obligatoire en production", envAllowedOrigins)
	}

	return errors.Join(errs...)
}

func (c Config) validateDatabase(fail func(string, ...any), production bool) {
	if c.Database.URL == "" {
		fail("%s est obligatoire", envDatabaseURL)
	} else if parsed, err := url.Parse(c.Database.URL); err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		fail("%s : URL postgres:// attendue", envDatabaseURL)
	} else if production && parsed.Query().Get("sslmode") == "disable" {
		fail("%s : sslmode=disable est interdit en production", envDatabaseURL)
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		fail("%s ne peut pas dépasser %s", envDatabaseMaxIdleConns, envDatabaseMaxOpenConns)
	}
}

func (c Config) validateSecrets(fail func(string, ...any)) {
	if len(c.Crypto.Keys) == 0 {
		fail("%s est obligatoire (au moins une clé de %d octets)", envCryptoKeys, MasterKeyLength)
	}
	for id, key := range c.Crypto.Keys {
		if len(key) != MasterKeyLength {
			fail("%s : la clé %d fait %d octets, %d attendus", envCryptoKeys, id, len(key), MasterKeyLength)
		}
	}
	if len(c.Crypto.Keys) > 0 && c.Crypto.ActiveKeyID == 0 {
		fail("%s est obligatoire", envCryptoActiveK)
	}
	if len(c.Security.CSRFSecret) < minCSRFSecret {
		fail("%s est obligatoire (au moins %d octets en base64)", envCSRFSecret, minCSRFSecret)
	}
}

func (c Config) validateSessions(fail func(string, ...any), production bool) {
	policies := map[string]SessionPolicy{
		"membre": c.Session.Member, "se souvenir de moi": c.Session.RememberMe, "modérateur": c.Session.Moderator,
	}
	for name, policy := range policies {
		if policy.IdleTimeout > policy.MaxLifetime {
			fail("session %s : l'inactivité maximale (%s) dépasse la durée maximale (%s)",
				name, policy.IdleTimeout, policy.MaxLifetime)
		}
	}
	if c.Session.TouchPeriod > min(c.Session.Member.IdleTimeout, c.Session.Moderator.IdleTimeout) {
		fail("%s doit rester inférieur aux délais d'inactivité", envSessionTouchPeriod)
	}
	if production && !c.Session.CookieSecure {
		fail("%s doit valoir true en production", envCookieSecure)
	}
}

func (c Config) validateMail(fail func(string, ...any), production bool) {
	if _, err := mail.ParseAddress(c.Mail.FromAddress); err != nil {
		fail("%s : adresse invalide %q", envMailFromAddress, c.Mail.FromAddress)
	}
	if address, err := mail.ParseAddress(c.Mail.ContactRecipient); err != nil || address.Name != "" {
		fail("%s : adresse simple attendue, reçu %q", envContactRecipient, c.Mail.ContactRecipient)
	}
	if c.Mail.Driver == MailDriverSMTP && c.Mail.SMTP.Host == "" {
		fail("%s est obligatoire avec le driver smtp", envSMTPHost)
	}
	if production && c.Mail.Driver != MailDriverSMTP {
		fail("%s doit valoir smtp en production", envMailDriver)
	}
	if production && c.Mail.SMTP.TLSMode == TLSModeNone {
		fail("%s ne peut pas valoir none en production", envSMTPTLSMode)
	}
}
