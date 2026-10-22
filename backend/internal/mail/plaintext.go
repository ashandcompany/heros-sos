package mail

import (
	"strings"

	"golang.org/x/net/html"
)

var blockTags = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "blockquote": true, "pre": true,
}

// plainText dérive la version texte d'un corps HTML : paragraphes séparés par une ligne vide, puces
// pour les listes, adresse des liens entre parenthèses.
func plainText(markup string) string {
	var out strings.Builder
	var hrefs []string
	tokenizer := html.NewTokenizer(strings.NewReader(markup))
	skipping := 0

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return tidyText(out.String())
		case html.TextToken:
			if skipping == 0 {
				text := string(tokenizer.Text())
				if strings.TrimSpace(text) == "" {
					out.WriteByte(' ')
					continue
				}
				if first := text[0]; first == ' ' || first == '\n' {
					out.WriteByte(' ')
				}
				out.WriteString(strings.Join(strings.Fields(text), " "))
				if last := text[len(text)-1]; last == ' ' || last == '\n' {
					out.WriteByte(' ')
				}
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tokenizer.TagName()
			switch tag := string(name); tag {
			case "script", "style":
				skipping++
			case "br", "hr":
				out.WriteString("\n")
			case "li":
				out.WriteString("\n- ")
			case "a":
				href := ""
				for hasAttr {
					var key, value []byte
					key, value, hasAttr = tokenizer.TagAttr()
					if string(key) == "href" {
						href = string(value)
					}
				}
				hrefs = append(hrefs, href)
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			switch tag := string(name); {
			case tag == "script" || tag == "style":
				if skipping > 0 {
					skipping--
				}
			case tag == "a" && len(hrefs) > 0:
				href := hrefs[len(hrefs)-1]
				hrefs = hrefs[:len(hrefs)-1]
				if href != "" {
					out.WriteString(" (" + href + ")")
				}
			case blockTags[tag]:
				out.WriteString("\n\n")
			}
		}
	}
}

// tidyText retire les espaces de fin de ligne et ramène les sauts de ligne multiples à une ligne vide.
func tidyText(text string) string {
	var out []string
	blank := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
