// Package newsletter gère les abonnements à la newsletter : l'abonnement d'un visiteur depuis le pied
// de page (double opt-in par lien signé), la préférence d'un titulaire de compte, et la purge des
// demandes jamais confirmées. L'envoi de la newsletter elle-même n'existe pas encore : quand il existera,
// il passera par mail.Service.SendNewsletter, qui refuse tout numéro sans lien de désinscription
// (Service.UnsubscribeLink).
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

// États d'un abonnement : adresse à confirmer, abonné, désabonné.
const (
	StatusPending      Status = "pending"
	StatusSubscribed   Status = "subscribed"
	StatusUnsubscribed Status = "unsubscribed"
)

// ErrLinkInvalid indique un lien (de confirmation ou de désinscription) falsifié, expiré, ou qui ne
// correspond plus à un abonnement (par exemple une confirmation après un désabonnement).
var ErrLinkInvalid = errors.New("newsletter : lien de confirmation invalide ou expiré")
