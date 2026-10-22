# MVP de Heros-SOS

Le MVP permet de jouer le cœur du projet : un citoyen signale un incident, un héros le prend en charge et le clôture, un modérateur surveille (cahier §1.3).

## Scénario de démo

1. Un citoyen s'inscrit, active son compte par email, se connecte.
2. Il consulte les incidents en cours pour éviter un doublon, puis déclare un incident.
3. Un modérateur valide un compte héros, et peut supprimer un doublon ou changer la priorité.
4. Le héros reçoit l'alerte, voit la liste par priorité, prend l'incident, puis le clôture (réussi ou échoué).
5. Les incidents réussis apparaissent dans le carrousel de la page d'accueil.

## Dans le MVP

| Domaine | Contenu |
|---|---|
| **Comptes** | Inscription (§5.4), activation par email, connexion (« Rester connecté » : 30 jours), déconnexion et déconnexion de tous les appareils, `/auth/me`, mot de passe oublié, sessions, 2FA TOTP des modérateurs |
| **Libre-service du compte** | Profil (`/me`, pour tous les rôles), photo de profil, changement d'adresse email (confirmé par un lien envoyé à la nouvelle adresse, puis validé par un modérateur), préférence de newsletter, formulaire de contact |
| **Rôles** | Citoyen, héros, modérateur ; demande « Devenir super-héros » validée par un modérateur |
| **Incidents** | Déclaration, liste des incidents en cours, mes incidents, détail, priorité, suppression d'un doublon |
| **Interventions** | Un héros prend un incident et le clôture. Un seul héros par intervention : la table `intervention_heroes` est déjà prête pour l'équipe, rien ne sera à refaire. |
| **Public** | Page d'accueil (derniers incidents réussis), liste des héros (alias, photo, description), liste des vilains |
| **Modération** | Valider les héros, gérer les incidents, désactiver un compte, gérer les vilains |
| **Mails** | Activation, mot de passe oublié, nouvel incident vers héros et modérateurs, changement d'adresse email (confirmation, décision du modérateur), newsletter (confirmation d'abonnement, lien de désinscription dans chaque numéro), contact |
| **Sécurité de base** | argon2id, cookie de session, CSRF, limitation de débit sur la connexion, contrôle d'accès à chaque endpoint, validation des entrées, TLS, journal d'audit simple, liens signés Ed25519 (confirmations et désinscription), 2FA TOTP des modérateurs (session limitée tant que le code n'est pas saisi) |

### Tables concernées (13 sur 25)

`users`, `hero_profiles`, `sessions`, `user_tokens`, `user_mfa`, `email_change_requests`, `newsletter_subscriptions`, `incident_types`, `incidents`, `interventions`, `intervention_heroes`, `villains`, `audit_logs`.

Ajouter `villain_specialties` et `incident_villains` si on lie les vilains aux incidents.

## Hors MVP (version 2)

- Notes et réponses des héros, films et OMDb, statistiques (§5.8.1 et §5.8.2).
- Envoi des numéros de la newsletter (l'abonnement par double opt-in et la désinscription existent déjà), articles du blog.
- Équipes de héros : invitations, confirmations, validation du résultat, arbitrage par un modérateur.
- 2FA TOTP des héros et des citoyens (celle des modérateurs n'est pas « hors MVP » : voir « À ne pas couper »), liste des appareils connectés.
- Proximité des héros (`hero_locations`), encyclopédie Marvel.

## À ne pas couper

- **`internal/crypto` (cascade AES-256-GCM puis XChaCha20-Poly1305) dès le départ.** Il touche tous les accès à `users`. L'ajouter après obligerait à migrer des données déjà enregistrées en clair. Exception : un MVP qui reste une démo avec des données fictives.
- **La 2FA des modérateurs avant tout vrai utilisateur.** Un compte modérateur piraté donne accès à tout. Elle est **facultative pour une démo** (en développement, `HEROS_MFA_MODERATOR_REQUIRED` vaut faux par défaut) et **obligatoire pour une mise en ligne** : en production le backend l'impose et refuse de démarrer si on la désactive. Elle est déjà implémentée (TOTP, codes de secours, commande `resetmfa`) ; seule celle des autres rôles reste en version 2.

## Mails et RabbitMQ

Pour le MVP, les mails partent depuis le worker ou une goroutine, derrière une interface `Mailer`. RabbitMQ s'y branche ensuite sans changer le reste. L'intégrer dès le MVP est possible, mais cela ajoute un service à héberger et à tester avant le premier incident déclaré.

## Ordre de construction

1. **Socle :** `go mod init`, Docker, `openapi.yaml`, `/health`, intégration continue.
2. **Base de données :** migrations, modèles GORM, `internal/crypto`.
3. **Authentification de bout en bout**, branchée sur les pages d'Emma (inscription, connexion, mot de passe oublié).
4. **Incidents et interventions.**
5. **Modération et pages publiques.**
6. **Mails**, puis finitions et déploiement.

## À vérifier avec Emma

Le front est branché sur l'API pour l'identité (inscription, connexion, 2FA des modérateurs, sessions), le compte (profil, photo, changement d'email), la modération des comptes, le contact et la newsletter. Les incidents, les interventions, les vilains, la vidéothèque, les avis, le blog et le tableau de bord héros tournent encore sur des données mockées : les brancher domaine par domaine avec Emma (voir `frontend/docs/backend-contract.md`).

Voir aussi [STACK.md](STACK.md) pour les choix techniques et [TABLES.md](TABLES.md) pour le schéma.
