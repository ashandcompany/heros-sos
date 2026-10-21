package avatar_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ashandcompany/heros-sos/backend/internal/avatar"
)

const maxDimension = 64

// pixelWebP est une image WebP sans perte de 1×1 pixel (le paquet ne sait pas encoder du WebP).
const pixelWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func solid(width, height int, fill color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, fill)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func decodeJPEG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("la sortie n'est pas un JPEG valide : %v", err)
	}
	return img
}

func TestNormalizeReencodesEveryAcceptedFormatAsJPEG(t *testing.T) {
	webp, err := base64.StdEncoding.DecodeString(pixelWebP)
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string][]byte{
		"jpeg": encodeJPEG(t, solid(40, 30, color.NRGBA{R: 200, A: 255})),
		"png":  encodePNG(t, solid(40, 30, color.NRGBA{G: 200, A: 255})),
		"webp": webp,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			output, err := avatar.Normalize(input, maxDimension)
			if err != nil {
				t.Fatal(err)
			}
			decodeJPEG(t, output)
			if !bytes.HasPrefix(output, []byte{0xFF, 0xD8}) {
				t.Errorf("la sortie ne commence pas par le marqueur JPEG : % x", output[:4])
			}
		})
	}
}

func TestNormalizeKeepsTheImageSizeAndFlattensTransparencyOnWhite(t *testing.T) {
	output, err := avatar.Normalize(encodePNG(t, solid(20, 10, color.NRGBA{})), maxDimension) // transparent
	if err != nil {
		t.Fatal(err)
	}
	img := decodeJPEG(t, output)
	if got := img.Bounds(); got.Dx() != 20 || got.Dy() != 10 {
		t.Errorf("dimensions = %v, attendu 20×10", got)
	}
	r, g, b, _ := img.At(5, 5).RGBA()
	const nearWhite = 0xF000
	if r < nearWhite || g < nearWhite || b < nearWhite {
		t.Errorf("un pixel transparent doit devenir blanc, obtenu %d %d %d", r>>8, g>>8, b>>8)
	}
}

// Une photo de téléphone porte sa position GPS dans l'EXIF : elle ne doit jamais être republiée.
func TestNormalizeStripsMetadataAndTrailingData(t *testing.T) {
	const secret = "GPS-48.8566N-2.3522E"

	original := encodeJPEG(t, solid(16, 16, color.NRGBA{B: 200, A: 255}))
	exif := append([]byte("Exif\x00\x00"), secret...)
	segment := append([]byte{0xFF, 0xE1, byte((len(exif) + 2) >> 8), byte(len(exif) + 2)}, exif...)
	withExif := append(append(append([]byte{}, original[:2]...), segment...), original[2:]...)
	if !bytes.Contains(withExif, []byte(secret)) {
		t.Fatal("le fichier de test doit contenir la métadonnée")
	}

	withTrailer := append(encodePNG(t, solid(16, 16, color.NRGBA{R: 255, A: 255})), "<script>alert(1)</script>"...)

	for name, input := range map[string][]byte{"exif": withExif, "données après l'image": withTrailer} {
		output, err := avatar.Normalize(input, maxDimension)
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		for _, leaked := range []string{secret, "Exif", "<script>"} {
			if bytes.Contains(output, []byte(leaked)) {
				t.Errorf("%s : la sortie contient encore %q", name, leaked)
			}
		}
	}
}

func TestNormalizeRejectsWhatIsNotAnAcceptableImage(t *testing.T) {
	valid := encodeJPEG(t, solid(16, 16, color.NRGBA{A: 255}))

	cases := map[string]struct {
		input []byte
		want  error
	}{
		"vide":         {nil, avatar.ErrInvalidImage},
		"texte":        {[]byte("ceci n'est pas une image"), avatar.ErrInvalidImage},
		"script":       {[]byte("<script>alert(1)</script>"), avatar.ErrInvalidImage},
		"SVG":          {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), avatar.ErrInvalidImage},
		"JPEG tronqué": {valid[:len(valid)/2], avatar.ErrInvalidImage},
		"trop large":   {encodePNG(t, solid(maxDimension+1, 1, color.White)), avatar.ErrTooLarge},
		"trop haut":    {encodeJPEG(t, solid(1, maxDimension+1, color.White)), avatar.ErrTooLarge},
	}
	for name, c := range cases {
		if _, err := avatar.Normalize(c.input, maxDimension); !errors.Is(err, c.want) {
			t.Errorf("%s : erreur = %v, attendu %v", name, err, c.want)
		}
	}

	if _, err := avatar.Normalize(encodePNG(t, solid(maxDimension, maxDimension, color.White)), maxDimension); err != nil {
		t.Errorf("une image exactement à la limite doit passer : %v", err)
	}
}

