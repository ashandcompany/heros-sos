package blog

import "testing"

func TestSanitizeHTML(t *testing.T) {
	const rel = ` rel="noopener noreferrer nofollow"`
	cases := []struct{ name, in, want string }{
		{"balises de l'éditeur", `<h2>Titre</h2><p>Un <strong>texte</strong> <em>net</em>.</p><ul><li><p>a</p></li></ul>`,
			`<h2>Titre</h2><p>Un <strong>texte</strong> <em>net</em>.</p><ul><li><p>a</p></li></ul>`},
		{"script supprimé avec son contenu", `<p>ok</p><script>alert(1)</script>`, `<p>ok</p>`},
		{"style supprimé avec son contenu", `<style>p{}</style><p>ok</p>`, `<p>ok</p>`},
		{"gestionnaire d'événement retiré", `<p onclick="x()" style="color:red">ok</p>`, `<p>ok</p>`},
		{"image retirée", `<img src=x onerror=alert(1)><p>ok</p>`, `<p>ok</p>`},
		{"balise inconnue : texte conservé", `<div>un <span>texte</span></div>`, `un texte`},
		{"lien http", `<a href="https://example.org/x?a=1&b=2" target="_blank">l</a>`,
			`<a href="https://example.org/x?a=1&amp;b=2"` + rel + `>l</a>`},
		{"lien mailto", `<a href="mailto:a@b.fr">m</a>`, `<a href="mailto:a@b.fr"` + rel + `>m</a>`},
		{"javascript: refusé", `<a href="javascript:alert(1)">x</a>`, `<a` + rel + `>x</a>`},
		{"javascript: avec majuscules et espaces", `<a href=" JaVaScRiPt:alert(1)">x</a>`, `<a` + rel + `>x</a>`},
		{"javascript: avec tabulation", "<a href=\"java\tscript:alert(1)\">x</a>", `<a` + rel + `>x</a>`},
		{"data: refusé", `<a href="data:text/html,<script>1</script>">x</a>`, `<a` + rel + `>x</a>`},
		{"rel source ignoré", `<a href="http://a.fr" rel="opener">x</a>`, `<a href="http://a.fr"` + rel + `>x</a>`},
		{"texte ré-échappé", `<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>`, `<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>`},
		{"commentaire retiré", `<p>a<!-- <script> -->b</p>`, `<p>ab</p>`},
		{"balises non fermées refermées", `<ul><li>a`, `<ul><li>a</li></ul>`},
		{"fermeture orpheline ignorée", `a</p></strong>`, `a`},
		{"br et hr", `<p>a<br/>b</p><hr>`, `<p>a<br>b</p><hr>`},
		{"script non fermé", `<p>ok</p><script>alert(1)`, `<p>ok</p>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := SanitizeHTML(tc.in); got != tc.want {
				t.Fatalf("SanitizeHTML(%q)\n got  %q\n want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeHTMLCountsVisibleText(t *testing.T) {
	if _, n := SanitizeHTML(`<p>Un  deux</p><script>caché caché</script><p> trois </p>`); n != len("Un deux")+len("trois") {
		t.Fatalf("longueur du texte visible : %d", n)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Héros & vilains : l'été !":   "heros-vilains-l-ete",
		"  Déjà--vu  ":                "deja-vu",
		"!!!":                         "article",
		"Comment bien déclarer 2026":  "comment-bien-declarer-2026",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, attendu %q", in, got, want)
		}
	}
	if got := Slugify(string(make([]rune, 0)) + "a-" + string(repeat('b', 200))); len(got) > maxSlugLength {
		t.Errorf("slug trop long : %d", len(got))
	}
}

func repeat(r rune, n int) []rune {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return out
}
