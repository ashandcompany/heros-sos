package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

// loader lit les variables d'environnement et accumule les erreurs de format, pour les
// signaler toutes d'un coup plutôt qu'une par exécution.
type loader struct {
	getenv func(string) string
	errs   []error
}

func (l *loader) fail(format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf(format, args...))
}

// fileSuffix désigne, pour toute variable NOM, la variable NOM_FILE qui contient le chemin d'un
// fichier dont le contenu est la valeur. C'est la convention des secrets Docker
// (/run/secrets/…) : le secret n'apparaît ni dans l'environnement du conteneur, ni dans
// `docker inspect`.
const fileSuffix = "_FILE"

// lookup lit la variable, ou le fichier désigné par NOM_FILE. Définir les deux est une erreur :
// il serait ambigu de savoir laquelle des deux valeurs l'emporte.
func (l *loader) lookup(name string) (string, bool) {
	direct := strings.TrimSpace(l.getenv(name))
	path := strings.TrimSpace(l.getenv(name + fileSuffix))

	switch {
	case path == "":
		return direct, direct != ""
	case direct != "":
		l.fail("%s et %s%s sont tous deux définis : n'en gardez qu'un", name, name, fileSuffix)
		return "", false
	}

	content, err := os.ReadFile(path)
	if err != nil {
		// Le message cite le chemin, jamais le contenu du fichier.
		l.fail("%s%s : lecture impossible de %q", name, fileSuffix, path)
		return "", false
	}
	value := strings.TrimSpace(string(content))
	return value, value != ""
}

func (l *loader) text(name, fallback string) string {
	if value, ok := l.lookup(name); ok {
		return value
	}
	return fallback
}

func (l *loader) duration(name string, fallback time.Duration) time.Duration {
	value, ok := l.lookup(name)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		l.fail("%s : durée invalide %q (exemple : 15m, 2h)", name, value)
		return fallback
	}
	return parsed
}

func (l *loader) positiveInt(name string, fallback int) int {
	value, ok := l.lookup(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		l.fail("%s : entier strictement positif attendu, reçu %q", name, value)
		return fallback
	}
	return parsed
}

// boundedInt lit un entier compris entre 1 et max. La borne protège les conversions vers un type plus étroit
// (uint8, uint32) : sans elle, une valeur trop grande serait tronquée en silence.
func (l *loader) boundedInt(name string, fallback, upper int) int {
	value, ok := l.lookup(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 || parsed > upper {
		l.fail("%s : entier entre 1 et %d attendu, reçu %q", name, upper, value)
		return fallback
	}
	return parsed
}

func (l *loader) boolean(name string, fallback bool) bool {
	value, ok := l.lookup(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		l.fail("%s : booléen attendu (true ou false), reçu %q", name, value)
		return fallback
	}
	return parsed
}

func (l *loader) list(name string) []string {
	value, ok := l.lookup(name)
	if !ok {
		return nil
	}
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// rateRule lit une règle au format "max/fenêtre", par exemple "20/15m".
func (l *loader) rateRule(name string, fallback RateRule) RateRule {
	value, ok := l.lookup(name)
	if !ok {
		return fallback
	}
	maxText, windowText, found := strings.Cut(value, "/")
	maxRequests, maxErr := strconv.Atoi(strings.TrimSpace(maxText))
	window, windowErr := time.ParseDuration(strings.TrimSpace(windowText))
	if !found || maxErr != nil || windowErr != nil || maxRequests <= 0 || window <= 0 {
		l.fail("%s : règle invalide %q (format attendu : max/fenêtre, exemple : 20/15m)", name, value)
		return fallback
	}
	return RateRule{Max: maxRequests, Window: window}
}

func (l *loader) environment() Environment {
	value := l.text(envEnvironment, string(EnvironmentDevelopment))
	switch env := Environment(value); env {
	case EnvironmentDevelopment, EnvironmentProduction, EnvironmentTest:
		return env
	default:
		l.fail("%s : valeur inconnue %q (development, production ou test)", envEnvironment, value)
		return EnvironmentDevelopment
	}
}

func (l *loader) logLevel() slog.Level {
	value, ok := l.lookup(envLogLevel)
	if !ok {
		return slog.LevelInfo
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		l.fail("%s : niveau inconnu %q (debug, info, warn ou error)", envLogLevel, value)
		return slog.LevelInfo
	}
	return level
}

func (l *loader) trustedProxies() []netip.Prefix {
	var prefixes []netip.Prefix
	for _, item := range l.list(envTrustedProxies) {
		if prefix, err := netip.ParsePrefix(item); err == nil {
			prefixes = append(prefixes, prefix)
			continue
		}
		addr, err := netip.ParseAddr(item)
		if err != nil {
			l.fail("%s : adresse ou réseau invalide %q", envTrustedProxies, item)
			continue
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes
}

func (l *loader) base64Secret(name string) []byte {
	value, ok := l.lookup(name)
	if !ok {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		l.fail("%s : base64 invalide", name)
		return nil
	}
	return decoded
}

// cryptoKeys lit la liste "identifiant:clé en base64" séparée par des virgules.
func (l *loader) cryptoKeys() map[uint16][]byte {
	keys := make(map[uint16][]byte)
	for _, item := range l.list(envCryptoKeys) {
		idText, keyText, found := strings.Cut(item, ":")
		id, idErr := strconv.ParseUint(strings.TrimSpace(idText), 10, 16)
		key, keyErr := base64.StdEncoding.DecodeString(strings.TrimSpace(keyText))
		if !found || idErr != nil || id == 0 || keyErr != nil {
			l.fail("%s : entrée invalide (format attendu : identifiant:base64, l'identifiant va de 1 à 65535)", envCryptoKeys)
			continue
		}
		if _, duplicate := keys[uint16(id)]; duplicate {
			l.fail("%s : identifiant de clé %d en double", envCryptoKeys, id)
			continue
		}
		keys[uint16(id)] = key
	}
	return keys
}

func (l *loader) activeKeyID(keys map[uint16][]byte) uint16 {
	value, ok := l.lookup(envCryptoActiveK)
	if !ok {
		return 0
	}
	id, err := strconv.ParseUint(value, 10, 16)
	if err != nil || id == 0 {
		l.fail("%s : identifiant de clé invalide %q", envCryptoActiveK, value)
		return 0
	}
	if _, exists := keys[uint16(id)]; !exists && len(keys) > 0 {
		l.fail("%s : la clé %d n'existe pas dans %s", envCryptoActiveK, id, envCryptoKeys)
	}
	return uint16(id)
}