// Une image GIF est lisible par la bibliothèque standard mais n'est pas un format accepté.
func TestIsAcceptedContentType(t *testing.T) {
	for contentType, want := range map[string]bool{
		"image/jpeg":                true,
		"image/png":                 true,
		"image/webp":                true,
		"IMAGE/PNG":                 true,
		"image/png; charset=binary": true,
		"image/gif":                 false,
		"image/svg+xml":             false,
		"application/json":          false,
		"application/octet-stream":  false,
		"text/html":                 false,
		"":                          false,
		"image/":                    false,
	} {
		if got := avatar.IsAcceptedContentType(contentType); got != want {
			t.Errorf("IsAcceptedContentType(%q) = %t, attendu %t", contentType, got, want)
		}
	}
}

func newStore(t *testing.T) (*avatar.DiskStore, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "avatars", "sous-dossier") // créé par le magasin
	store, err := avatar.NewDiskStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func TestStoreSavesOpensAndDeletes(t *testing.T) {
	store, dir := newStore(t)

	first, err := store.Save([]byte("première"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Save([]byte("première"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("chaque envoi doit produire un nom différent, même pour un contenu identique")
	}
	if !avatar.IsValidName(first) {
		t.Errorf("nom produit invalide : %q", first)
	}

	file, err := store.Open(first)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(file)
	_ = file.Close()
	if string(content) != "première" {
		t.Errorf("contenu = %q", content)
	}
	if info, err := os.Stat(filepath.Join(dir, first)); err != nil || info.Mode().Perm()&0o007 != 0 {
		t.Errorf("le fichier ne doit pas être lisible par tous : %v %v", info, err)
	}

	if err := store.Delete(first); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(first); !errors.Is(err, avatar.ErrNotFound) {
		t.Errorf("Open après suppression = %v", err)
	}
	if err := store.Delete(first); err != nil {
		t.Errorf("supprimer deux fois doit réussir : %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, second)); err != nil {
		t.Errorf("l'autre photo ne doit pas être touchée : %v", err)
	}
}

func TestStoreRefusesNamesOutsideItsFormat(t *testing.T) {
	store, dir := newStore(t)
	outside := filepath.Join(filepath.Dir(dir), "secret.jpg")
	if err := os.WriteFile(outside, []byte("hors du dossier"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"", ".", "..", "../secret.jpg", "..%2Fsecret.jpg", "/etc/passwd",
		"secret.jpg", "0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10.png", // pas un UUID v4 / pas du JPEG
		"0b8c1c3e-6f1d-7c7e-9a52-3d2f7b9e4a10.jpg", // UUID v7 : jamais produit par Save
		strings.Repeat("a", 5000),
	} {
		if avatar.IsValidName(name) {
			t.Errorf("IsValidName(%q) = true", name)
		}
		if _, err := store.Open(name); !errors.Is(err, avatar.ErrNotFound) {
			t.Errorf("Open(%q) = %v", name, err)
		}
		if err := store.Delete(name); !errors.Is(err, avatar.ErrNotFound) {
			t.Errorf("Delete(%q) = %v", name, err)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("un fichier hors du dossier a été touché : %v", err)
	}
}

func TestStoreFailsWhenTheFolderCannotBeCreated(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "fichier")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := avatar.NewDiskStore(filepath.Join(blocker, "avatars")); err == nil {
		t.Error("créer un dossier sous un fichier doit échouer")
	}
}

func TestHandlerServesStoredPhotosWithImmutableCaching(t *testing.T) {
	store, _ := newStore(t)
	name, err := store.Save(encodeJPEG(t, solid(8, 8, color.White)))
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /avatars/{name}", avatar.NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil))).Serve)

	get := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}

	found := get("/avatars/" + name)
	if found.Code != http.StatusOK {
		t.Fatalf("statut = %d", found.Code)
	}
	if got := found.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := found.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") || strings.Contains(got, "no-store") {
		t.Errorf("Cache-Control = %q", got)
	}
	decodeJPEG(t, found.Body.Bytes())

	for _, path := range []string{
		"/avatars/0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10.jpg", // bien formé mais absent
		"/avatars/..%2f..%2fetc%2fpasswd",
		"/avatars/" + strings.TrimSuffix(name, ".jpg"),
	} {
		if got := get(path); got.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, attendu 404", path, got.Code)
		}
	}
}
