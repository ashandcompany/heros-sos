package incident_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
)

func TestCursorSurvivesAnEncodeDecodeRoundTrip(t *testing.T) {
	// Précision de la microseconde : celle de timestamptz. Un curseur imprécis sauterait ou répéterait des lignes.
	original := incident.Cursor{CreatedAt: time.Date(2026, time.October, 22, 11, 30, 15, 123456000, time.UTC), ID: ids.New()}

	decoded, err := incident.ParseCursor(original.String())
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) || decoded.ID != original.ID {
		t.Errorf("curseur = %+v, attendu %+v", decoded, original)
	}
}

func TestCursorIsOpaqueAndURLSafe(t *testing.T) {
	encoded := incident.Cursor{CreatedAt: time.Now(), ID: ids.New()}.String()
	for _, char := range encoded {
		if char == '+' || char == '/' || char == '=' || char == '|' || char == ' ' {
			t.Fatalf("le curseur %q contient %q : il doit se glisser tel quel dans une URL", encoded, char)
		}
	}
}

func TestParseCursorRejectsAnythingTheAPIDidNotIssue(t *testing.T) {
	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	for name, raw := range map[string]string{
		"vide":                  "",
		"pas du base64":         "%%%",
		"sans séparateur":       encode("2026-10-22T11:30:15Z"),
		"date illisible":        encode("hier|" + ids.New().String()),
		"identifiant illisible": encode("2026-10-22T11:30:15Z|pas-un-uuid"),
		"requête injectée":      encode("2026-10-22T11:30:15Z|x'; DROP TABLE incidents; --"),
	} {
		if _, err := incident.ParseCursor(raw); !errors.Is(err, incident.ErrInvalidCursor) {
			t.Errorf("%s : erreur = %v, attendu ErrInvalidCursor", name, err)
		}
	}
}
