// Package events tient le journal des événements métier des incidents et des interventions : chaque action y
// laisse une ligne, écrite dans la même transaction que la mise à jour de l'état. Les tables d'état (incidents,
// interventions…) restent la source des lectures ; le journal dit comment on y est arrivé, et alimente la
// diffusion en temps réel (flux SSE) et la journalisation (ELK).
//
// Seuls des identifiants et des états entrent dans un événement : ni adresse, ni texte libre. Il peut donc
// quitter la base tel quel.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Types d'agrégats : l'entité dont une suite d'événements raconte l'histoire.
const (
	AggregateIncident     = "incident"
	AggregateIntervention = "intervention"
)

// Types d'événements. Le préfixe est celui de l'agrégat.
const (
	IncidentDeclared                = "incident.declared"
	IncidentModerated               = "incident.moderated"
	IncidentCorrected               = "incident.corrected"
	IncidentPriorityChanged         = "incident.priority_changed"
	InterventionStarted             = "intervention.started"
	InterventionJoined              = "intervention.joined"
	InterventionInvited             = "intervention.invited"
	InterventionInvitationConfirmed = "intervention.invitation_confirmed"
	InterventionInvitationDeclined  = "intervention.invitation_declined"
	InterventionOutcomeProposed     = "intervention.outcome_proposed"
	InterventionOutcomeValidated    = "intervention.outcome_validated"
	InterventionOutcomeDisputed     = "intervention.outcome_disputed"
	InterventionClosed              = "intervention.closed"
	InterventionArbitrated          = "intervention.arbitrated"
)

var (
	// ErrNoTransaction indique qu'un événement a été écrit hors transaction : il pourrait survivre à un état
	// annulé, ou l'inverse. C'est une erreur de programmation, pas de saisie.
	ErrNoTransaction = errors.New("events : écriture hors transaction")
	// ErrVersionConflict indique que l'agrégat a changé depuis la version que l'appelant avait lue.
	ErrVersionConflict = errors.New("events : l'agrégat a changé depuis la version attendue")
)

// Event est un événement à enregistrer.
type Event struct {
	Type          string
	AggregateType string
	AggregateID   uuid.UUID
	// IncidentID est l'incident concerné (l'agrégat lui-même pour un incident).
	IncidentID uuid.UUID
	ActorID    *uuid.UUID
	// Payload ne contient que des identifiants et des états.
	Payload map[string]any
	// ExpectedVersion, quand elle est renseignée, est la version de l'agrégat que l'appelant a lue : si
	// l'agrégat a avancé depuis, l'écriture échoue avec ErrVersionConflict.
	ExpectedVersion *int
}

// Appended est ce que l'enregistrement d'un événement retourne.
type Appended struct {
	Seq     int64
	Version int
}

// Recorded est un événement relu depuis le journal.
type Recorded struct {
	Seq           int64
	ID            uuid.UUID
	Type          string
	AggregateType string
	AggregateID   uuid.UUID
	Version       int
	IncidentID    uuid.UUID
	ActorID       *uuid.UUID
	CommandID     *uuid.UUID
	Payload       map[string]any
	// OccurredAt est l'horloge du client qui a créé la commande (nil si elle n'en a pas fourni).
	OccurredAt *time.Time
	RecordedAt time.Time
}

// SyncDelay est le temps écoulé entre la création de la commande sur l'appareil et son enregistrement par le
// serveur : c'est le délai de synchronisation d'une action faite hors ligne. Faux quand le client n'a pas
// fourni d'horloge, ou que la sienne est en avance.
func (r Recorded) SyncDelay() (time.Duration, bool) {
	if r.OccurredAt == nil || r.OccurredAt.After(r.RecordedAt) {
		return 0, false
	}
	return r.RecordedAt.Sub(*r.OccurredAt), true
}

// Recorder écrit dans le journal. Les services en dépendent par interface.
type Recorder interface {
	// Lock réserve l'écriture d'événements jusqu'à la fin de la transaction en cours. À appeler EN PREMIER dans
	// la transaction : tout verrou de ligne pris avant lui (compteurs, état) pourrait former un interblocage avec
	// une autre transaction qui l'a déjà.
	Lock(ctx context.Context) error
	// Append enregistre l'événement dans la transaction de ctx (ErrNoTransaction sinon).
	Append(ctx context.Context, event Event) (Appended, error)
	// Version retourne la version courante d'un agrégat (0 s'il n'a aucun événement) : celle qu'un client doit
	// renvoyer comme version attendue.
	Version(ctx context.Context, aggregateID uuid.UUID) (int, error)
}

// Command décrit la commande à l'origine d'une action : sa clé d'idempotence et l'horloge du client qui l'a créée.
// L'identifiant et l'horodatage viennent des en-têtes de la requête, posés dans le contexte par le middleware
// d'idempotence ; les événements écrits pendant la requête les reprennent.
type Command struct {
	ID              uuid.UUID
	ClientCreatedAt *time.Time
}

type commandKey struct{}

// WithCommand rattache la commande au contexte.
func WithCommand(ctx context.Context, command Command) context.Context {
	return context.WithValue(ctx, commandKey{}, command)
}

// CommandFrom retourne la commande du contexte, s'il y en a une.
func CommandFrom(ctx context.Context) (Command, bool) {
	command, ok := ctx.Value(commandKey{}).(Command)
	return command, ok
}

func encodePayload(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("events : encodage du contenu : %w", err)
	}
	return encoded, nil
}
