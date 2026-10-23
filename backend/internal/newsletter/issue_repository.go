package newsletter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	tableIssues         = "newsletters"
	issueRecipientKey   = "newsletters_recipient_user_id_fkey"
	issueSummaryColumns = "id, subject, '' AS body_html, recipient_user_id, created_by, created_at, sent_at"
)

// IssueRepository est le stockage des numéros de la newsletter.
type IssueRepository interface {
	Create(ctx context.Context, issue *Issue) error
	Find(ctx context.Context, id uuid.UUID) (*Issue, error)
	// List retourne les numéros du plus récent au plus ancien, sans leur corps.
	List(ctx context.Context, limit, offset int) ([]Issue, int64, error)
	// Update remplace le sujet, le corps et le destinataire d'un brouillon : ErrIssueSent si le
	// numéro est parti, ErrIssueNotFound s'il n'existe pas.
	Update(ctx context.Context, issue *Issue) error
	// Delete supprime un brouillon (mêmes erreurs que Update).
	Delete(ctx context.Context, id uuid.UUID) error
	// MarkSent date l'envoi d'un brouillon en une seule instruction : deux envois simultanés ne
	// peuvent pas réussir tous les deux (ErrIssueSent pour le second).
	MarkSent(ctx context.Context, id uuid.UUID, now time.Time) error
	// Recipients retourne les abonnés confirmés d'un numéro : tous, ou ceux du compte désigné.
	Recipients(ctx context.Context, userID *uuid.UUID) ([]Recipient, error)
}

type issueRecord struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	Subject         string
	BodyHTML        string `gorm:"column:body_html"`
	RecipientUserID *uuid.UUID
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
	SentAt          *time.Time
}

func (issueRecord) TableName() string { return tableIssues }

func (r issueRecord) toIssue() Issue { return Issue(r) }

type issueRepo struct{ db *gorm.DB }

// NewIssueRepository crée le dépôt des numéros, adossé à PostgreSQL.
func NewIssueRepository(db *gorm.DB) IssueRepository { return &issueRepo{db: db} }

func (r *issueRepo) conn(ctx context.Context) *gorm.DB { return database.Conn(ctx, r.db) }

func (r *issueRepo) Create(ctx context.Context, issue *Issue) error {
	rec := issueRecord{
		ID: issue.ID, Subject: issue.Subject, BodyHTML: issue.BodyHTML, RecipientUserID: issue.RecipientUserID,
		CreatedBy: issue.CreatedBy, CreatedAt: issue.CreatedAt,
	}
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsForeignKeyViolation(err, issueRecipientKey) {
			return ErrUnknownRecipient
		}
		return fmt.Errorf("newsletter : création du numéro : %w", err)
	}
	return nil
}

func (r *issueRepo) Find(ctx context.Context, id uuid.UUID) (*Issue, error) {
	var rec issueRecord
	err := r.conn(ctx).Where("id = ?", id).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrIssueNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("newsletter : lecture du numéro : %w", err)
	}
	issue := rec.toIssue()
	return &issue, nil
}

func (r *issueRepo) List(ctx context.Context, limit, offset int) ([]Issue, int64, error) {
	var total int64
	if err := r.conn(ctx).Table(tableIssues).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("newsletter : comptage des numéros : %w", err)
	}
	var records []issueRecord
	err := r.conn(ctx).Table(tableIssues).Select(issueSummaryColumns).
		Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&records).Error
	if err != nil {
		return nil, 0, fmt.Errorf("newsletter : liste des numéros : %w", err)
	}
	issues := make([]Issue, len(records))
	for i, rec := range records {
		issues[i] = rec.toIssue()
	}
	return issues, total, nil
}

func (r *issueRepo) Update(ctx context.Context, issue *Issue) error {
	result := r.conn(ctx).Table(tableIssues).Where("id = ? AND sent_at IS NULL", issue.ID).Updates(map[string]any{
		"subject": issue.Subject, "body_html": issue.BodyHTML, "recipient_user_id": issue.RecipientUserID,
	})
	if result.Error != nil {
		if database.IsForeignKeyViolation(result.Error, issueRecipientKey) {
			return ErrUnknownRecipient
		}
		return fmt.Errorf("newsletter : mise à jour du numéro : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return r.whyUntouched(ctx, issue.ID)
	}
	return nil
}

func (r *issueRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.conn(ctx).Exec(`DELETE FROM newsletters WHERE id = ? AND sent_at IS NULL`, id)
	if result.Error != nil {
		return fmt.Errorf("newsletter : suppression du numéro : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return r.whyUntouched(ctx, id)
	}
	return nil
}

func (r *issueRepo) MarkSent(ctx context.Context, id uuid.UUID, now time.Time) error {
	result := r.conn(ctx).Exec(`UPDATE newsletters SET sent_at = ? WHERE id = ? AND sent_at IS NULL`, now, id)
	if result.Error != nil {
		return fmt.Errorf("newsletter : marquage du numéro : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return r.whyUntouched(ctx, id)
	}
	return nil
}

// whyUntouched distingue un numéro inexistant d'un numéro déjà envoyé, quand une écriture réservée
// aux brouillons n'a touché aucune ligne.
func (r *issueRepo) whyUntouched(ctx context.Context, id uuid.UUID) error {
	var count int64
	if err := r.conn(ctx).Table(tableIssues).Where("id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("newsletter : lecture du numéro : %w", err)
	}
	if count == 0 {
		return ErrIssueNotFound
	}
	return ErrIssueSent
}

const recipientsSQL = `
SELECT id AS subscription_id, email FROM newsletter_subscriptions WHERE status = 'subscribed'`

// Le compte désigné est retrouvé par le rattachement de son abonnement ou par son adresse, comme
// pour sa préférence (IsSubscribed).
const accountRecipientsSQL = recipientsSQL + `
  AND (user_id = @user OR email = (SELECT email FROM users WHERE id = @user))`

func (r *issueRepo) Recipients(ctx context.Context, userID *uuid.UUID) ([]Recipient, error) {
	var rows []Recipient
	var err error
	if userID == nil {
		err = r.conn(ctx).Raw(recipientsSQL + ` ORDER BY created_at, id`).Scan(&rows).Error
	} else {
		err = r.conn(ctx).Raw(accountRecipientsSQL+` ORDER BY created_at, id`, map[string]any{"user": *userID}).Scan(&rows).Error
	}
	if err != nil {
		return nil, fmt.Errorf("newsletter : destinataires : %w", err)
	}
	return rows, nil
}
