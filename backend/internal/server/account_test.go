package server_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- Aides ------------------------------------------------------------------------------------

type accountView struct {
	ID, Role, Civility, FirstName, LastName, BirthDate, Email, PendingEmail, MobilePhone, AvatarURL string
	Address                                                                                         struct{ Street, Complement, PostalCode, City, Country string }
	NewsletterSubscribed                                                                            bool
	Hero                                                                                            *struct{ Alias, Bio string }
}

func (a *actor) account(t *testing.T) accountView {
	t.Helper()
	got := a.get("/me")
	if got.status != http.StatusOK {
		t.Fatalf("GET /me : %d %s", got.status, got.body)
	}
	var view accountView
	got.decode(t, &view)
	return view
}

// raw envoie un corps qui n'est pas du JSON (une image), comme le ferait un navigateur : l'en-tête
// Origin et le jeton CSRF accompagnent les écritures, et l'échange est vérifié contre le contrat.
func (a *actor) raw(method, path, contentType string, data []byte) response {
	a.e.t.Helper()
	var reader io.Reader
	if data != nil {
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, a.e.http.URL+apiBase+path, reader)
	if err != nil {
		a.e.t.Fatal(err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if method != http.MethodGet {
		request.Header.Set("Origin", frontOrigin)
		if a.csrf != "" {
			request.Header.Set("X-CSRF-Token", a.csrf)
		}
	}
	httpResponse, err := a.client.Do(request)
	if err != nil {
		a.e.t.Fatal(err)
	}
	defer func() { _ = httpResponse.Body.Close() }()
	payload, _ := io.ReadAll(httpResponse.Body)

	if a.e.spec != nil {
		a.e.spec.verify(a.e.t, request, data, httpResponse.StatusCode, httpResponse.Header, payload)
	}
	return response{status: httpResponse.StatusCode, header: httpResponse.Header, body: payload}
}

func (a *actor) patch(path string, body any) response {
	return a.request(http.MethodPatch, path, body)
}

func (a *actor) put(path string, body any) response {
	return a.request(http.MethodPut, path, body)
}

func solidImage(width, height int, fill color.NRGBA) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, fill)
		}
	}
	return img
}

func jpegBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, solidImage(width, height, color.NRGBA{R: 200, A: 255}), nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, solidImage(width, height, color.NRGBA{G: 200, A: 255})); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// pixelWebP est une image WebP sans perte de 1×1 pixel (le paquet standard ne sait pas en encoder).
const pixelWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func webpBytes(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(pixelWebP)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// withExif insère un segment EXIF (ici une position GPS fictive) juste après l'en-tête d'un JPEG.
func withExif(jpegData []byte, secret string) []byte {
	exif := append([]byte("Exif\x00\x00"), secret...)
	segment := append([]byte{0xFF, 0xE1, byte((len(exif) + 2) >> 8), byte(len(exif) + 2)}, exif...)
	return append(append(append([]byte{}, jpegData[:2]...), segment...), jpegData[2:]...)
}

func avatarFiles(t *testing.T, e *e2e) []string {
	t.Helper()
	entries, err := os.ReadDir(e.cfg.Avatar.Dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

// ---- Consultation du profil ------------------------------------------------------------------

func TestMyProfileIsTheSameRouteForEveryRole(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	hero := e.approvedHero(moderator, "tony@stark.io", "Iron Man")

	view := citizen.account(t)
	if view.Role != "citizen" || view.FirstName != "Steve" || view.LastName != "Rogers" || view.BirthDate != "1990-07-04" ||
		view.Email != "steve@avengers.example" || view.MobilePhone != "0612345678" || view.NewsletterSubscribed {
		t.Errorf("citoyen = %+v", view)
	}
	if view.Address.Street != "569 Leaman Place" || view.Address.City != "Brooklyn" || view.Address.Country != "US" || view.Address.Complement != "" {
		t.Errorf("adresse = %+v", view.Address)
	}
	if view.Hero != nil || view.PendingEmail != "" || view.AvatarURL != "" {
		t.Errorf("un citoyen sans photo ni demande n'a ni hero, ni pendingEmail, ni avatarUrl : %+v", view)
	}
	if body := citizen.get("/me").body; strings.Contains(string(body), `"hero"`) || strings.Contains(string(body), `"pendingEmail"`) {
		t.Errorf("les champs absents ne doivent pas être sérialisés : %s", body)
	}

	// Un super-héros est un compte comme les autres, avec son pseudo en plus.
	view = hero.account(t)
	if view.Role != "hero" || view.Hero == nil || view.Hero.Alias != "Iron Man" || view.Email != "tony@stark.io" {
		t.Errorf("héros = %+v", view)
	}

	// Un modérateur créé en ligne de commande n'a ni téléphone ni adresse : le profil se lit quand même.
	view = moderator.account(t)
	if view.Role != "moderator" || view.MobilePhone != "" || view.Address.Street != "" {
		t.Errorf("modérateur = %+v", view)
	}

	if got := e.newActor().get("/me"); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d", got.status)
	}
}

// ---- Coordonnées -----------------------------------------------------------------------------

func TestUpdateContact(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")

	// Le téléphone seul : normalisé, et l'adresse n'est pas touchée.
	got := citizen.patch("/me", map[string]any{"mobilePhone": "06 98 76 54 32"})
	if got.status != http.StatusOK {
		t.Fatalf("téléphone : %d %s", got.status, got.body)
	}
	var view accountView
	got.decode(t, &view)
	if view.MobilePhone != "0698765432" || view.Address.Street != "569 Leaman Place" {
		t.Errorf("après le téléphone : %+v", view)
	}

	// Une adresse avec complément.
	address := map[string]any{"street": "12 rue des Archives", "complement": "Interphone Dupont", "postalCode": "75003", "city": "Paris", "country": "FR"}
	got = citizen.patch("/me", map[string]any{"address": address})
	got.decode(t, &view)
	if got.status != http.StatusOK || view.Address.Street != "12 rue des Archives" || view.Address.Complement != "Interphone Dupont" || view.MobilePhone != "0698765432" {
		t.Fatalf("adresse : %d %s", got.status, got.body)
	}

	// Sans complément (ou avec null), celui-ci est conservé : le formulaire d'un héros n'en a pas.
	for name, patch := range map[string]map[string]any{
		"complement absent": {"street": "1 place Vendôme", "postalCode": "75001", "city": "Paris", "country": "FR"},
		"complement null":   {"street": "1 place Vendôme", "postalCode": "75001", "city": "Paris", "country": "FR", "complement": nil},
	} {
		got = citizen.patch("/me", map[string]any{"address": patch})
		got.decode(t, &view)
		if got.status != http.StatusOK || view.Address.Street != "1 place Vendôme" || view.Address.Complement != "Interphone Dupont" {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}

	// Une chaîne vide l'efface.
	got = citizen.patch("/me", map[string]any{"address": map[string]any{
		"street": "1 place Vendôme", "postalCode": "75001", "city": "Paris", "country": "FR", "complement": "",
	}})
	var cleared accountView // neuf : un champ absent du JSON ne remet pas à zéro la valeur d'un décodage précédent
	got.decode(t, &cleared)
	if got.status != http.StatusOK || cleared.Address.Complement != "" || cleared.Address.Street != "1 place Vendôme" {
		t.Errorf("complément effacé : %d %s", got.status, got.body)
	}
	if again := citizen.account(t); again.Address.Street != "1 place Vendôme" || again.Address.Complement != "" || again.MobilePhone != "0698765432" {
		t.Errorf("l'état doit persister : %+v", again)
	}
}

func TestUpdateContactAppliesTheRegistrationRules(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	before := citizen.account(t)

	// Toutes les erreurs sont retournées ensemble, avec le nom imbriqué des champs de l'adresse.
	got := citizen.patch("/me", map[string]any{
		"mobilePhone": "12345",
		"address":     map[string]any{"street": "", "postalCode": "7500", "city": strings.Repeat("v", 101), "country": "FR", "complement": strings.Repeat("c", 201)},
	})
	fields := got.fields()
	for field, code := range map[string]string{
		"mobilePhone": "invalid", "address.street": "required", "address.postalCode": "invalid",
		"address.city": "too_long", "address.complement": "too_long",
	} {
		if fields[field] != code {
			t.Errorf("%s = %q, attendu %q (champs : %v)", field, fields[field], code, fields)
		}
	}
	if got.status != http.StatusUnprocessableEntity || got.code() != "validation_failed" {
		t.Errorf("statut = %d %s", got.status, got.body)
	}
	if got := citizen.patch("/me", map[string]any{"address": map[string]any{"street": "1 rue X", "postalCode": "75001", "city": "Paris", "country": "france"}}); got.fields()["address.country"] != "invalid" {
		t.Errorf("pays invalide : %d %s", got.status, got.body)
	}

	// Un code postal français a cinq chiffres ; ailleurs, la forme est libre.
	if got := citizen.patch("/me", map[string]any{"address": map[string]any{"street": "1 rue X", "postalCode": "SW1A 1AA", "city": "Londres", "country": "GB"}}); got.status != http.StatusOK {
		t.Errorf("code postal britannique : %d %s", got.status, got.body)
	}
	if got := citizen.patch("/me", map[string]any{"address": map[string]any{"street": "1 rue X", "postalCode": "SW1A 1AA", "city": "Paris", "country": "FR"}}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("code postal britannique pour la France : %d", got.status)
	}

	// Rien à modifier, champ inconnu, rôle des champs protégés : aucun effet sur le compte.
	if got := citizen.patch("/me", map[string]any{}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("corps vide : %d %s", got.status, got.body)
	}
	for _, field := range []string{"email", "role", "firstName", "birthDate", "password"} {
		if got := citizen.patch("/me", map[string]any{field: "x"}); got.status != http.StatusBadRequest || got.code() != "bad_request" {
			t.Errorf("champ %q : %d %s", field, got.status, got.body)
		}
	}
	after := citizen.account(t)
	if after.Role != before.Role || after.Email != before.Email || after.FirstName != before.FirstName {
		t.Errorf("le compte a changé : %+v → %+v", before, after)
	}
}

func TestUpdateContactNeedsASessionAndTheCSRFToken(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	body := map[string]any{"mobilePhone": "0698765432"}

	if got := e.newActor().patch("/me", body); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d", got.status)
	}
	citizen.csrf = ""
	if got := citizen.patch("/me", body); got.status != http.StatusForbidden || got.code() != "csrf_invalid" {
		t.Errorf("sans jeton CSRF : %d %s", got.status, got.body)
	}
	if got := citizen.request(http.MethodPatch, "/me", body, func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }); got.status != http.StatusForbidden || got.code() != "origin_not_allowed" {
		t.Errorf("origine étrangère : %d %s", got.status, got.body)
	}
}

func TestHeroUpdatesContactWithoutLosingTheirAddressComplement(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	hero := e.approvedHero(moderator, "tony@stark.io", "Iron Man")
	if got := hero.patch("/me", map[string]any{"address": map[string]any{
		"street": "10880 Malibu Point", "complement": "Atelier", "postalCode": "90265", "city": "Malibu", "country": "US",
	}}); got.status != http.StatusOK {
		t.Fatal(got.status, string(got.body))
	}

	// Le formulaire du profil d'un héros n'envoie pas de complément.
	got := hero.patch("/me", map[string]any{"mobilePhone": "0611223344", "address": map[string]any{
		"street": "1 Stark Tower", "postalCode": "10001", "city": "New York", "country": "US",
	}})
	var view accountView
	got.decode(t, &view)
	if got.status != http.StatusOK || view.Address.Street != "1 Stark Tower" || view.Address.Complement != "Atelier" || view.Hero == nil || view.Hero.Alias != "Iron Man" {
		t.Errorf("héros : %d %s", got.status, got.body)
	}
}

// ---- Photo de profil -------------------------------------------------------------------------

func TestAvatarUploadServesAnImmutablePublicJPEG(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")

	got := citizen.raw(http.MethodPut, "/me/avatar", "image/png", pngBytes(t, 32, 32))
	if got.status != http.StatusOK {
		t.Fatalf("envoi : %d %s", got.status, got.body)
	}
	var uploaded struct{ AvatarURL string }
	got.decode(t, &uploaded)
	if !strings.HasPrefix(uploaded.AvatarURL, apiBase+"/avatars/") || !strings.HasSuffix(uploaded.AvatarURL, ".jpg") {
		t.Fatalf("avatarUrl = %q", uploaded.AvatarURL)
	}
	if view := citizen.account(t); view.AvatarURL != uploaded.AvatarURL {
		t.Errorf("GET /me : avatarUrl = %q, attendu %q", view.AvatarURL, uploaded.AvatarURL)
	}

	// La photo est publique (encyclopédie) : un visiteur anonyme la lit, avec un cache d'un an.
	photo := e.newActor().get(strings.TrimPrefix(uploaded.AvatarURL, apiBase))
	if photo.status != http.StatusOK || photo.header.Get("Content-Type") != "image/jpeg" ||
		!strings.Contains(photo.header.Get("Cache-Control"), "immutable") || photo.header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("lecture : %d %v", photo.status, photo.header)
	}
	if decoded, err := jpeg.Decode(bytes.NewReader(photo.body)); err != nil || decoded.Bounds().Dx() != 32 {
		t.Errorf("la photo servie n'est pas le JPEG attendu : %v", err)
	}
}

