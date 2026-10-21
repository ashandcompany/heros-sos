// Package ids génère les identifiants des enregistrements.
package ids

import "github.com/google/uuid"

// New retourne un UUID version 7 : triable par date de création, ce qui garde les index
// B-tree compacts. L'identifiant est généré côté Go pour être connu avant l'insertion (il
// sert de contexte authentifié au chiffrement des champs).
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
