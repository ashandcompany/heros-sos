package testutil

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

// SeedReference crée de quoi rattacher des incidents : trois types d'incident, des citoyens actifs, des héros
// approuvés et des vilains. Les champs chiffrés des comptes reçoivent un octet quelconque : les tests qui s'en
// servent ne les lisent pas. Retourne les identifiants des héros.
func SeedReference(t testing.TB, db *gorm.DB, citizens, heroes, villains int) []uuid.UUID {
	t.Helper()
	exec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO incident_types (code, label, endangers_life) VALUES ('theft', 'Vol', false), ('murder', 'Meurtre', true), ('hostage', 'Prise d''otages', true)`)
	account := func(role string, i int) uuid.UUID {
		id := ids.New()
		exec(`INSERT INTO users (id, email, password_hash, role, status, civility, last_name, first_name, birth_date)
			VALUES (?, ?, 'x', ?, 'active', 'non-precise', '\x00', '\x00', '\x00')`, id, fmt.Sprintf("%s%d@example.test", role, i), role)
		return id
	}
	for i := range citizens {
		account("citizen", i)
	}
	heroIDs := make([]uuid.UUID, heroes)
	for i := range heroes {
		heroIDs[i] = account("hero", i)
		exec(`INSERT INTO hero_profiles (user_id, alias, verification_status) VALUES (?, ?, 'approved')`, heroIDs[i], fmt.Sprintf("hero-%d", i))
	}
	for i := range villains {
		exec(`INSERT INTO villains (id, alias) VALUES (?, ?)`, ids.New(), fmt.Sprintf("vilain-%d", i))
	}
	return heroIDs
}
