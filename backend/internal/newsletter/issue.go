package newsletter

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Bornes d'un numéro de la newsletter.
const (
	MaxSubjectLength = 200
	// MaxBodyBytes borne le HTML saisi, avant nettoyage.
	MaxBodyBytes = 200_000
)

var (
	// ErrIssueNotFound indique qu'aucun numéro ne porte cet identifiant.
	ErrIssueNotFound = errors.New("newsletter : numéro introuvable")
	// ErrIssueSent indique un numéro déjà envoyé : il est en lecture seule et ne part qu'une fois.
	ErrIssueSent = errors.New("newsletter : numéro déjà envoyé")
	// ErrNoRecipient indique qu'aucun abonné ne recevrait ce numéro.
	ErrNoRecipient = errors.New("newsletter : aucun destinataire abonné")
	// ErrUnknownRecipient indique que le destinataire choisi n'est pas un compte existant.
	ErrUnknownRecipient = errors.New("newsletter : destinataire inconnu")
)

// Issue est un numéro de la newsletter rédigé par la modération. Brouillon tant que SentAt est nil.
type Issue struct {
	ID       uuid.UUID
	Subject  string
	BodyHTML string
	// RecipientUserID est le seul destinataire du numéro ; nil pour tous les abonnés.
	RecipientUserID *uuid.UUID
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
	SentAt          *time.Time
}

// Sent indique si le numéro est parti (il n'est alors plus modifiable).
func (i Issue) Sent() bool { return i.SentAt != nil }

// Recipient est un abonné confirmé qui reçoit un numéro.
type Recipient struct {
	SubscriptionID uuid.UUID
	Email          string
}
