# Contrat backend de Maxime (Dev A) : ce qui est fait, ce qui reste

État au 21/10/2026, branche `feat/account-security` (poussée sur GitHub, **pas encore fusionnée** dans `develop`).
La source de vérité des routes est [openapi/openapi.yaml](openapi/openapi.yaml) : les tests échouent si une route
diverge du fichier. Ce document n'en recopie pas le détail ; il dit **qui livre quoi**, **où en est chaque domaine** et
**ce qui reste avant une vraie mise en ligne**.

Légende : ✅ fait, testé et dans le contrat · 🟡 fait mais à finir ou à décider · ⬜ pas fait.

## 1. Périmètre

Le contrat compte **91 opérations** (plus `/health` et `/ready`). Elles se répartissent ainsi :

| Responsable | Domaines (tags OpenAPI) | Opérations |
| --- | --- | --- |
| **Maxime (Dev A)** | Authentification (14), Compte (7), Contact et newsletter (4), Super-héros : demandes et liste publique (4), Modération : comptes, demandes de héros, changements d'email, annuaire (11) | **40** |
| Dev B | Incidents (11), Réputation (10), Films (15), Vilains (4), Types d'intervention (3), Blog (7), arbitrage d'une intervention (1) | 51 |
| Socle commun | Santé (`/health`, `/ready`) | 2 |

Code de Dev A : `auth`, `totp`, `user`, `account`, `hero`, `moderation`, `newsletter`, `contact`, `signedlink`, `mail`,
`audit`, `crypto`, `avatar`, `bootstrap`, `config`, et les commandes `cmd/createmoderator` et `cmd/resetmfa`.

## 2. Ce qui est fait

### 2.1 Les 40 routes de Dev A

Colonne « Front » : l'écran appelle bien la route quand `API_URL` est renseignée (voir
[frontend/docs/backend-contract.md](../frontend/docs/backend-contract.md)).

| Domaine | Routes | Backend | Front |
| --- | --- | --- | --- |
| Inscription et activation | `POST /auth/register`, `/auth/activate`, `/auth/resend-activation` | ✅ | ✅ |
| Connexion et session | `POST /auth/login`, `/auth/logout`, `/auth/logout-all` ; `GET /auth/me` | ✅ | ✅ |
| Mot de passe oublié | `POST /auth/password-reset`, `/auth/password-reset/confirm` | ✅ | ✅ |
| **Double authentification (TOTP)** | `GET /auth/mfa` ; `POST /auth/mfa/setup`, `/enable`, `/verify`, `/recovery-codes` | ✅ | ✅ |
| Profil (citoyen et héros, une seule route) | `GET /me`, `PATCH /me` (téléphone, adresse) | ✅ | ✅ |
| Photo de profil | `PUT /me/avatar` (image brute), `GET /avatars/{name}` | ✅ | ✅ |
| **Changement d'email confirmé par lien** | `POST /me/email-change-requests`, `POST /email-change-requests/confirm` | ✅ | ✅ |
| Newsletter | `POST /newsletter/subscriptions`, `/confirm`, **`/unsubscribe`** ; `PUT /me/newsletter` | ✅ | ✅ |
| Contact | `POST /contact/messages` | ✅ | ✅ |
| Demande du statut de héros | `POST /heroes/requests`, `GET /heroes/requests/me` | ✅ | ✅ |
| Liste et fiche publiques des héros | `GET /heroes`, `GET /heroes/{heroId}` | ✅ | ✅ (branché par Emma) |
| Modération des comptes | `GET /admin/users` ; `POST /admin/users/{id}/deactivate`, `/reactivate`, `/verify` | ✅ | ✅ |
| Modération des demandes de héros | `GET /admin/hero-requests` ; `POST …/{userId}/approve`, `/reject` | ✅ | ✅ |
| Modération des changements d'email | `GET /admin/email-change-requests` ; `POST …/{id}/approve`, `/reject` | ✅ | ✅ |
| **Annuaire admin des héros** | `GET /admin/heroes` | ✅ | ✅ |

### 2.2 Sécurité et transversal

