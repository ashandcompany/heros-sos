package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type articleView struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Excerpt     string  `json:"excerpt"`
	Content     string  `json:"content"`
	Status      string  `json:"status"`
	AuthorName  string  `json:"authorName"`
	PublishedAt *string `json:"publishedAt"`
}

type articlePage struct {
	Items []articleView `json:"items"`
	Total int           `json:"total"`
}

func articleBody(title, status, content string) map[string]string {
	return map[string]string{"title": title, "excerpt": "Un chapeau", "content": content, "status": status}
}

const goodContent = "<p>Un contenu <strong>suffisamment</strong> long pour passer.</p>"

func TestBlogDraftsAreModeratorOnlyAndPublishedArePublic(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	citizen := e.signUpAndLogin("citizen@example.org")
	anonymous := e.newActor()

	draft := moderator.post("/admin/articles", articleBody("Mon brouillon", "brouillon", goodContent))
	var draftArticle articleView
	draft.decode(t, &draftArticle)
	if draft.status != http.StatusCreated || draftArticle.Status != "brouillon" || draftArticle.PublishedAt != nil ||
		draftArticle.Slug != "mon-brouillon" {
		t.Fatalf("création du brouillon : %d %s", draft.status, draft.body)
	}
	published := moderator.post("/admin/articles", articleBody("Héros & vilains : l'été !", "publie", goodContent))
	var publishedArticle articleView
	published.decode(t, &publishedArticle)
	if published.status != http.StatusCreated || publishedArticle.PublishedAt == nil ||
		publishedArticle.Slug != "heros-vilains-l-ete" {
		t.Fatalf("création de l'article publié : %d %s", published.status, published.body)
	}

	// Public : seul l'article publié est visible, sous son slug uniquement.
	var list articlePage
	listed := anonymous.get("/articles")
	listed.decode(t, &list)
	if listed.status != http.StatusOK || list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != publishedArticle.ID {
		t.Fatalf("liste publique : %d %s", listed.status, listed.body)
	}
	if got := anonymous.get("/articles/heros-vilains-l-ete"); got.status != http.StatusOK {
		t.Fatalf("article publié : %d %s", got.status, got.body)
	}
	if got := anonymous.get("/articles/mon-brouillon"); got.status != http.StatusNotFound {
		t.Fatalf("un brouillon ne doit pas être public : %d %s", got.status, got.body)
	}

	// Back-office : modérateur uniquement.
	for _, path := range []string{"/admin/articles", "/admin/articles/" + draftArticle.ID} {
		if got := anonymous.get(path); got.status != http.StatusUnauthorized {
			t.Fatalf("anonyme sur %s : %d", path, got.status)
		}
		if got := citizen.get(path); got.status != http.StatusForbidden {
			t.Fatalf("citoyen sur %s : %d", path, got.status)
		}
	}
	if got := citizen.post("/admin/articles", articleBody("Interdit ici", "publie", goodContent)); got.status != http.StatusForbidden {
		t.Fatalf("création par un citoyen : %d", got.status)
	}

	var all articlePage
	moderator.get("/admin/articles").decode(t, &all)
	if all.Total != 2 {
		t.Fatalf("liste du modérateur : %+v", all)
	}
	var drafts articlePage
	moderator.get("/admin/articles?status=brouillon").decode(t, &drafts)
	if drafts.Total != 1 || drafts.Items[0].ID != draftArticle.ID {
		t.Fatalf("filtre brouillon : %+v", drafts)
	}
	if got := moderator.get("/admin/articles?status=nope"); got.status != http.StatusUnprocessableEntity {
		t.Fatalf("statut inconnu : %d", got.status)
	}
	if got := moderator.get("/admin/articles/" + draftArticle.ID); got.status != http.StatusOK {
		t.Fatalf("lecture d'un brouillon : %d %s", got.status, got.body)
	}
}

