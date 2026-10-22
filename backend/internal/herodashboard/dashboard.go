// Package herodashboard calcule le tableau de bord professionnel d'un super-héros : les statistiques de
// toute la plateforme, les siennes et deux listes d'incidents. Tout est calculé à la demande en SQL
// (voir backend/tables/statistiques.md) ; le package ne possède aucune table.
package herodashboard

import "github.com/google/uuid"

// TypeCount est le nombre d'incidents d'un type du catalogue.
type TypeCount struct {
	Code  string `gorm:"column:code"`
	Label string `gorm:"column:label"`
	Count int64  `gorm:"column:count"`
}

// Global regroupe les statistiques de toute la plateforme.
type Global struct {
	// Interventions compte les interventions terminées : une par équipe, pas une par héros.
	Interventions int64 `gorm:"column:interventions"`
	// RescuedCitizens additionne les citoyens secourus des incidents réussis, une seule fois par incident.
	RescuedCitizens    int64 `gorm:"column:rescued_citizens"`
	VillainsIdentified int64 `gorm:"column:villains_identified"`
	VillainsImprisoned int64 `gorm:"column:villains_imprisoned"`
	// IncidentsByType liste tout le catalogue, y compris les types sans incident, du plus fréquent au moins fréquent.
	IncidentsByType []TypeCount `gorm:"-"`
}

// Enemy est un vilain que le héros a affronté.
type Enemy struct {
	ID    uuid.UUID `gorm:"column:id"`
	Alias string    `gorm:"column:alias"`
	// Encounters compte les incidents où le héros a affronté ce vilain.
	Encounters int64 `gorm:"column:encounters"`
}

// Personal regroupe les statistiques d'un héros.
type Personal struct {
	// Interventions compte les interventions terminées auxquelles le héros a participé (confirmé).
	Interventions           int64
	SuccessfulInterventions int64 `gorm:"column:successful_interventions"`
	FailedInterventions     int64 `gorm:"column:failed_interventions"`
	// AverageRating et RatingsCount ne comptent que les avis publiés.
	AverageRating float64 `gorm:"column:average_rating"`
	RatingsCount  int64   `gorm:"column:ratings_count"`
	// Videos compte tous les films liés au héros, PublishedVideos seulement ceux que la modération a approuvés.
	Videos          int64   `gorm:"column:videos"`
	PublishedVideos int64   `gorm:"column:published_videos"`
	MainEnemies     []Enemy `gorm:"-"`
}
