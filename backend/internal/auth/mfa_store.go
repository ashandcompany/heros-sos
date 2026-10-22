package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

const (
	tableUserMFA      = "user_mfa"
	tableRecoveryCode = "mfa_recovery_codes"
	columnTOTPSecret  = "totp_secret"
)

// errMFANotFound indique qu'un compte n'a jamais commencé l'activation de la 2FA.
var errMFANotFound = errors.New("auth : 2FA non configurée")

// MFA est l'état de la double authentification d'un compte.
type MFA struct {
	// Secret est le secret TOTP en clair (déchiffré à la lecture).
	Secret []byte
	// Confirmed vaut vrai une fois qu'un premier code a prouvé que l'application est configurée.
	Confirmed bool
	// LastUsedStep est le dernier pas de temps accepté : un code ne sert qu'une fois.
	LastUsedStep int64
}

type mfaRecord struct {
	UserID       uuid.UUID `gorm:"primaryKey"`
	TotpSecret   []byte
	ConfirmedAt  *time.Time
	LastUsedStep int64
	CreatedAt    time.Time
}

func (mfaRecord) TableName() string { return tableUserMFA }

// MFAStore est le stockage PostgreSQL de la double authentification. Chaque changement d'état se fait en
// une seule instruction : deux requêtes simultanées avec le même code ne peuvent pas réussir toutes les deux.
type MFAStore struct {
	db     *gorm.DB
	sealer user.Sealer
}

// NewMFAStore crée le stockage. Le secret TOTP est chiffré avec sealer, lié à sa table, sa colonne et son
// propriétaire : il ne peut pas être déplacé vers un autre compte.
func NewMFAStore(db *gorm.DB, sealer user.Sealer) *MFAStore {
	return &MFAStore{db: db, sealer: sealer}
}

func (s *MFAStore) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, s.db) }

func (s *MFAStore) aad(userID uuid.UUID) crypto.AAD {
	return crypto.AAD{Table: tableUserMFA, Column: columnTOTPSecret, RecordID: userID}
}

// startSetupSQL enregistre le secret d'une activation. Une activation non confirmée est remplacée ; une
// 2FA déjà confirmée n'est jamais écrasée (aucune ligne retournée).
const startSetupSQL = `
INSERT INTO user_mfa (user_id, totp_secret, created_at)
VALUES (?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    totp_secret = EXCLUDED.totp_secret,
    created_at = EXCLUDED.created_at
WHERE user_mfa.confirmed_at IS NULL
RETURNING user_id`

// StartSetup enregistre le secret d'une activation de la 2FA, qui reste en attente de confirmation. Il
// retourne false si la 2FA est déjà activée.
func (s *MFAStore) StartSetup(ctx context.Context, userID uuid.UUID, secret []byte, now time.Time) (bool, error) {
	sealed, err := s.sealer.Seal(secret, s.aad(userID))
	if err != nil {
		return false, fmt.Errorf("auth : chiffrement du secret TOTP : %w", err)
	}
	var result struct{ UserID uuid.UUID }
	query := s.conn(ctx).Raw(startSetupSQL, userID, sealed, now).Scan(&result)
	if query.Error != nil {
		return false, fmt.Errorf("auth : enregistrement du secret TOTP : %w", query.Error)
	}
	return query.RowsAffected > 0, nil
}

// Find retourne l'état de la 2FA d'un compte (errMFANotFound s'il n'a rien commencé).
func (s *MFAStore) Find(ctx context.Context, userID uuid.UUID) (*MFA, error) {
	var rec mfaRecord
	err := s.conn(ctx).Where("user_id = ?", userID).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errMFANotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth : lecture de la 2FA : %w", err)
	}
	secret, err := s.sealer.Open(rec.TotpSecret, s.aad(userID))
	if err != nil {
		return nil, fmt.Errorf("auth : déchiffrement du secret TOTP : %w", err)
	}
	return &MFA{Secret: secret, Confirmed: rec.ConfirmedAt != nil, LastUsedStep: rec.LastUsedStep}, nil
}

