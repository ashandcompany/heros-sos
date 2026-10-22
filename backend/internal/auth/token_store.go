package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
)

const tableUserTokens = "user_tokens"

// TokenType distingue les usages d'un jeton envoyé par mail.
type TokenType string

// Usages d'un jeton : activation d'un compte, réinitialisation d'un mot de passe.
const (
	TokenActivation    TokenType = "activation"
	TokenPasswordReset TokenType = "password_reset"
)

type tokenRecord struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    uuid.UUID
	Type      string
	TokenHash []byte
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (tokenRecord) TableName() string { return tableUserTokens }

// TokenStore est le stockage des jetons à usage unique (activation, mot de passe oublié).
type TokenStore struct {
	db *gorm.DB
}

// NewTokenStore crée le stockage des jetons.
func NewTokenStore(db *gorm.DB) *TokenStore {
	return &TokenStore{db: db}
}

func (s *TokenStore) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, s.db) }

// Create enregistre l'empreinte d'un jeton.
func (s *TokenStore) Create(ctx context.Context, userID uuid.UUID, tokenType TokenType, hash []byte, expiresAt, now time.Time) error {
	rec := tokenRecord{
		ID: ids.New(), UserID: userID, Type: string(tokenType),
		TokenHash: hash, ExpiresAt: expiresAt, CreatedAt: now,
	}
	if err := s.conn(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("auth : création du jeton : %w", err)
	}
	return nil
}

// consumeSQL marque le jeton utilisé et retourne son propriétaire dans une seule instruction :
// deux requêtes simultanées avec le même jeton ne peuvent pas réussir toutes les deux.
const consumeSQL = `
UPDATE user_tokens SET used_at = ?
WHERE token_hash = ? AND type = ? AND used_at IS NULL AND expires_at > ?
RETURNING user_id`

// Consume utilise un jeton valide (existant, du bon type, non utilisé, non expiré) et retourne
// l'utilisateur concerné. Un jeton ne sert qu'une fois.
func (s *TokenStore) Consume(ctx context.Context, tokenType TokenType, hash []byte, now time.Time) (uuid.UUID, error) {
	var result struct{ UserID uuid.UUID }
	query := s.conn(ctx).Raw(consumeSQL, now, hash, string(tokenType), now).Scan(&result)
	if query.Error != nil {
		return uuid.Nil, fmt.Errorf("auth : utilisation du jeton : %w", query.Error)
	}
	if query.RowsAffected == 0 {
		return uuid.Nil, ErrTokenInvalid
	}
	return result.UserID, nil
}

// DeleteUnused supprime les jetons non utilisés d'un utilisateur pour un usage : une nouvelle
// demande invalide les liens précédents.
func (s *TokenStore) DeleteUnused(ctx context.Context, userID uuid.UUID, tokenType TokenType) error {
	err := s.conn(ctx).
		Where("user_id = ? AND type = ? AND used_at IS NULL", userID, string(tokenType)).
		Delete(&tokenRecord{}).Error
	if err != nil {
		return fmt.Errorf("auth : suppression des jetons : %w", err)
	}
	return nil
}

// DeleteStale supprime les jetons expirés ou utilisés avant la date donnée.
func (s *TokenStore) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	result := s.conn(ctx).
		Where("expires_at < ? OR used_at < ?", before, before).
		Delete(&tokenRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("auth : nettoyage des jetons : %w", result.Error)
	}
	return result.RowsAffected, nil
}
