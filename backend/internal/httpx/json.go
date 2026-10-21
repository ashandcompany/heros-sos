package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

// WriteJSON écrit v en JSON avec le code HTTP donné.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	if v == nil {
		return
	}
	// L'en-tête est déjà parti : une erreur d'encodage ne peut plus être signalée au client.
	_ = json.NewEncoder(w).Encode(v)
}

// WriteNoContent répond 204.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// DecodeJSON lit le corps de la requête dans dst. Il est strict : type de contenu JSON,
// taille bornée, aucun champ inconnu, une seule valeur JSON dans le corps.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return NewError(http.StatusUnsupportedMediaType, CodeUnsupportedMediaType,
			"Le corps de la requête doit être au format application/json.")
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return decodeError(err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return BadRequest("Le corps de la requête ne doit contenir qu'un seul objet JSON.")
	}
	return nil
}

func decodeError(err error) error {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return NewError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "Le corps de la requête est trop volumineux.")
	case errors.Is(err, io.EOF):
		return BadRequest("Le corps de la requête est vide.")
	default:
		// Le détail de l'erreur de décodage n'est pas renvoyé : il peut citer des valeurs saisies.
		return BadRequest("Le corps de la requête n'est pas un JSON valide.")
	}
}
