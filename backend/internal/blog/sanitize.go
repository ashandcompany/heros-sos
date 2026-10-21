package blog

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

var allowedTags = map[string]bool{
	"p": true, "br": true, "h2": true, "h3": true, "strong": true, "em": true, "s": true,
	"ul": true, "ol": true, "li": true, "blockquote": true, "hr": true, "a": true,
}

var voidTags = map[string]bool{"br": true, "hr": true}

// Balises supprimées avec leur contenu (sinon leur code deviendrait du texte visible).
var droppedWithContent = map[string]bool{
	"script": true, "style": true, "iframe": true, "noscript": true, "template": true,
	"object": true, "embed": true, "textarea": true, "title": true, "svg": true, "math": true,
}

var allowedSchemes = map[string]bool{"http": true, "https": true, "mailto": true}

const linkRel = "noopener noreferrer nofollow"

// SanitizeHTML ne garde que les balises produites par l'éditeur du back-office (liste blanche
// identique à frontend/src/lib/sanitize.ts). Tout le reste est supprimé : balises inconnues
// (leur texte est conservé), attributs, commentaires, scripts. Les liens n'acceptent que les
// schémas http, https et mailto et reçoivent toujours un rel restrictif. Le résultat est un
// HTML bien formé ; la seconde valeur est la longueur du texte visible.
func SanitizeHTML(input string) (clean string, textLength int) {
	var out strings.Builder
	var open []string
	skipDepth := 0
	z := html.NewTokenizer(strings.NewReader(input))

	closeTo := func(name string) {
		for i := len(open) - 1; i >= 0; i-- {
			if open[i] != name {
				continue
			}
			for j := len(open) - 1; j >= i; j-- {
				out.WriteString("</" + open[j] + ">")
			}
			open = open[:i]
			return
		}
	}

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			for j := len(open) - 1; j >= 0; j-- {
				out.WriteString("</" + open[j] + ">")
			}
			return out.String(), textLength
		case html.TextToken:
			if skipDepth > 0 {
				continue
			}
			text := string(z.Text())
			out.WriteString(html.EscapeString(text))
			textLength += utf8.RuneCountInString(strings.Join(strings.Fields(text), " "))
		case html.StartTagToken, html.SelfClosingTagToken:
			nameBytes, hasAttr := z.TagName()
			name := string(nameBytes)
			if droppedWithContent[name] {
				if tt == html.StartTagToken {
					skipDepth++
				}
				continue
			}
			if skipDepth > 0 || !allowedTags[name] {
				continue
			}
			if voidTags[name] {
				out.WriteString("<" + name + ">")
				continue
			}
			out.WriteString("<" + name)
			if name == "a" {
				if href := safeHref(z, hasAttr); href != "" {
					out.WriteString(` href="` + html.EscapeString(href) + `"`)
				}
				out.WriteString(` rel="` + linkRel + `"`)
			}
			out.WriteString(">")
			if tt == html.SelfClosingTagToken {
				out.WriteString("</" + name + ">")
				continue
			}
			open = append(open, name)
		case html.EndTagToken:
			nameBytes, _ := z.TagName()
			name := string(nameBytes)
			if droppedWithContent[name] {
				if skipDepth > 0 {
					skipDepth--
				}
				continue
			}
			if skipDepth == 0 && allowedTags[name] && !voidTags[name] {
				closeTo(name)
			}
		}
	}
}

// safeHref lit l'attribut href de la balise courante et ne le retourne que s'il est autorisé.
func safeHref(z *html.Tokenizer, hasAttr bool) string {
	for hasAttr {
		var key, value []byte
		key, value, hasAttr = z.TagAttr()
		if string(key) != "href" {
			continue
		}
		raw := strings.TrimSpace(string(value))
		parsed, err := url.Parse(raw)
		if err != nil || !allowedSchemes[strings.ToLower(parsed.Scheme)] {
			return ""
		}
		return raw
	}
	return ""
}