| Sujet | État |
| --- | --- |
| Mots de passe argon2id, mémoire bornée par un sémaphore, réhachage automatique | ✅ |
| Cookie de session (`HttpOnly`, `Secure` en production), CSRF sur toute écriture, contrôle d'origine | ✅ |
| Sessions par rôle : membre 2 h d'inactivité / 7 j, « rester connecté » 30 j, modérateur 15 min / 8 h | ✅ |
| Verrouillage progressif des comptes, limitation de débit par IP, par identifiant, par adresse et par utilisateur | ✅ |
| Données personnelles chiffrées en base (AES-256-GCM puis XChaCha20, clés versionnées) | ✅ |
| Liens signés Ed25519 sans jeton en base : confirmation newsletter, confirmation d'email, désinscription | ✅ |
| **2FA des modérateurs** : obligatoire en production (le serveur refuse de démarrer sinon), session limitée tant que le code n'est pas saisi, codes de secours, anti-rejeu, jeton renouvelé | ✅ |
| Réinitialisation de la 2FA réservée à l'exploitation (`make resetmfa EMAIL=…`), tracée dans l'audit | ✅ |
| Journal d'audit des actions sensibles (écriture) | ✅ |
| Création du premier modérateur (`make moderator`) | ✅ |
| Envoi de mails asynchrone et borné (activation, mot de passe, changement d'email, contact, notification de décision) | ✅ |
| Image Docker : Go épinglé à 1.26.10, `govulncheck` sans vulnérabilité atteignable | ✅ |
| Listes de modération sans « N+1 » (test de garde), annuaire des héros 44 ms → 1 ms, index ajoutés après mesure | ✅ |
| Nettoyage périodique borné par un délai (`HEROS_CLEANUP_TIMEOUT`) | ✅ |

### 2.3 Preuves

- 354 fonctions de test côté backend ; `go test ./... -race -count=1` passe sur chaque commit (aucun test ignoré).
- Les tests de bout en bout valident chaque requête et chaque réponse contre l'OpenAPI.
- Le front est vérifié par `tsc`, eslint, prettier et la vérification des types générés depuis l'OpenAPI.

### 2.4 Interfaces que Maxime fournit aux autres

| Pour | Interface | Garantie |
| --- | --- | --- |
| Dev B | `auth.PrincipalFrom`, `Protected(rôles…)` | Une session **limitée** (2FA pas encore validée) n'ouvre rien par défaut : les routes de Dev B sont protégées sans changement. Seules les routes déclarées `ProtectedPendingMFA` lui sont ouvertes. |
| Dev B | `mail.Service` / `mail.Mailer`, `audit.Recorder`, `user.Repository` | Compatibles. Nouveau : `user.Repository.FindByIDs`, à utiliser dans une liste au lieu de `FindByID` dans une boucle. |
| Emma | Contrat OpenAPI + types générés (`npm run api:types`) | À jour pour toutes les routes ci-dessus. |

## 3. Ce qui reste à faire

### 3.1 Pour livrer ce qui est fait

- [x] **`develop` est fusionné dans `feat/account-security`** (PR blog #26, correctifs de tests #28, incidents et
  arbitrage, pages publiques des héros, puis la PR #31 d'Emma : blog, films des super-vilains, incidents publics).
  Emma avait câblé en parallèle le compte, la newsletter, le contact et la déconnexion partout ; en cas de
  doublon, la version de cette branche a été gardée (elle gère la confirmation d'email, la 2FA et la
  désinscription). Les tests du backend, le build du front et celui du site de documentation passent sur le
  résultat. L'index sur `intervention_heroes` touche une table de Dev B : à lui signaler.
- [ ] Ouvrir la PR et la faire relire (Dev B pour le contrat et la migration, Emma pour le front).
- [ ] Vérifier les pseudos de `.github/CODEOWNERS` (déduits de l'historique).
- [ ] Activer GitHub Pages si le site doit être publié (réglages décrits dans `docs-site/README.md`). Le dépôt est
  public : le site le sera aussi.

### 3.2 Avant tout vrai utilisateur (écarts avec le cahier des charges ou la production)

| # | Point | État | Décision ou travail |
| --- | --- | --- | --- |
| 1 | **Droit à l'effacement et export des données (RGPD)** | ⬜ Le principe est écrit (suppression réelle ou anonymisation, pas de `deleted_at`), mais aucune route ni commande ne le fait. | Concevoir : suppression ou anonymisation d'un compte, avec les cascades (sessions, abonnements, demandes), puis l'exposer. |
| 2 | **Changer son mot de passe quand on est connecté** | ⬜ Seule la réinitialisation par mail existe. | À décider avec Emma : route `POST /me/password` (mot de passe actuel exigé, autres sessions fermées). |
| 3 | **Modifier son pseudo ou sa description de héros** | ⬜ `PATCH /me` ne prend que le téléphone et l'adresse. | Confirmer avec le cahier ; sinon ajouter les champs (le pseudo est unique, sans casse). |
| 4 | **2FA récente pour les actions sensibles** | ⬜ La 2FA protège la session entière, pas chaque action (valider un héros, désactiver un compte). | Durcissement : exiger un code de moins de N minutes sur ces routes. |
| 5 | **Lire le journal d'audit** | 🟡 Il s'écrit, mais aucune route ne le lit. | Décider : route réservée aux modérateurs, ou requête SQL côté exploitation. |
| 6 | **Supprimer sa photo de profil** | ⬜ On peut la remplacer, pas la retirer. | Petite route `DELETE /me/avatar`, si l'écran le demande. |
| 7 | **Limiteur de débit en mémoire** | 🟡 Exact pour **une** instance de l'API. | À revoir (Redis ou table) avant d'en lancer plusieurs. |
| 8 | **Mails non durables** | 🟡 La file est dans le processus : un arrêt brutal perd les mails en attente. RabbitMQ est démarré mais pas utilisé. | Worker RabbitMQ derrière `mail.Mailer`, sans changer les appelants. |
| 9 | **Exploitation** | ⬜ Sauvegarde séparée de `HEROS_CRYPTO_KEYS`, du volume `avatars` et de la base ; SMTP réel ; premiers modérateurs et leur 2FA ; supervision. | Checklist « Production » du README du backend, à dérouler. |
| 10 | **Envoi de la newsletter** | ⬜ L'abonnement et la désinscription existent ; aucun numéro ne part. | Version 2 : tâche d'envoi, avec `mail.SendNewsletter` (refuse un message sans lien de désinscription). |

### 3.3 Avec Emma (phase 3)

- [x] Fiches publiques des héros branchées sur `GET /heroes` (Emma) ; `asMockDataUser` est supprimé.
- [ ] Supprimer le reste des mocks une fois tous les domaines branchés : `userFromToken` et
  `MOCK_SESSION_MAX_AGE`. Il reste sur mocks : les incidents résolus publics (accueil, fiches héros et vilains),
  le tableau de bord héros et le résumé de modération (statistiques, sans contrat). Le blog et les films des
  super-vilains sont branchés.
- [ ] Jouer le scénario de démonstration de bout en bout (voir `MVP.md`).
- [ ] Durcissement puis déploiement (rien n'est déployé ; aucune mise en ligne sans feu vert explicite).

### 3.4 Dette technique

| Point | Détail |
| --- | --- |
| CI | `make golangci` est dans la CI, **sur les remarques nouvelles seulement** (`--new-from-merge-base`) : celles de la ligne suivante ne la font pas échouer. À retirer quand elles seront corrigées (retirer `GOLANGCI_FLAGS` du workflow). La CI ne lance ni `gofmt` ni `go vet` (`make lint`), ne construit pas les images Docker et ne vérifie pas les fichiers Compose. |
| Code de Dev B | 10 remarques de `make golangci` (`film` 4, `blog` 2, `incident`, `intervention`, `interventiontype`, `villain` 1 chacun) et 4 fichiers mal formatés (`blog/handler.go`, `blog/blog_test.go`, `villain/handler.go`, `server/incident_test.go`). Corrections d'une ligne : `make fmt`, puis `make golangci` pour la liste. |
| Fonctions longues | `config.Load` (119 lignes) et `server.NewWithDB` (49 instructions) : non découpées, car les deux développeurs les modifient (conflits). |
| Note moyenne des héros | Le tri par note relit tous les avis (10 ms à 60 000 avis). Au-delà de 100 000, stocker la moyenne sur `hero_profiles`. |
| OpenAPI | Découpage en `auth.yaml` et `incidents.yaml` : optionnel, à faire entre deux PR de Dev B. |
| `govulncheck` | Un avis sur `x/crypto/openpgp`, sans correctif et que le code n'appelle pas : à surveiller. |

### 3.5 Hors MVP (version 2)

2FA des héros et des citoyens, liste des appareils connectés, envoi des numéros de la newsletter, équipes de héros,
proximité (`hero_locations`), encyclopédie Marvel.

## 4. Comment vérifier

```bash
cd backend
make test                 # go test ./... -race -count=1 (PostgreSQL requis : make db-start)
make check                # formatage, vet, tests, govulncheck (la CI lance les tests et govulncheck, pas le formatage ni vet)
make golangci             # analyses supplémentaires (voir .golangci.yml)
make resetmfa EMAIL=…     # exploitation : retirer la 2FA d'un compte
```

Contrat et branchement du front : [frontend/docs/backend-contract.md](../frontend/docs/backend-contract.md).
Périmètre et décisions : [MVP.md](MVP.md), [STACK.md](STACK.md), [TABLES.md](TABLES.md).
