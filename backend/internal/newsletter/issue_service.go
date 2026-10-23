package newsletter

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/audit"
	"github.com/ashandcompany/heros-sos/backend/internal/blog"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/mail"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

const (
	issueEntityType = "newsletter"
	// queueRetryDelay et dispatchTimeout encadrent l'attente d'une place dans la file d'envoi.
	queueRetryDelay = 200 * time.Millisecond
	dispatchTimeout = 30 * time.Minute
)

// IssueInput est la saisie d'un numéro.
type IssueInput struct {
	Subject string
	Body    string
	// RecipientUserID désigne l'unique destinataire ; nil pour tous les abonnés.
	RecipientUserID *uuid.UUID
}

func (s *Service) cleanIssue(in IssueInput) (subject, body string, err error) {
	var fields validation.Collector
	subject = strings.TrimSpace(in.Subject)
	switch n := utf8.RuneCountInString(subject); {
	case n == 0:
		fields.Add("subject", validation.CodeRequired)
	case n > MaxSubjectLength:
		fields.Add("subject", validation.CodeTooLong)
	}
	if len(in.Body) > MaxBodyBytes {
		fields.Add("body", validation.CodeTooLong)
	} else {
		var textLength int
		body, textLength = blog.SanitizeHTML(in.Body)
		if textLength == 0 {
			fields.Add("body", validation.CodeRequired)
		}
	}
	return subject, body, fields.Err()
}

func (s *Service) recordIssue(ctx context.Context, actorID, id uuid.UUID, action string, ip netip.Addr) error {
	return s.Audit.Record(ctx, audit.Entry{
		ActorID: &actorID, Action: action, EntityType: issueEntityType, EntityID: &id, IP: ip,
	})
}

func unknownRecipient(err error) error {
	if errors.Is(err, ErrUnknownRecipient) {
		return &validation.Error{Fields: map[string]string{"recipient_user_id": validation.CodeInvalid}}
	}
	return err
}

// CreateIssue enregistre un brouillon.
func (s *Service) CreateIssue(ctx context.Context, actorID uuid.UUID, in IssueInput, ip netip.Addr) (*Issue, error) {
	subject, body, err := s.cleanIssue(in)
	if err != nil {
		return nil, err
	}
	issue := &Issue{
		ID: ids.New(), Subject: subject, BodyHTML: body, RecipientUserID: in.RecipientUserID,
		CreatedBy: &actorID, CreatedAt: s.Clock.Now(),
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Issues.Create(ctx, issue); err != nil {
			return err
		}
		return s.recordIssue(ctx, actorID, issue.ID, audit.ActionNewsletterCreated, ip)
	})
	if err != nil {
		return nil, unknownRecipient(err)
	}
	return issue, nil
}

// UpdateIssue remplace un brouillon ; un numéro envoyé retourne ErrIssueSent.
func (s *Service) UpdateIssue(ctx context.Context, actorID, id uuid.UUID, in IssueInput, ip netip.Addr) (*Issue, error) {
	subject, body, err := s.cleanIssue(in)
	if err != nil {
		return nil, err
	}
	issue, err := s.Issues.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	issue.Subject, issue.BodyHTML, issue.RecipientUserID = subject, body, in.RecipientUserID
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Issues.Update(ctx, issue); err != nil {
			return err
		}
		return s.recordIssue(ctx, actorID, id, audit.ActionNewsletterUpdated, ip)
	})
	if err != nil {
		return nil, unknownRecipient(err)
	}
	return issue, nil
}

// DeleteIssue supprime un brouillon ; un numéro envoyé retourne ErrIssueSent.
func (s *Service) DeleteIssue(ctx context.Context, actorID, id uuid.UUID, ip netip.Addr) error {
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Issues.Delete(ctx, id); err != nil {
			return err
		}
		return s.recordIssue(ctx, actorID, id, audit.ActionNewsletterDeleted, ip)
	})
}

// GetIssue retourne un numéro, corps compris.
func (s *Service) GetIssue(ctx context.Context, id uuid.UUID) (*Issue, error) {
	return s.Issues.Find(ctx, id)
}

// ListIssues retourne une page de numéros, du plus récent au plus ancien.
func (s *Service) ListIssues(ctx context.Context, limit, offset int) ([]Issue, int64, error) {
	return s.Issues.List(ctx, limit, offset)
}

// SendIssue envoie un brouillon à ses abonnés confirmés. Le numéro est daté « envoyé » avant tout
// mail, de façon atomique : il ne part qu'une fois, même si deux modérateurs cliquent ensemble. Sans
// aucun destinataire abonné, rien n'est marqué et ErrNoRecipient est retournée. Les mails sont ensuite
// confiés à la file d'envoi en arrière-plan ; le nombre de destinataires est retourné.
func (s *Service) SendIssue(ctx context.Context, actorID, id uuid.UUID, ip netip.Addr) (int, error) {
	issue, err := s.Issues.Find(ctx, id)
	if err != nil {
		return 0, err
	}
	if issue.Sent() {
		return 0, ErrIssueSent
	}
	recipients, err := s.Issues.Recipients(ctx, issue.RecipientUserID)
	if err != nil {
		return 0, err
	}
	if len(recipients) == 0 {
		return 0, ErrNoRecipient
	}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.Issues.MarkSent(ctx, id, s.Clock.Now()); err != nil {
			return err
		}
		return s.recordIssue(ctx, actorID, id, audit.ActionNewsletterSent, ip)
	})
	if err != nil {
		return 0, err
	}
	go s.dispatch(context.WithoutCancel(ctx), issue, recipients)
	return len(recipients), nil
}

// dispatch dépose un mail par destinataire dans la file d'envoi. Une file saturée est attendue ;
// l'échec d'un destinataire est journalisé (sans son adresse) et n'arrête pas les autres.
func (s *Service) dispatch(ctx context.Context, issue *Issue, recipients []Recipient) {
	ctx, cancel := context.WithTimeout(ctx, dispatchTimeout)
	defer cancel()
	failed := 0
	for _, rc := range recipients {
		link := s.UnsubscribeLink(rc.SubscriptionID)
		for {
			err := s.Mailer.SendNewsletter(ctx, rc.Email, issue.Subject, issue.BodyHTML, link)
			if errors.Is(err, mail.ErrQueueFull) && ctx.Err() == nil {
				time.Sleep(queueRetryDelay)
				continue
			}
			if err != nil {
				failed++
				s.Logger.ErrorContext(ctx, "envoi de la newsletter impossible", "newsletter_id", issue.ID,
					"subscription_id", rc.SubscriptionID, "error", err)
			}
			break
		}
	}
	s.Logger.InfoContext(ctx, "newsletter confiée à la file d'envoi", "newsletter_id", issue.ID,
		"recipients", len(recipients), "failed", failed)
}