func TestAvatarReplacementDeletesThePreviousFile(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")
	anonymous := e.newActor()

	var first, second struct{ AvatarURL string }
	citizen.raw(http.MethodPut, "/me/avatar", "image/jpeg", jpegBytes(t, 16, 16)).decode(t, &first)
	citizen.raw(http.MethodPut, "/me/avatar", "image/webp", webpBytes(t)).decode(t, &second)
	if first.AvatarURL == "" || second.AvatarURL == "" || first.AvatarURL == second.AvatarURL {
		t.Fatalf("chaque envoi doit produire une nouvelle adresse : %q %q", first.AvatarURL, second.AvatarURL)
	}

	if got := anonymous.get(strings.TrimPrefix(first.AvatarURL, apiBase)); got.status != http.StatusNotFound || got.code() != "not_found" {
		t.Errorf("l'ancienne photo doit avoir disparu : %d %s", got.status, got.body)
	}
	if got := anonymous.get(strings.TrimPrefix(second.AvatarURL, apiBase)); got.status != http.StatusOK {
		t.Errorf("la nouvelle photo : %d", got.status)
	}
	if files := avatarFiles(t, e); len(files) != 1 {
		t.Errorf("fichiers sur le disque = %v, attendu un seul", files)
	}
}

func TestAvatarStripsMetadataFromTheStoredFile(t *testing.T) {
	e := newE2E(t, nil)
	hero := e.approvedHero(e.newModerator("fury@shield.example"), "tony@stark.io", "Iron Man")
	const secret = "GPS-48.8566N-2.3522E"

	var uploaded struct{ AvatarURL string }
	hero.raw(http.MethodPut, "/me/avatar", "image/jpeg", withExif(jpegBytes(t, 16, 16), secret)).decode(t, &uploaded)

	for _, name := range avatarFiles(t, e) {
		content, err := os.ReadFile(e.cfg.Avatar.Dir + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(content, []byte(secret)) || bytes.Contains(content, []byte("Exif")) {
			t.Errorf("le fichier stocké %s contient encore l'EXIF", name)
		}
	}
	if photo := e.newActor().get(strings.TrimPrefix(uploaded.AvatarURL, apiBase)); bytes.Contains(photo.body, []byte(secret)) {
		t.Error("la photo servie contient la métadonnée")
	}
}

func TestAvatarRejectsAnythingThatIsNotAnAcceptableImage(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_AVATAR_MAX_BYTES": "4096", "HEROS_AVATAR_MAX_DIMENSION": "64"})
	citizen := e.signUpAndLogin("steve@avengers.example")

	cases := map[string]struct {
		contentType string
		data        []byte
		status      int
		code        string
		field       string
	}{
		"JSON":              {"application/json", []byte(`{"a":1}`), http.StatusUnsupportedMediaType, "unsupported_media_type", ""},
		"sans type":         {"", jpegBytes(t, 8, 8), http.StatusUnsupportedMediaType, "unsupported_media_type", ""},
		"GIF":               {"image/gif", []byte("GIF89a"), http.StatusUnsupportedMediaType, "unsupported_media_type", ""},
		"SVG":               {"image/svg+xml", []byte("<svg/>"), http.StatusUnsupportedMediaType, "unsupported_media_type", ""},
		"texte déguisé":     {"image/png", []byte("<script>alert(1)</script>"), http.StatusUnprocessableEntity, "validation_failed", "invalid"},
		"JPEG tronqué":      {"image/jpeg", jpegBytes(t, 40, 40)[:30], http.StatusUnprocessableEntity, "validation_failed", "invalid"},
		"corps vide":        {"image/png", []byte{}, http.StatusUnprocessableEntity, "validation_failed", "required"},
		"trop de pixels":    {"image/png", pngBytes(t, 65, 8), http.StatusUnprocessableEntity, "validation_failed", "too_large"},
		"trop d'octets":     {"image/jpeg", append(jpegBytes(t, 8, 8), make([]byte, 5000)...), http.StatusRequestEntityTooLarge, "payload_too_large", ""},
		"type en majuscule": {"IMAGE/PNG; charset=binary", []byte("pas une image"), http.StatusUnprocessableEntity, "validation_failed", "invalid"},
	}
	for name, c := range cases {
		got := citizen.raw(http.MethodPut, "/me/avatar", c.contentType, c.data)
		if got.status != c.status || got.code() != c.code || (c.field != "" && got.fields()["avatar"] != c.field) {
			t.Errorf("%s : %d %s", name, got.status, got.body)
		}
	}
	if files := avatarFiles(t, e); len(files) != 0 {
		t.Errorf("un envoi refusé ne doit rien écrire : %v", files)
	}
	if view := citizen.account(t); view.AvatarURL != "" {
		t.Errorf("avatarUrl = %q", view.AvatarURL)
	}

	// Une image dans les limites passe.
	if got := citizen.raw(http.MethodPut, "/me/avatar", "image/png", pngBytes(t, 64, 64)); got.status != http.StatusOK {
		t.Errorf("image à la limite : %d %s", got.status, got.body)
	}
}

func TestAvatarNeedsASessionTheCSRFTokenAndIsRateLimited(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_AVATAR_PER_USER": "2/1h"})
	citizen := e.signUpAndLogin("steve@avengers.example")
	image := jpegBytes(t, 8, 8)

	if got := e.newActor().raw(http.MethodPut, "/me/avatar", "image/jpeg", image); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d", got.status)
	}
	csrf := citizen.csrf
	citizen.csrf = ""
	if got := citizen.raw(http.MethodPut, "/me/avatar", "image/jpeg", image); got.status != http.StatusForbidden || got.code() != "csrf_invalid" {
		t.Errorf("sans CSRF : %d %s", got.status, got.body)
	}
	citizen.csrf = csrf

	for range 2 {
		if got := citizen.raw(http.MethodPut, "/me/avatar", "image/jpeg", image); got.status != http.StatusOK {
			t.Fatalf("envoi : %d %s", got.status, got.body)
		}
	}
	third := citizen.raw(http.MethodPut, "/me/avatar", "image/jpeg", image)
	if third.status != http.StatusTooManyRequests || third.header.Get("Retry-After") == "" {
		t.Errorf("troisième envoi : %d %v", third.status, third.header)
	}
	// La limite est par utilisateur : un autre compte n'est pas pénalisé.
	other := e.signUpAndLogin("natasha@avengers.example")
	if got := other.raw(http.MethodPut, "/me/avatar", "image/jpeg", image); got.status != http.StatusOK {
		t.Errorf("autre utilisateur : %d %s", got.status, got.body)
	}
}

