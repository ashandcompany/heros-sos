package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	statusReported = "reported"
	// verifyChunk borne le nombre d'incidents lus par requête lors de la comparaison.
	verifyChunk = 1000
)

// Mismatch décrit un incident dont l'état en base diffère de celui que raconte son journal.
type Mismatch struct {
	IncidentID uuid.UUID
	// Journal est le statut obtenu en rejouant les événements ; State, celui de la table incidents ("" si l'incident
	// n'existe plus).
	Journal, State string
}

// Verify rejoue tout le journal et compare, pour chaque incident qui y figure, le statut obtenu à celui de la
// table incidents. C'est la preuve que le journal et l'état racontent la même histoire : un chemin d'écriture qui
// changerait un statut sans écrire d'événement (ou l'inverse) y apparaît. Les incidents sans événement (les
// données importées avant le journal, par exemple) ne sont pas vérifiés.
func Verify(ctx context.Context, db *gorm.DB, store *Store, batch int) (checked int, mismatches []Mismatch, err error) {
	derived := map[uuid.UUID]string{}
	for after := int64(0); ; {
		events, err := store.Since(ctx, after, batch)
		if err != nil {
			return 0, nil, err
		}
		for _, event := range events {
			derived[event.IncidentID] = replayStatus(derived[event.IncidentID], event)
			after = event.Seq
		}
		if len(events) < batch {
			break
		}
	}

	incidentIDs := make([]uuid.UUID, 0, len(derived))
	for id := range derived {
		incidentIDs = append(incidentIDs, id)
	}
	for start := 0; start < len(incidentIDs); start += verifyChunk {
		chunk := incidentIDs[start:min(start+verifyChunk, len(incidentIDs))]
		var rows []struct {
			ID     uuid.UUID
			Status string
		}
		if err := database.Conn(ctx, db).Table("incidents").Select("id, status").Where("id IN ?", chunk).Scan(&rows).Error; err != nil {
			return 0, nil, fmt.Errorf("events : lecture des incidents : %w", err)
		}
		state := make(map[uuid.UUID]string, len(rows))
		for _, row := range rows {
			state[row.ID] = row.Status
		}
		for _, id := range chunk {
			if state[id] != derived[id] {
				mismatches = append(mismatches, Mismatch{IncidentID: id, Journal: derived[id], State: state[id]})
			}
		}
	}
	return len(derived), mismatches, nil
}

// replayStatus applique un événement au statut d'un incident : déclaré, il est « signalé » ; ensuite, il suit le
// statut de la dernière intervention (en cours, réussie, échouée) que les événements d'intervention portent.
func replayStatus(current string, event Recorded) string {
	switch {
	case event.Type == IncidentDeclared:
		return statusReported
	case event.AggregateType == AggregateIntervention:
		if status, ok := event.Payload["status"].(string); ok {
			return status
		}
	}
	return current
}

// RebuildCounters recalcule les compteurs de lecture (incident_stats, reporter_stats, villain_stats, hero_stats)
// depuis les tables, avec la même fonction que la migration qui les a créés. À lancer après une dérive.
func RebuildCounters(ctx context.Context, db *gorm.DB) error {
	if err := database.Conn(ctx, db).Exec(`SELECT rebuild_read_counters()`).Error; err != nil {
		return fmt.Errorf("events : recalcul des compteurs : %w", err)
	}
	return nil
}