// Confirm active la 2FA d'un compte qui avait commencé l'activation : un premier code valide l'a prouvée. Il
// retourne false si elle était déjà confirmée (ou jamais commencée).
func (s *MFAStore) Confirm(ctx context.Context, userID uuid.UUID, step int64, now time.Time) (bool, error) {
	result := s.conn(ctx).Table(tableUserMFA).
		Where("user_id = ? AND confirmed_at IS NULL", userID).
		Updates(map[string]any{"confirmed_at": now, "last_used_step": step})
	if result.Error != nil {
		return false, fmt.Errorf("auth : confirmation de la 2FA : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ConsumeStep mémorise le pas d'un code accepté, de façon atomique : il retourne false si un code de ce pas
// ou d'un pas plus récent a déjà servi, y compris pendant cette même seconde dans une autre requête.
func (s *MFAStore) ConsumeStep(ctx context.Context, userID uuid.UUID, step int64) (bool, error) {
	result := s.conn(ctx).Table(tableUserMFA).
		Where("user_id = ? AND confirmed_at IS NOT NULL AND last_used_step < ?", userID, step).
		Update("last_used_step", step)
	if result.Error != nil {
		return false, fmt.Errorf("auth : mémorisation du code TOTP : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ReplaceRecoveryCodes remplace tous les codes de secours d'un compte par ces empreintes : les anciens, utilisés
// ou non, cessent de valoir.
func (s *MFAStore) ReplaceRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes [][]byte) error {
	if err := s.conn(ctx).Where("user_id = ?", userID).Delete(&recoveryCodeRecord{}).Error; err != nil {
		return fmt.Errorf("auth : suppression des codes de secours : %w", err)
	}
	records := make([]recoveryCodeRecord, len(hashes))
	for i, hash := range hashes {
		records[i] = recoveryCodeRecord{ID: ids.New(), UserID: userID, CodeHash: hash}
	}
	if len(records) == 0 {
		return nil
	}
	if err := s.conn(ctx).Create(&records).Error; err != nil {
		return fmt.Errorf("auth : enregistrement des codes de secours : %w", err)
	}
	return nil
}

type recoveryCodeRecord struct {
	ID       uuid.UUID `gorm:"primaryKey"`
	UserID   uuid.UUID
	CodeHash []byte
	UsedAt   *time.Time
}

func (recoveryCodeRecord) TableName() string { return tableRecoveryCode }

// UseRecoveryCode consomme un code de secours : il retourne false si ce code n'existe pas ou a déjà servi.
// C'est atomique, un code ne peut pas servir deux fois même sur deux requêtes simultanées.
func (s *MFAStore) UseRecoveryCode(ctx context.Context, userID uuid.UUID, hash []byte, now time.Time) (bool, error) {
	result := s.conn(ctx).Table(tableRecoveryCode).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hash).
		Update("used_at", now)
	if result.Error != nil {
		return false, fmt.Errorf("auth : utilisation d'un code de secours : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// RecoveryCodesRemaining compte les codes de secours qui n'ont pas servi.
func (s *MFAStore) RecoveryCodesRemaining(ctx context.Context, userID uuid.UUID) (int, error) {
	var remaining int64
	err := s.conn(ctx).Table(tableRecoveryCode).Where("user_id = ? AND used_at IS NULL", userID).Count(&remaining).Error
	if err != nil {
		return 0, fmt.Errorf("auth : comptage des codes de secours : %w", err)
	}
	return int(remaining), nil
}

// IsEnrolled indique si le compte a confirmé sa 2FA.
func (s *MFAStore) IsEnrolled(ctx context.Context, userID uuid.UUID) (bool, error) {
	var enrolled bool
	err := s.conn(ctx).Raw(`SELECT EXISTS (SELECT 1 FROM user_mfa WHERE user_id = ? AND confirmed_at IS NOT NULL)`, userID).Scan(&enrolled).Error
	if err != nil {
		return false, fmt.Errorf("auth : lecture de la 2FA : %w", err)
	}
	return enrolled, nil
}

// Reset supprime la 2FA d'un compte, secret et codes de secours compris (appareil perdu, par exemple). Il
// retourne false si le compte n'en avait pas. Réservé à l'exploitation (commande resetmfa) : aucune route de
// l'API ne l'appelle, pour qu'un compte piraté ne puisse pas désactiver sa propre protection.
func (s *MFAStore) Reset(ctx context.Context, userID uuid.UUID) (bool, error) {
	if err := s.conn(ctx).Where("user_id = ?", userID).Delete(&recoveryCodeRecord{}).Error; err != nil {
		return false, fmt.Errorf("auth : suppression des codes de secours : %w", err)
	}
	result := s.conn(ctx).Where("user_id = ?", userID).Delete(&mfaRecord{})
	if result.Error != nil {
		return false, fmt.Errorf("auth : suppression de la 2FA : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}
