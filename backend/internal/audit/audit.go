// Package audit tient le journal des actions sensibles (modération, réinitialisation de mot de
// passe). Une entrée dit qui a fait quoi sur quoi, jamais avec des données personnelles : les
// détails ne contiennent que des identifiants et des états.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

const (
	tableAuditLogs = "audit_logs"
	emptyDetails   = "{}"
)

// Actions enregistrées. Le préfixe indique le domaine.
const (
	ActionHeroRequested     = "hero.request"
	ActionHeroApproved      = "hero.approve"
	ActionHeroRejected      = "hero.reject"
	ActionUserDeactivated   = "user.deactivate"
	ActionUserReactivated   = "user.reactivate"
	ActionUserVerified      = "user.verify"
	ActionPasswordReset     = "auth.password_reset"
	ActionLogoutAllSessions = "auth.logout_all"

	ActionInterventionTypeCreated     = "intervention_type.create"
	ActionInterventionTypeDeactivated = "intervention_type.deactivate"
	ActionVillainCreated              = "villain.create"
	ActionIncidentDeclared            = "incident.declare"
)

// Entry décrit une action à journaliser.
type Entry struct {
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	// Details ne doit contenir aucune donnée personnelle.
	Details map[string]any
	IP      netip.Addr
}

// Recorder enregistre une entrée du journal.
type Recorder interface {
	Record(ctx context.Context, entry Entry) error
}

type record struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Details    string
	IPAddress  *string
	CreatedAt  time.Time
}

func (record) TableName() string { return tableAuditLogs }

// Store écrit le journal dans PostgreSQL. Il participe à la transaction du contexte : l'action
// et sa trace réussissent ou échouent ensemble.
type Store struct {
	db    *gorm.DB
	clock clock.Clock
}

// NewStore crée le journal d'audit.
func NewStore(db *gorm.DB, clk clock.Clock) *Store {
	return &Store{db: db, clock: clk}
}

// Record implémente Recorder.
func (s *Store) Record(ctx context.Context, entry Entry) error {
	details := emptyDetails
	if len(entry.Details) > 0 {
		encoded, err := json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("audit : encodage des détails : %w", err)
		}
		details = string(encoded)
	}

	rec := record{
		ID: ids.New(), ActorID: entry.ActorID, Action: entry.Action,
		EntityType: entry.EntityType, EntityID: entry.EntityID,
		Details: details, CreatedAt: s.clock.Now(),
	}
	if entry.IP.IsValid() {
		ip := entry.IP.String()
		rec.IPAddress = &ip
	}
	if err := database.Conn(ctx, s.db).Create(&rec).Error; err != nil {
		return fmt.Errorf("audit : enregistrement : %w", err)
	}
	return nil
}