func TestAvatarRouteRefusesEveryNameThatIsNotOurs(t *testing.T) {
	e := newE2E(t, nil)
	e.spec = nil // ces chemins sont volontairement hors du contrat
	anonymous := e.newActor()

	if err := os.WriteFile(e.cfg.Avatar.Dir+"/../secret.jpg", []byte("hors du dossier"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/avatars/secret.jpg", "/avatars/..%2Fsecret.jpg", "/avatars/%2e%2e%2fsecret.jpg",
		"/avatars/0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10.jpg", "/avatars/0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10",
		"/avatars/" + strings.Repeat("a", 3000),
	} {
		got := anonymous.get(path)
		if got.status == http.StatusOK || bytes.Contains(got.body, []byte("hors du dossier")) {
			t.Errorf("GET %s : %d %s", path, got.status, got.body)
		}
	}
}

// ---- Changement d'adresse email --------------------------------------------------------------

type changeView struct {
	ID, UserID, FirstName, LastName, CurrentEmail, NewEmail, Status string
	ReviewedAt                                                      *string
}

func (a *actor) emailChanges(t *testing.T, query string) []changeView {
	t.Helper()
	got := a.get("/admin/email-change-requests" + query)
	if got.status != http.StatusOK {
		t.Fatalf("liste des changements d'adresse : %d %s", got.status, got.body)
	}
	var page struct{ Items []changeView }
	got.decode(t, &page)
	return page.Items
}

func (a *actor) requestEmailChange(t *testing.T, email string) response {
	t.Helper()
	return a.post("/me/email-change-requests", map[string]string{"email": email})
}

func TestEmailChangeNeedsAModeratorsApproval(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")

	got := citizen.requestEmailChange(t, "Steve.Rogers@Avengers.Example")
	var created struct{ NewEmail, Status string }
	got.decode(t, &created)
	if got.status != http.StatusCreated || created.NewEmail != "steve.rogers@avengers.example" || created.Status != "pending" {
		t.Fatalf("demande : %d %s", got.status, got.body)
	}

	// En attente : rien ne change pour la connexion, et le profil montre la demande.
	view := citizen.account(t)
	if view.Email != "steve@avengers.example" || view.PendingEmail != "steve.rogers@avengers.example" {
		t.Errorf("profil en attente : %+v", view)
	}
	if got := e.newActor().login("steve.rogers@avengers.example", goodPassword); got.status != http.StatusUnauthorized {
		t.Errorf("la nouvelle adresse ne connecte pas avant validation : %d", got.status)
	}

	// Le modérateur voit l'identité du demandeur, les deux adresses, et peut décider sans autre appel.
	pending := moderator.emailChanges(t, "?status=pending")
	if len(pending) != 1 || pending[0].UserID != citizen.userID() || pending[0].FirstName != "Steve" || pending[0].LastName != "Rogers" ||
		pending[0].CurrentEmail != "steve@avengers.example" || pending[0].NewEmail != "steve.rogers@avengers.example" || pending[0].Status != "pending" {
		t.Fatalf("liste = %+v", pending)
	}
	if got := moderator.post("/admin/email-change-requests/"+pending[0].ID+"/approve", nil); got.status != http.StatusNoContent {
		t.Fatalf("validation : %d %s", got.status, got.body)
	}

	// Après validation : la nouvelle adresse connecte, l'ancienne non, et la demande est close.
	view = citizen.account(t)
	if view.Email != "steve.rogers@avengers.example" || view.PendingEmail != "" {
		t.Errorf("profil après validation : %+v", view)
	}
	if got := e.newActor().login("steve.rogers@avengers.example", goodPassword); got.status != http.StatusOK {
		t.Errorf("connexion par la nouvelle adresse : %d %s", got.status, got.body)
	}
	if got := e.newActor().login("steve@avengers.example", goodPassword); got.status != http.StatusUnauthorized {
		t.Errorf("l'ancienne adresse ne connecte plus : %d", got.status)
	}
	if left := moderator.emailChanges(t, "?status=pending"); len(left) != 0 {
		t.Errorf("demandes encore en attente : %+v", left)
	}
	if done := moderator.emailChanges(t, "?status=approved"); len(done) != 1 || done[0].ReviewedAt == nil {
		t.Errorf("demandes validées : %+v", done)
	}

	// Une décision ne se prend qu'une fois.
	if got := moderator.post("/admin/email-change-requests/"+pending[0].ID+"/approve", nil); got.status != http.StatusConflict || got.code() != "email_change_not_pending" {
		t.Errorf("seconde validation : %d %s", got.status, got.body)
	}
	if got := moderator.post("/admin/email-change-requests/"+pending[0].ID+"/reject", nil); got.status != http.StatusConflict || got.code() != "email_change_not_pending" {
		t.Errorf("refus après validation : %d %s", got.status, got.body)
	}
}

func TestEmailChangeRejectionKeepsTheCurrentAddress(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.requestEmailChange(t, "autre@avengers.example")

	change := moderator.emailChanges(t, "")[0]
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/reject", nil); got.status != http.StatusNoContent {
		t.Fatalf("refus : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.Email != "steve@avengers.example" || view.PendingEmail != "" {
		t.Errorf("profil après refus : %+v", view)
	}
	if got := e.newActor().login("steve@avengers.example", goodPassword); got.status != http.StatusOK {
		t.Errorf("l'adresse actuelle doit toujours connecter : %d", got.status)
	}

	// Après un refus, une nouvelle demande est possible.
	if got := citizen.requestEmailChange(t, "autre@avengers.example"); got.status != http.StatusCreated {
		t.Errorf("nouvelle demande : %d %s", got.status, got.body)
	}
	if rejected := moderator.emailChanges(t, "?status=rejected"); len(rejected) != 1 || rejected[0].ReviewedAt == nil {
		t.Errorf("demandes refusées : %+v", rejected)
	}
}

// Un modérateur qui a ouvert la liste ne doit jamais valider une adresse qu'il n'a pas vue : une demande
// remplacée par son titulaire change d'identifiant.
func TestReplacingAPendingRequestInvalidatesTheOneAModeratorSaw(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")

	citizen.requestEmailChange(t, "premiere@avengers.example")
	seen := moderator.emailChanges(t, "?status=pending")[0]
	citizen.requestEmailChange(t, "seconde@avengers.example")

	if got := moderator.post("/admin/email-change-requests/"+seen.ID+"/approve", nil); got.status != http.StatusNotFound || got.code() != "email_change_not_found" {
		t.Fatalf("validation de la demande remplacée : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.Email != "steve@avengers.example" || view.PendingEmail != "seconde@avengers.example" {
		t.Errorf("profil : %+v", view)
	}

	current := moderator.emailChanges(t, "?status=pending")
	if len(current) != 1 || current[0].NewEmail != "seconde@avengers.example" || current[0].ID == seen.ID {
		t.Fatalf("une seule demande en attente, la nouvelle : %+v", current)
	}
	if got := moderator.post("/admin/email-change-requests/"+current[0].ID+"/approve", nil); got.status != http.StatusNoContent {
		t.Errorf("validation de la nouvelle demande : %d %s", got.status, got.body)
	}
	if e.count("email_change_requests", "user_id = ?", citizen.userID()) != 1 {
		t.Error("la demande remplacée ne doit pas laisser de ligne")
	}
}

func TestEmailChangeValidationAndConflicts(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	e.signUp("natasha@avengers.example")

	for email, want := range map[string]string{"": "required", "pas-un-email": "invalid", strings.Repeat("a", 250) + "@x.io": "too_long"} {
		got := citizen.requestEmailChange(t, email)
		if got.status != http.StatusUnprocessableEntity || got.fields()["email"] != want {
			t.Errorf("%q : %d %s", email, got.status, got.body)
		}
	}
	if got := citizen.requestEmailChange(t, "STEVE@avengers.example"); got.status != http.StatusConflict || got.code() != "email_unchanged" {
		t.Errorf("même adresse (casse ignorée) : %d %s", got.status, got.body)
	}
	if got := citizen.requestEmailChange(t, "Natasha@Avengers.Example"); got.status != http.StatusConflict || got.code() != "email_already_used" {
		t.Errorf("adresse d'un autre compte : %d %s", got.status, got.body)
	}
	if got := citizen.post("/me/email-change-requests", map[string]any{}); got.status != http.StatusUnprocessableEntity {
		t.Errorf("sans adresse : %d %s", got.status, got.body)
	}
	if count := e.count("email_change_requests", "user_id = ?", citizen.userID()); count != 0 {
		t.Errorf("une demande refusée ne laisse aucune ligne : %d", count)
	}

	// Le compte d'un modérateur se gère hors ligne : il ne passe pas par cette route.
	if got := moderator.requestEmailChange(t, "nouveau@shield.example"); got.status != http.StatusForbidden || got.code() != "forbidden" {
		t.Errorf("modérateur : %d %s", got.status, got.body)
	}
	if got := e.newActor().post("/me/email-change-requests", map[string]string{"email": "x@y.io"}); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d", got.status)
	}
	// Seul un modérateur décide.
	if got := citizen.get("/admin/email-change-requests"); got.status != http.StatusForbidden {
		t.Errorf("liste par un citoyen : %d", got.status)
	}
	if got := citizen.post("/admin/email-change-requests/"+citizen.userID()+"/approve", nil); got.status != http.StatusForbidden {
		t.Errorf("décision par un citoyen : %d", got.status)
	}
	for _, path := range []string{"/admin/email-change-requests/pas-un-uuid/approve", "/admin/email-change-requests/" + citizen.userID() + "/reject"} {
		if got := moderator.post(path, nil); got.status != http.StatusNotFound || got.code() != "email_change_not_found" {
			t.Errorf("%s : %d %s", path, got.status, got.body)
		}
	}
	if got := moderator.get("/admin/email-change-requests?status=bogus"); got.status != http.StatusUnprocessableEntity {
		t.Errorf("statut invalide : %d", got.status)
	}
}

func TestHeroCanChangeEmailToo(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	hero := e.approvedHero(moderator, "tony@stark.io", "Iron Man")

	if got := hero.requestEmailChange(t, "tony@stark-industries.io"); got.status != http.StatusCreated {
		t.Fatalf("demande : %d %s", got.status, got.body)
	}
	change := moderator.emailChanges(t, "?status=pending")[0]
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil); got.status != http.StatusNoContent {
		t.Fatalf("validation : %d %s", got.status, got.body)
	}
	// Le pseudo et l'email fonctionnent ensemble après le changement.
	if got := e.newActor().login("iron man", goodPassword); got.status != http.StatusOK {
		t.Errorf("connexion par pseudo : %d %s", got.status, got.body)
	}
	if view := hero.account(t); view.Email != "tony@stark-industries.io" || view.Hero == nil {
		t.Errorf("profil : %+v", view)
	}
}

