package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const tableSessions = "sessions"

var errSessionNotFound = errors.New("auth : session introuvable")

// Session est une session ouverte. Seule l'empreinte du jeton est connue du serveur.
type Session struct {
	TokenHash     []byte
	UserID        uuid.UUID
	CreatedAt     time.Time
	LastSeenAt    time.Time
	ExpiresAt     time.Time
	RememberMe    bool
	MFAVerifiedAt *time.Time
	RevokedAt     *time.Time
	IPAddress     string
	DeviceLabel   string
}

type sessionRecord struct {
	TokenHash     []byte `gorm:"primaryKey"`
	UserID        uuid.UUID
	CreatedAt     time.Time
	LastSeenAt    time.Time
	ExpiresAt     time.Time
	RememberMe    bool
	MFAVerifiedAt *time.Time
	RevokedAt     *time.Time
	IPAddress     *string
	DeviceLabel   string
}

func (sessionRecord) TableName() string { return tableSessions }

// sessionView est une session jointe à l'état courant de son propriétaire : le rôle et le
// statut sont lus à chaque requête, donc une désactivation ou un changement de rôle prend effet
// immédiatement, sans attendre l'expiration de la session.
type sessionView struct {
	Session    sessionRecord `gorm:"embedded"`
	Role       string
	UserStatus string
}

// SessionStore est le stockage PostgreSQL des sessions.
type SessionStore struct {
	db *gorm.DB
}

// NewSessionStore crée le stockage des sessions.
func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, s.db) }

// Create enregistre une session.
func (s *SessionStore) Create(ctx context.Context, session *Session) error {
	rec := sessionRecord{
		TokenHash:     session.TokenHash,
		UserID:        session.UserID,
		CreatedAt:     session.CreatedAt,
		LastSeenAt:    session.LastSeenAt,
		ExpiresAt:     session.ExpiresAt,
		RememberMe:    session.RememberMe,
		MFAVerifiedAt: session.MFAVerifiedAt,
		DeviceLabel:   session.DeviceLabel,
	}
	if session.IPAddress != "" {
		rec.IPAddress = &session.IPAddress
	}
	if err := s.conn(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("auth : création de la session : %w", err)
	}
	return nil
}

// find retourne la session et l'état de son propriétaire, quelle que soit la validité de la
// session : c'est au service d'en décider.
func (s *SessionStore) find(ctx context.Context, tokenHash []byte) (*sessionView, error) {
	var view sessionView
	err := s.conn(ctx).Table(tableSessions).
		Select("sessions.*, users.role AS role, users.status AS user_status").
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("sessions.token_hash = ?", tokenHash).
		Take(&view).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth : lecture de la session : %w", err)
	}
	return &view, nil
}

func (s *SessionStore) touch(ctx context.Context, tokenHash []byte, now time.Time) error {
	err := s.conn(ctx).Table(tableSessions).
		Where("token_hash = ?", tokenHash).
		Update("last_seen_at", now).Error
	if err != nil {
		return fmt.Errorf("auth : mise à jour de la session : %w", err)
	}
	return nil
}

// revoke invalide une session ; il retourne false si elle n'existait pas ou l'était déjà.
func (s *SessionStore) revoke(ctx context.Context, tokenHash []byte, now time.Time) (bool, error) {
	result := s.conn(ctx).Table(tableSessions).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", now)
	if result.Error != nil {
		return false, fmt.Errorf("auth : révocation de la session : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// RevokeAllForUser invalide toutes les sessions ouvertes d'un utilisateur.
func (s *SessionStore) RevokeAllForUser(ctx context.Context, userID uuid.UUID, now time.Time) (int64, error) {
	result := s.conn(ctx).Table(tableSessions).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)
	if result.Error != nil {
		return 0, fmt.Errorf("auth : révocation des sessions : %w", result.Error)
	}
	return result.RowsAffected, nil
}

// DeleteStale supprime les sessions expirées ou révoquées avant la date donnée.
func (s *SessionStore) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	result := s.conn(ctx).
		Where("expires_at < ? OR revoked_at < ?", before, before).
		Delete(&sessionRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("auth : nettoyage des sessions : %w", result.Error)
	}
	return result.RowsAffected, nil
}