func TestBlogPublishingLifecycle(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	anonymous := e.newActor()

	var article articleView
	moderator.post("/admin/articles", articleBody("Titre initial", "brouillon", goodContent)).decode(t, &article)

	put := func(title, status string) (response, articleView) {
		got := moderator.request(http.MethodPut, "/admin/articles/"+article.ID, articleBody(title, status, goodContent))
		var updated articleView
		got.decode(t, &updated)
		return got, updated
	}

	// Brouillon renommé : le slug suit le titre.
	if got, updated := put("Titre modifié", "brouillon"); got.status != http.StatusOK || updated.Slug != "titre-modifie" {
		t.Fatalf("renommage du brouillon : %d %s", got.status, got.body)
	}
	got, published := put("Titre modifié", "publie")
	if got.status != http.StatusOK || published.PublishedAt == nil {
		t.Fatalf("publication : %d %s", got.status, got.body)
	}
	firstDate := *published.PublishedAt
	if got := anonymous.get("/articles/titre-modifie"); got.status != http.StatusOK {
		t.Fatalf("lecture publique : %d", got.status)
	}

	// Publié : le slug est figé, la date de première publication aussi.
	e.clock.Advance(48 * time.Hour)
	if got, updated := put("Encore un autre titre", "brouillon"); got.status != http.StatusOK || updated.Slug != "titre-modifie" {
		t.Fatalf("slug figé : %d %s", got.status, got.body)
	}
	if got := anonymous.get("/articles/titre-modifie"); got.status != http.StatusNotFound {
		t.Fatalf("dépublié : %d", got.status)
	}
	if _, republished := put("Encore un autre titre", "publie"); republished.PublishedAt == nil || *republished.PublishedAt != firstDate {
		t.Fatalf("la date de première publication doit être conservée : %+v", republished)
	}

	if got := moderator.request(http.MethodDelete, "/admin/articles/"+article.ID, nil); got.status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", got.status, got.body)
	}
	if got := anonymous.get("/articles/titre-modifie"); got.status != http.StatusNotFound {
		t.Fatalf("supprimé : %d", got.status)
	}
	if got := moderator.request(http.MethodDelete, "/admin/articles/"+article.ID, nil); got.status != http.StatusNotFound {
		t.Fatalf("suppression répétée : %d", got.status)
	}
	if n := e.count("audit_logs", "entity_type = 'article'"); n != 6 {
		t.Fatalf("entrées d'audit : %d", n)
	}
}

func TestBlogSlugsAreUniqueAndValidationIsEnforced(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")

	var first, second articleView
	moderator.post("/admin/articles", articleBody("Même titre", "publie", goodContent)).decode(t, &first)
	moderator.post("/admin/articles", articleBody("Même titre", "publie", goodContent)).decode(t, &second)
	if first.Slug != "meme-titre" || second.Slug != "meme-titre-2" {
		t.Fatalf("slugs : %q %q", first.Slug, second.Slug)
	}

	for name, body := range map[string]map[string]string{
		"titre trop court": articleBody("Abc", "publie", goodContent),
		"chapeau vide":     {"title": "Un titre", "excerpt": " ", "content": goodContent, "status": "publie"},
		"contenu court":    articleBody("Un titre", "publie", "<p>court</p>"),
		"statut inconnu":   articleBody("Un titre", "archive", goodContent),
		"chapeau long":     {"title": "Un titre", "excerpt": strings.Repeat("a", 301), "content": goodContent, "status": "publie"},
	} {
		if got := moderator.post("/admin/articles", body); got.status != http.StatusUnprocessableEntity {
			t.Fatalf("%s : %d %s", name, got.status, got.body)
		}
	}
}

func TestBlogContentIsSanitizedOnWrite(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("moderator@shield.example")
	anonymous := e.newActor()

	dirty := `<p onclick="x()">Un paragraphe assez long pour passer.</p><script>alert(1)</script>` +
		`<a href="javascript:alert(1)">piège</a><img src=x onerror=alert(1)>`
	var article articleView
	created := moderator.post("/admin/articles", articleBody("Contenu piégé", "publie", dirty))
	created.decode(t, &article)
	if created.status != http.StatusCreated {
		t.Fatalf("création : %d %s", created.status, created.body)
	}
	var read articleView
	anonymous.get("/articles/contenu-piege").decode(t, &read)
	for _, forbidden := range []string{"<script", "onclick", "javascript:", "<img", "onerror"} {
		if strings.Contains(read.Content, forbidden) {
			t.Fatalf("%q ne devrait pas survivre : %s", forbidden, read.Content)
		}
	}
	if !strings.Contains(read.Content, "Un paragraphe assez long") {
		t.Fatalf("le texte légitime doit rester : %s", read.Content)
	}
}
