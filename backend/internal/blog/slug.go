package blog

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	maxSlugLength = 100
	fallbackSlug  = "article"
)

// Slugify transforme un titre en identifiant d'URL :
// « Héros & vilains : l'été ! » → « heros-vilains-l-ete ».
func Slugify(title string) string {
	stripped, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), title)
	if err != nil {
		stripped = title
	}
	var b strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(stripped) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
		} else {
			pendingDash = true
		}
	}
	slug := b.String()
	if len(slug) > maxSlugLength {
		slug = strings.TrimRight(slug[:maxSlugLength], "-")
	}
	if slug == "" {
		return fallbackSlug
	}
	return slug
}

// slugCandidate retourne le slug de base (attempt 0) ou suffixé (-2, -3…).
func slugCandidate(base string, attempt int) string {
	if attempt == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(attempt+1)
}
