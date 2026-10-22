package incident

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor indique qu'un curseur de pagination n'est pas un curseur émis par l'API.
var ErrInvalidCursor = errors.New("incident : curseur invalide")

// Cursor est la position d'une page dans la liste des incidents (les plus récents d'abord) : le dernier
// incident déjà vu. Contrairement à un décalage (offset), il garde le même coût à la millionième page, et il
// ne saute ni ne répète d'incident quand de nouveaux incidents arrivent entre deux pages.
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// cursorSeparator sépare la date et l'identifiant dans la forme encodée. Ni l'un ni l'autre n'en contient.
const cursorSeparator = "|"

// CursorOf retourne le curseur qui suit l'incident donné.
func CursorOf(in Incident) Cursor { return Cursor{CreatedAt: in.CreatedAt, ID: in.ID} }

// String est la forme opaque envoyée aux clients : à renvoyer telle quelle dans le paramètre cursor.
func (c Cursor) String() string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + cursorSeparator + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor décode un curseur. Toute altération (base64 invalide, date ou identifiant illisible) est
// refusée : le curseur finit dans une requête SQL, il ne doit contenir que ce que String produit.
func ParseCursor(raw string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	createdAt, id, found := strings.Cut(string(decoded), cursorSeparator)
	if !found {
		return Cursor{}, ErrInvalidCursor
	}
	parsedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{CreatedAt: parsedAt, ID: parsedID}, nil
}
