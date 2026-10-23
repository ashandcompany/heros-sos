# MVP de Heros-SOS

Le MVP permet de jouer le cœur du projet : un citoyen signale un incident, un héros le prend en charge et le clôture, un modérateur surveille (cahier §1.3).

> **État au 23/10/2026.** Le périmètre du MVP est livré, à une exception près : la suppression d'un doublon, remplacée par le rejet de l'incident en modération. Presque tout ce qui était rangé en version 2 l'a été aussi (voir « Hors MVP » plus bas), et le défi technique (performance, hors ligne, journal d'événements, ELK) s'y est ajouté.

## Scénario de démo

1. Un citoyen s'inscrit, active son compte par email, se connecte.
2. Il consulte les incidents en cours pour éviter un doublon, puis déclare un incident.
3. Un modérateur valide un compte héros, approuve ou rejette les incidents déclarés (le rejet écarte un doublon) et peut changer la priorité.
4. Le héros reçoit l'alerte, voit la liste par priorité, prend l'incident, puis le clôture (réussi ou échoué).
5. Les incidents réussis apparaissent dans le carrousel de la page d'accueil.

## Dans le MVP

| Domaine | Contenu |
|---|---|
| **Comptes** | Inscription (§5.4), activation par email, connexion (« Rester connecté » : 30 jours), déconnexion et déconnexion de tous les appareils, `/auth/me`, mot de passe oublié, sessions, 2FA TOTP des modérateurs |
| **Libre-service du compte** | Profil (`/me`, pour tous les rôles), photo de profil, changement d'adresse email (confirmé par un lien envoyé à la nouvelle adresse, puis validé par un modérateur), préférence de newsletter, formulaire de contact |
| **Rôles** | Citoyen, héros, modérateur ; demande « Devenir super-héros » validée par un modérateur |
| **Incidents** | Déclaration, liste des incidents (résolus ou non, filtrable par statut, héros et vilain), mes incidents, détail, priorité, modération (approuver, rejeter, corriger le statut). Il n'y a pas de suppression d'un doublon : un incident rejeté n'est plus visible du public ni des héros |
| **Interventions** | Un héros prend un incident et le clôture. La table `intervention_heroes` était prête pour l'équipe, qui a été livrée ensuite (voir « Hors MVP »). |
| **Public** | Page d'accueil (derniers incidents réussis), liste des héros (alias, photo, description), liste des vilains |
| **Modération** | Valider les héros, gérer les incidents, désactiver un compte, gérer les vilains |
| **Mails** | Activation, mot de passe oublié, nouvel incident vers héros et modérateurs, changement d'adresse email (confirmation, décision du modérateur), newsletter (confirmation d'abonnement, lien de désinscription dans chaque numéro), contact |
| **Sécurité de base** | argon2id, cookie de session, CSRF, limitation de débit sur la connexion, contrôle d'accès à chaque endpoint, validation des entrées, TLS, journal d'audit simple, liens signés Ed25519 (confirmations et désinscription), 2FA TOTP des modérateurs (session limitée tant que le code n'est pas saisi) |

### Tables concernées (13 sur 25)

`users`, `hero_profiles`, `sessions`, `user_tokens`, `user_mfa`, `email_change_requests`, `newsletter_subscriptions`, `incident_types`, `incidents`, `interventions`, `intervention_heroes`, `villains`, `audit_logs`.

`villain_specialties` et `incident_villains` ont été ajoutées : les vilains sont liés aux incidents.

Le schéma compte aujourd'hui **31 tables** (migrations du 23/10/2026) : 24 des 25 du schéma d'origine (`mail_outbox`, le repli sans RabbitMQ, n'est pas implémentée) et 7 ajoutées par le travail de performance (quatre tables de compteurs, `events`, `event_publication` et `processed_commands`). Voir [TABLES.md](TABLES.md).

## Hors MVP (état au 23/10/2026)

Ce qui était rangé en version 2, et ce qui s'y est ajouté. ✅ livré, ⬜ pas fait.

| Sujet | État | Détail |
|---|---|---|
| Notes et réponses des héros | ✅ | `hero_reviews` et `hero_review_replies`, avec modération des avis |
| Films et OMDb | ✅ | `/films`, `/hero/films`, `/imdb/titles` (voir [OMDb](stack/omdb.md)) |
| Statistiques (§5.8.1 et §5.8.2) | ✅ | Tableau de bord des héros (`GET /hero/dashboard`) et résumé de modération (`GET /admin/moderation/summary`), voir [Statistiques](tables/statistiques.md) |
| Articles du blog | ✅ | `/articles` et `/admin/articles` |
| Envoi des numéros de la newsletter | ✅ | `/admin/newsletters` : brouillons, envoi à tous les abonnés confirmés ou à un seul |
| Suivi de remise des numéros | ⬜ | Les envois partent en arrière-plan, sans suivi par destinataire |
| Équipes de héros | ✅ | Invitations, confirmations, validation du résultat, arbitrage par un modérateur |
| 2FA TOTP des héros et des citoyens | ✅ | Facultative (obligatoire pour les modérateurs en production) |
| Libre-service du compte : mot de passe, pseudo et description du héros, retrait de la photo, export et effacement des données (RGPD) | ✅ | `POST /me/password`, `PATCH /me`, `DELETE /me/avatar`, `GET /me/export`, `DELETE /me` (anonymisation) |
| Liste des appareils connectés | ⬜ | Aucune route dans le contrat OpenAPI |
| Proximité des héros (`hero_locations`) | ✅ | Position partagée avec consentement, alerte des héros proches (rayon de 10 km) |
| RabbitMQ pour les mails | ✅ | Voir « Mails et RabbitMQ » plus bas |
| Encyclopédie Marvel | ⬜ | Source de données à trouver (voir les [points ouverts](tables/points-ouverts.md)) |
| Performance à 5 millions d'incidents, hors ligne, journal d'événements, ELK | ✅ | Ajouté au périmètre : [Performance et synchronisation](stack/performance-et-synchronisation.md) |

## À ne pas couper

- **`internal/crypto` (cascade AES-256-GCM puis XChaCha20-Poly1305) dès le départ.** Il touche tous les accès à `users`. L'ajouter après obligerait à migrer des données déjà enregistrées en clair. Exception : un MVP qui reste une démo avec des données fictives.
- **La 2FA des modérateurs avant tout vrai utilisateur.** Un compte modérateur piraté donne accès à tout. Elle est **facultative pour une démo** (en développement, `HEROS_MFA_MODERATOR_REQUIRED` vaut faux par défaut) et **obligatoire pour une mise en ligne** : en production le backend l'impose et refuse de démarrer si on la désactive. Elle est implémentée (TOTP, codes de secours, commande `resetmfa`), et facultative pour les autres rôles.

## Mails et RabbitMQ

Pour le MVP, les mails partent d'une goroutine de l'API, derrière une interface `Mailer`, et RabbitMQ s'y est branché ensuite sans changer le reste : c'est fait depuis le 23/10/2026. Deux modes, choisis par `HEROS_MAIL_QUEUE` : `memory` (le défaut : file en mémoire, les mails en attente sont perdus si l'API est tuée brutalement) et `rabbitmq` (file durable, service `mail-worker`, confirmation du courtier, reprises et file morte `mail.dead`). En production, le mode durable est à activer (`COMPOSE_PROFILES=rabbitmq` et `HEROS_MAIL_QUEUE=rabbitmq`) : voir [RabbitMQ](stack/rabbitmq.md).

## Ordre de construction

1. **Socle :** `go mod init`, Docker, `openapi.yaml`, `/health`, intégration continue.
2. **Base de données :** migrations, modèles GORM, `internal/crypto`.
3. **Authentification de bout en bout**, branchée sur les pages d'Emma (inscription, connexion, mot de passe oublié).
4. **Incidents et interventions.**
5. **Modération et pages publiques.**
6. **Mails**, puis finitions et déploiement.

L'ordre réel et les écarts, jour par jour, sont dans `PLANNING.md` (à la racine du dépôt).

## Branchement du frontend

Le front est branché sur l'API pour tous les domaines dès que `API_URL` est renseignée (116 opérations du contrat, plus `/health` et `/ready`) : identité, 2FA, compte, modération, incidents et interventions, vilains, vidéothèque, avis, blog, newsletter, contact et tableaux de bord. Sans `API_URL`, il tourne sur des données mockées. L'état exact, domaine par domaine, est dans `frontend/docs/backend-contract.md`.

Voir aussi [STACK.md](STACK.md) pour les choix techniques et [TABLES.md](TABLES.md) pour le schéma.