// L'adresse peut être prise entre la demande et la validation : la base garantit l'unicité, et la demande
// reste alors à refuser.
func TestApprovalFailsCleanlyWhenTheAddressWasTakenInTheMeantime(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.requestEmailChange(t, "convoitee@avengers.example")

	e.signUp("convoitee@avengers.example") // quelqu'un s'inscrit avec cette adresse entre-temps

	change := moderator.emailChanges(t, "?status=pending")[0]
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil); got.status != http.StatusConflict || got.code() != "email_already_used" {
		t.Fatalf("validation : %d %s", got.status, got.body)
	}
	if view := citizen.account(t); view.Email != "steve@avengers.example" || view.PendingEmail != "convoitee@avengers.example" {
		t.Errorf("rien ne doit avoir changé : %+v", view)
	}
	if e.count("audit_logs", "action = 'user.email_change_approve'") != 0 {
		t.Error("une validation annulée ne doit pas être journalisée")
	}
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/reject", nil); got.status != http.StatusNoContent {
		t.Errorf("refus de la demande : %d %s", got.status, got.body)
	}
}

func TestConcurrentApprovalsOfTheSameRequestSucceedOnce(t *testing.T) {
	e := newE2E(t, nil)
	first := e.newModerator("fury@shield.example")
	second := e.newModerator("hill@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.requestEmailChange(t, "nouveau@avengers.example")
	change := first.emailChanges(t, "?status=pending")[0]

	moderators := []*actor{first, second, first, second, first, second}
	statuses := make(chan int, len(moderators))
	var wg sync.WaitGroup
	for _, moderator := range moderators {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil).status
		}()
	}
	wg.Wait()
	close(statuses)

	approved, conflicts := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusNoContent:
			approved++
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("statut inattendu : %d", status)
		}
	}
	if approved != 1 || conflicts != len(moderators)-1 {
		t.Fatalf("validations = %d, conflits = %d", approved, conflicts)
	}
	if e.count("audit_logs", "action = 'user.email_change_approve'") != 1 {
		t.Error("la décision doit être journalisée une seule fois")
	}
}

