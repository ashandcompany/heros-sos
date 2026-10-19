// Package migrations embarque les migrations SQL du backend. Les fichiers sont nommés
// <horodatage>_<description>.up.sql / .down.sql (golang-migrate) : l'horodatage évite les
// conflits de numérotation entre développeurs travaillant en parallèle.
package migrations

import "embed"

// FS contient tous les fichiers de migration.
//
//go:embed *.sql
var FS embed.FS
