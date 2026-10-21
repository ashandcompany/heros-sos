package avatar

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// immutableCache : le nom d'une photo change à chaque envoi, donc un nom donné ne change jamais de
// contenu et le navigateur peut le garder un an.
const immutableCache = "public, max-age=31536000, immutable"

// Handler sert les photos de profil : GET /avatars/{name}.
//
// La route est publique, comme les photos des héros dans l'encyclopédie. Pour un citoyen, la photo
// n'est accessible qu'à qui connaît son adresse, un UUID aléatoire que seule l'API donne à son
// propriétaire : c'est un secret de capacité, pas un contrôle d'accès par compte.
type Handler struct {
	store  *DiskStore
	logger *slog.Logger
}

// NewHandler crée le handler.
func NewHandler(store *DiskStore, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

// Serve répond avec la photo, ou 404.
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	file, err := h.store.Open(r.PathValue("name"))
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, httpx.NotFound("Photo introuvable."))
		return
	}
	if err != nil {
		httpx.Fail(w, r, h.logger, err)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		httpx.Fail(w, r, h.logger, err)
		return
	}
	header := w.Header()
	header.Set("Content-Type", StoredContentType)
	header.Set("Cache-Control", immutableCache)
	http.ServeContent(w, r, "", info.ModTime(), file)
}