// Deux comptes qui demandent la même adresse : un seul peut l'obtenir.
func TestTwoUsersCannotBothObtainTheSameAddress(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	steve := e.signUpAndLogin("steve@avengers.example")
	natasha := e.signUpAndLogin("natasha@avengers.example")
	steve.requestEmailChange(t, "partagee@avengers.example")
	natasha.requestEmailChange(t, "partagee@avengers.example")

	changes := moderator.emailChanges(t, "?status=pending")
	if len(changes) != 2 {
		t.Fatalf("demandes = %+v", changes)
	}
	statuses := make(chan int, len(changes))
	var wg sync.WaitGroup
	for _, change := range changes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil).status
		}()
	}
	wg.Wait()
	close(statuses)

	approved, conflicts := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusNoContent:
			approved++
		case http.StatusConflict:
			conflicts++
		}
	}
	if approved != 1 || conflicts != 1 {
		t.Fatalf("validations = %d, conflits = %d", approved, conflicts)
	}
	if e.count("users", "email = 'partagee@avengers.example'") != 1 {
		t.Error("l'adresse ne doit appartenir qu'à un seul compte")
	}
}

func TestEmailChangeDecisionsAreAuditedWithoutPersonalData(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.requestEmailChange(t, "audit@avengers.example")
	change := moderator.emailChanges(t, "")[0]
	moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil)

	var details []string
	if err := e.db.Raw(`SELECT details::text FROM audit_logs WHERE action LIKE 'user.email_change_%' ORDER BY created_at`).Scan(&details).Error; err != nil {
		t.Fatal(err)
	}
	if len(details) != 2 {
		t.Fatalf("entrées d'audit = %v", details)
	}
	for _, entry := range details {
		if strings.Contains(entry, "@") || !strings.Contains(entry, change.ID) {
			t.Errorf("le journal doit référencer la demande sans adresse email : %s", entry)
		}
	}
}

