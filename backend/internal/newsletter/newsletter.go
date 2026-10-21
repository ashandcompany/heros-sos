// Package newsletter gère les abonnements à la newsletter : l'abonnement d'un visiteur depuis le pied
// de page (double opt-in par lien signé), la préférence d'un titulaire de compte, et la purge des
// demandes jamais confirmées. L'envoi de la newsletter elle-même n'existe pas encore.
//
// Un abonnement est une ligne par adresse (newsletter_subscriptions) :
//
//	pending      demande déposée, lien de confirmation envoyé, pas encore de consentement ;
//	subscribed   consentement prouvé par confirmed_at ;
//	unsubscribed l'abonné s'est retiré (la ligne reste : elle empêche de le réinscrire sans nouveau consentement).
package newsletter

import "errors"

// Status est l'état d'un abonnement.
type Status string

const (
	StatusPending      Status = "pending"
	StatusSubscribed   Status = "subscribed"
	StatusUnsubscribed Status = "unsubscribed"
)

// ErrLinkInvalid indique un lien de confirmation falsifié, expiré, ou qui ne correspond plus à une
// demande en attente (par exemple après un désabonnement).
var ErrLinkInvalid = errors.New("newsletter : lien de confirmation invalide ou expiré")