func TestEmailChangeRequestsAreRateLimitedPerUser(t *testing.T) {
	e := newE2E(t, map[string]string{"HEROS_RATE_EMAIL_CHANGE_PER_USER": "2/1h"})
	citizen := e.signUpAndLogin("steve@avengers.example")

	for i := range 2 {
		if got := citizen.requestEmailChange(t, fmt.Sprintf("essai%d@avengers.example", i)); got.status != http.StatusCreated {
			t.Fatalf("demande %d : %d %s", i, got.status, got.body)
		}
	}
	third := citizen.requestEmailChange(t, "essai3@avengers.example")
	if third.status != http.StatusTooManyRequests || third.code() != "rate_limited" || third.header.Get("Retry-After") == "" {
		t.Errorf("troisième demande : %d %s", third.status, third.body)
	}
	// La fenêtre (1 h) passée, la limite se libère ; la session (2 h d'inactivité) est toujours valable.
	e.clock.Advance(time.Hour + time.Second)
	if got := citizen.requestEmailChange(t, "essai4@avengers.example"); got.status != http.StatusCreated {
		t.Errorf("après la fenêtre : %d %s", got.status, got.body)
	}
}

// ---- Newsletter du compte --------------------------------------------------------------------

func TestAccountNewsletterPreference(t *testing.T) {
	e := newE2E(t, nil)
	citizen := e.signUpAndLogin("steve@avengers.example")

	if got := citizen.put("/me/newsletter", map[string]any{"subscribed": true}); got.status != http.StatusNoContent {
		t.Fatalf("abonnement : %d %s", got.status, got.body)
	}
	if !citizen.account(t).NewsletterSubscribed {
		t.Error("GET /me doit refléter l'abonnement")
	}
	// Sans mail de confirmation : l'adresse du compte est déjà vérifiée. Le consentement est daté.
	if len(e.mailer.Messages()) != 1 { // le seul mail est celui de l'activation
		t.Errorf("mails envoyés = %d", len(e.mailer.Messages()))
	}
	var stored struct {
		Status, Source string
		Confirmed      bool
	}
	if err := e.db.Raw(`SELECT status, source, confirmed_at IS NOT NULL AS confirmed FROM newsletter_subscriptions WHERE email = 'steve@avengers.example'`).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "subscribed" || stored.Source != "account" || !stored.Confirmed {
		t.Errorf("abonnement stocké = %+v", stored)
	}

	// Idempotent dans les deux sens.
	for _, want := range []bool{true, false, false, true} {
		if got := citizen.put("/me/newsletter", map[string]any{"subscribed": want}); got.status != http.StatusNoContent {
			t.Fatalf("subscribed=%t : %d %s", want, got.status, got.body)
		}
		if citizen.account(t).NewsletterSubscribed != want {
			t.Errorf("après subscribed=%t, GET /me dit l'inverse", want)
		}
	}
	if e.count("newsletter_subscriptions", "email = 'steve@avengers.example'") != 1 {
		t.Error("une seule ligne par adresse")
	}

	for name, body := range map[string]any{"absent": map[string]any{}, "null": map[string]any{"subscribed": nil}} {
		if got := citizen.put("/me/newsletter", body); got.status != http.StatusUnprocessableEntity || got.fields()["subscribed"] != "required" {
			t.Errorf("subscribed %s : %d %s", name, got.status, got.body)
		}
	}
	if got := citizen.put("/me/newsletter", map[string]any{"subscribed": "oui"}); got.status != http.StatusBadRequest {
		t.Errorf("subscribed texte : %d", got.status)
	}
	if got := e.newActor().put("/me/newsletter", map[string]any{"subscribed": true}); got.status != http.StatusUnauthorized {
		t.Errorf("sans session : %d", got.status)
	}
}

func TestNewsletterSubscriptionFollowsTheApprovedEmailChange(t *testing.T) {
	e := newE2E(t, nil)
	moderator := e.newModerator("fury@shield.example")
	citizen := e.signUpAndLogin("steve@avengers.example")
	citizen.put("/me/newsletter", map[string]any{"subscribed": true})

	citizen.requestEmailChange(t, "nouveau@avengers.example")
	change := moderator.emailChanges(t, "?status=pending")[0]
	if got := moderator.post("/admin/email-change-requests/"+change.ID+"/approve", nil); got.status != http.StatusNoContent {
		t.Fatalf("validation : %d %s", got.status, got.body)
	}

	if !citizen.account(t).NewsletterSubscribed {
		t.Error("l'abonnement doit survivre au changement d'adresse")
	}
	if e.count("newsletter_subscriptions", "email = 'steve@avengers.example'") != 0 || e.count("newsletter_subscriptions", "email = 'nouveau@avengers.example' AND status = 'subscribed'") != 1 {
		t.Error("la newsletter doit être envoyée à la nouvelle adresse, plus à l'ancienne")
	}
}
