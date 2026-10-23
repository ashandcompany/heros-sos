# Backend Heros-SOS

API REST en Go (PostgreSQL, GORM). Cette partie couvre l'**identité et la sécurité** : comptes,
sessions, demande du statut de super-héros, modération des comptes, chiffrement des données
personnelles, mails.

## Démarrage rapide

Prérequis : Go (voir `go.mod`), Docker, `openssl`. Sans `make` sur la machine, `./make` à la racine du dépôt lance les mêmes cibles dans un conteneur (voir le [README racine](../README.md#sans-make-installé)).

**Toute l'application dans Docker** (proxy, front, API, PostgreSQL 18, Mailpit, RabbitMQ, documentation) :

```bash
# Depuis la RACINE du dépôt : les fichiers docker-compose*.yml y sont, le back, le front et la doc démarrent ensemble.
make up              # crée backend/.env au premier lancement, puis http://localhost:8088 (front + API)
                     # · /api/v1/docs (Swagger) · :8025 (mails reçus) · :3001/heros-sos/ (documentation)
make moderator EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
```

`make help` liste les raccourcis de la racine ; ils délèguent à `backend/Makefile`, dont les cibles
(`make up-dev`, `make moderator-dev`…) restent utilisables depuis `backend/`.

**API seule, lancée depuis Go** (le plus rapide pour travailler sur le backend) :

```bash
cd backend
make db-start        # PostgreSQL 18 dans Docker (bases herossos et herossos_test)
make env             # crée .env avec des secrets aléatoires
make migrate         # applique les migrations
make run             # API sur http://localhost:8080/api/v1 (HEROS_HTTP_ADDR pour changer de port)
```

Créer le premier modérateur (il valide ensuite les super-héros) :

```bash
make createmoderator EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
```

Avec `make run`, les mails ne sont pas envoyés : le driver `log` les écrit dans la sortie du
serveur, avec les liens d'activation et de réinitialisation. Avec `make up-dev`, ils partent par
SMTP vers Mailpit et se lisent sur http://localhost:8025.

> **Sauvegardez `HEROS_CRYPTO_KEYS`.** Sans cette clé, les données personnelles chiffrées en base
> sont définitivement illisibles.

## Tests

```bash
make db-start   # une fois
make test       # go test ./... -race -count=1
make check      # tout ce que vérifie le CI : formatage, vet, tests, vulnérabilités
make golangci   # analyse plus poussée (conventions Go, erreurs courantes) : voir .golangci.yml
```

Les tests d'intégration utilisent un vrai PostgreSQL 18, comme le CI et la production
(`HEROS_TEST_DATABASE_URL`, défini par le Makefile ; `make db-stop` supprime la base de test avec ses
données) et chacun travaille dans son propre schéma : ils tournent en parallèle sans se gêner. En
CI (`CI=true`), l'absence de base fait **échouer** les tests au lieu de les ignorer.

Ce qui protège de la régression :

- **Contrat OpenAPI** (`openapi/openapi.yaml`) : les tests de bout en bout valident chaque requête et
  chaque réponse contre lui, et vérifient que les routes enregistrées sont exactement celles
  décrites. Un statut ou un champ non documenté fait échouer les tests.
- **Vecteur d'or du chiffrement** : un test fige le format des données chiffrées ; le casser rendrait
  illisibles les données déjà en base.
- **Tests de concurrence** (`-race`) : inscriptions, activations, validations de demandes et
  verrouillages simultanés.
- `.env.example` est vérifié contre les variables réellement lues par le code.
- **Coût en requêtes SQL** : un test compte les requêtes d'une liste de modération avec 1 puis 4 lignes et échoue
  si le nombre change (une lecture par ligne, le « N+1 »), et vérifie qu'authentifier une requête ne coûte
  qu'une lecture. Une liste lit les comptes de sa page avec `user.Repository.FindByIDs`, jamais `FindByID`
  dans une boucle.

## Documentation interactive (Swagger UI)

En développement, l'API sert sa propre documentation, générée à partir du contrat
`openapi/openapi.yaml` : **http://localhost:8088/api/v1/docs** avec `make up` (ou `make up-dev` depuis
`backend/`), ou http://localhost:8080/api/v1/docs avec `make run`. On y lit chaque route, ses champs et
ses erreurs, et on peut l'essayer (« Try it out »).

Le même contrat est aussi rendu dans le site de documentation du projet, sans bouton d'essai : une page
par route, à http://localhost:3001/heros-sos/api/heros-sos-api (voir [docs-site/README.md](../docs-site/README.md)).
Les documents de conception (`MVP.md`, `STACK.md` et `stack/`, `TABLES.md` et `tables/`) y sont lus sur place.

- **Une écriture authentifiée** demande une session et le jeton CSRF : appeler `POST /auth/login`
  (le cookie est posé par le navigateur), copier le `csrfToken` de la réponse dans « Authorize »
  (`csrfToken`), puis essayer la route. L'origine de la page doit figurer dans
  `HEROS_ALLOWED_ORIGINS`, sinon les écritures sont refusées (`origin_not_allowed`) : c'est le cas du
  proxy (`:8088`) avec `make up-dev`, mais pas de l'accès direct à l'API (`:8081`). Avec `make run`,
  ajouter `http://localhost:8080` à cette variable dans `.env`. Les lectures (`GET`) n'y sont pas soumises.
- **Rien ne sort de la machine** : Swagger UI est embarqué dans le binaire (pas de CDN), le validateur
  en ligne est désactivé et la politique de sécurité de la page interdit les scripts en ligne et
  toute ressource extérieure. Seule cette page assouplit la politique de l'API.
- **Absente en production** : les routes `/docs` ne sont pas enregistrées (404).
- Le contrat n'a pas besoin d'être répété : ajouter une route dans `openapi.yaml` (voir plus bas) la
  fait apparaître dans la page. Les routes de la documentation elles-mêmes ne font pas partie du
  contrat (`Router.HandleOutsideContract`).
- Les fichiers de Swagger UI (`internal/apidocs/assets/`) sont ceux de la distribution officielle,
  sans modification, et vérifiés par un test : voir `internal/apidocs/THIRD_PARTY.md` pour la version
  et la mise à jour.

## Organisation du code

```
cmd/api                 serveur HTTP (-migrate pour appliquer les migrations)
cmd/createmoderator     crée un modérateur (mot de passe par HEROS_MODERATOR_PASSWORD)
cmd/resetmfa            réinitialise la 2FA d'un compte (appareil perdu), jamais par l'API
migrations/             SQL versionné par horodatage (golang-migrate), embarqué dans le binaire
openapi/openapi.yaml    contrat de l'API (référence), embarqué par openapi/openapi.go
internal/
  config/               TOUTE la configuration : variables d'environnement, valeurs par défaut,
                        validation (plus stricte en production)
  server/               assemblage : dépendances, routes, middlewares, tâches de fond, arrêt propre
  auth/                 mots de passe (argon2id), sessions, CSRF, jetons, 2FA (activation, vérification,
                        codes de secours, sessions limitées), endpoints /auth/*
  totp/                 mots de passe à usage unique basés sur le temps (RFC 6238)
  user/                 comptes ; chiffrement des données personnelles dans le dépôt
  hero/                 demande du statut de super-héros, annuaire des héros (public et admin)
  moderation/           actions des modérateurs sur les comptes
  account/              profil de l'utilisateur connecté : coordonnées, photo, changement d'adresse
                        email (confirmé par lien, puis validé par un modérateur), préférence de newsletter
  avatar/               photos de profil : contrôle et ré-encodage de l'image, stockage disque, service
  newsletter/           abonnements (double opt-in et désinscription par lien signé, préférence d'un compte)
  contact/              formulaire de contact : message transmis à l'équipe par mail
  signedlink/           liens signés Ed25519 (sans jeton en base) : confirmation de la newsletter, du
                        changement d'email, désinscription ; clé dérivée des clés de chiffrement
  crypto/               cascade AES-256-GCM puis XChaCha20-Poly1305 (clé de 512 bits)
  mail/                 construction MIME, SMTP, file d'envoi asynchrone, modèles
  apidocs/              documentation interactive (Swagger UI embarqué), hors production
  audit/                journal des actions sensibles
  httpx/                format des erreurs, JSON strict, IP client, middlewares
  validation/           règles de saisie (alignées sur le formulaire du frontend)
  ratelimit/ database/ clock/ ids/ bootstrap/ testutil/
```

Architecture : `handler → service → repository`. Les handlers traduisent HTTP, les services portent
les règles métier (sans connaître HTTP), les dépôts parlent à PostgreSQL. Les entités GORM ne sont
jamais renvoyées par l'API (DTO séparés).

## Conventions

- **Go idiomatique** : `PascalCase` pour ce qui est exporté, `camelCase` sinon ; erreurs enveloppées
  avec `%w` ; interfaces définies côté consommateur ; commentaires de documentation en français.
- **Aucune valeur codée en dur** : durées, tailles, seuils et limites vivent dans
  `internal/config/defaults.go` (surchargeables par variable d'environnement, documentées dans
  `.env.example`). Les constantes de format (longueur d'un jeton, limites de saisie) sont nommées
  dans leur paquet.
- **Concurrence** : toute goroutine appartient à un `errgroup` ou à un `WaitGroup` et s'arrête sur
  annulation du contexte ; les files sont bornées ; l'horloge est injectable (`clock.Clock`) ; les
  vérifications « puis écriture » sont des instructions SQL atomiques (pas de lecture puis écriture) ;
  `go test -race` est obligatoire.
- **Ressources** : les listes agrègent après avoir choisi leur page (voir `hero/repository.go`), et une tâche
  de fond est bornée par un délai (`HEROS_CLEANUP_TIMEOUT`). Un index n'est ajouté qu'après avoir mesuré la
  requête (`EXPLAIN ANALYZE`) : les deux derniers sont dans `migrations/…_add_lookup_indexes.up.sql`.
- **Données personnelles** : chiffrées dans le dépôt (`user`), jamais dans les logs ni dans les
  erreurs ; le SQL n'est jamais journalisé.

## Ajouter une route (autres domaines)

1. Décrire la route dans `openapi/openapi.yaml` (un test échoue sinon).
2. Écrire le handler dans son paquet, et lire l'utilisateur courant avec
   `auth.PrincipalFrom(r.Context())`.
3. L'enregistrer dans une fonction `register<Domaine>Routes` de `internal/server/routes_<domaine>.go`,
   appelée depuis `registerRoutes` :

```go
router.Handle(http.MethodPost, "/incidents", c.protected(h.Create))                       // connecté
router.Handle(http.MethodPost, "/incidents/{id}/close", c.protected(h.Close, user.RoleHero)) // rôle
```

`protected` exige une session valide, le rôle demandé, et le jeton CSRF sur les écritures. Pour
tester un handler sans session, injecter l'utilisateur avec `auth.WithPrincipal`.

Envoyer un mail : dépendre de l'interface `mail.Mailer` (la file asynchrone l'implémente).

## Contrat avec le frontend

- Toutes les routes sont sous `/api/v1`. Le navigateur ne parle qu'à une origine : un reverse proxy
  route `/api/v1/*` vers l'API et le reste vers Next.js, y compris ses propres routes `/api/…`.
- **Session** : cookie `HttpOnly` posé par `POST /auth/login` (`__Host-session` en production,
  `session` en développement). Le serveur Next relaie le `Set-Cookie` au navigateur et l'en-tête
  `Cookie` à l'API ; l'API n'accepte **que** le cookie (jamais `Authorization: Bearer`).
- **CSRF** : toute écriture authentifiée porte `X-CSRF-Token` (valeur de `csrfToken`, renvoyée par
  `/auth/login` et `/auth/me`).
- **Inscription** : `POST /auth/register` ne connecte pas l'utilisateur. Il faut activer le compte
  (lien reçu par mail, route `HEROS_ACTIVATION_PATH` du frontend, jeton dans `?token=`) puis se
  connecter.
- **Rôles** : `citizen`, `hero`, `moderator` (le frontend actuel utilise `citoyen` et `super-heros`).
- **Profil** : `GET` et `PATCH /me` valent pour tous les rôles (un super-héros est un compte avec un
  bloc `hero` en plus : pas de routes `/hero/me` à part). `PUT /me/avatar` reçoit **l'image brute**
  (`Content-Type: image/jpeg|png|webp`, ni JSON ni multipart) et répond l'`avatarUrl`, un chemin
  absolu sur la même origine (`/api/v1/avatars/<uuid>.jpg`, public, qui change à chaque envoi).
  L'adresse email change par `POST /me/email-change-requests`, validée par un modérateur
  (`/admin/email-change-requests`) : tant qu'elle ne l'est pas, `pendingEmail` la montre.
- **Newsletter et contact** (publics, sans session, limités par IP) : `POST /newsletter/subscriptions`
  envoie un lien de confirmation vers la page `HEROS_NEWSLETTER_CONFIRM_PATH` du frontend (jeton dans
  `?token=`, à poster sur `/newsletter/subscriptions/confirm`) ; `POST /contact/messages` écrit à
  l'équipe (`HEROS_CONTACT_RECIPIENT`), jamais au visiteur.
- **2FA (TOTP)** : obligatoire pour les modérateurs quand `HEROS_MFA_MODERATOR_REQUIRED=true` (imposé en
  production, le serveur refuse de démarrer sinon ; faux par défaut en développement). Une session dont la
  2FA n'est pas validée est **limitée** : l'API ne lui ouvre que `/auth/mfa/*`, `/auth/me` et la
  déconnexion, et répond `403 mfa_required` partout ailleurs. `/auth/login` renvoie l'état dans `mfa`
  (`verified`, `verification_required`, `enrollment_required`, `not_enrolled`). Il n'existe pas de route pour
  désactiver la 2FA : un appareil perdu se traite par l'exploitation, `make resetmfa-prod EMAIL=…` en
  production (voir « Production »), `make resetmfa EMAIL=…` à la racine en développement.
- **Confirmation de la nouvelle adresse** : l'adresse demandée reçoit un lien (page
  `HEROS_EMAIL_CHANGE_CONFIRM_PATH` du frontend, jeton dans `?token=`, à poster sur
  `/email-change-requests/confirm`). Un modérateur ne peut accepter la demande qu'une fois l'adresse
  confirmée (`email_change_not_confirmed` sinon) ; l'utilisateur est prévenu par mail de sa décision.
- **Désinscription** : chaque envoi de newsletter porte un lien vers la page
  `HEROS_NEWSLETTER_UNSUBSCRIBE_PATH` (jeton à poster sur `/newsletter/subscriptions/unsubscribe`).

## Environnements Docker

Les fichiers Compose sont **à la racine du dépôt** (et non dans `backend/`), pour lancer le back et le
front ensemble. Leurs chemins partent de la racine ; l'environnement de l'API (`.env`, `.env.production`)
et les secrets restent dans `backend/`, et le Makefile les passe à Compose (`--env-file`).

Un fichier de base (`docker-compose.yml`) décrit ce qui est commun ; `docker-compose.dev.yml` et
`docker-compose.prod.yml` s'y ajoutent. Chaque environnement est un projet Compose distinct
(`heros-sos-dev`, `heros-sos-prod`) : leurs conteneurs et leurs volumes ne se mélangent jamais.

|                  | Développement (`make up`)                                                                          | Production (`make up-prod`)                                                                               |
| ---------------- | ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| Ports publiés    | proxy `8088`, API `8081`, Mailpit `8025`, PostgreSQL `5432`, RabbitMQ, documentation `3001` — **sur `127.0.0.1` uniquement** | **proxy seul** : 80 et 443                                                                                |
| Front            | serveur de développement Next.js, code monté et rechargé à chaud                                       | image de production (sortie `standalone` de Next.js, `node server.js`), sans dépendances de développement |
| TLS              | non (HTTP)                                                                                             | automatique (Let's Encrypt, via Caddy)                                                                    |
| Secrets          | mots de passe publics de `.env.example`                                                                | fichiers dans `secrets/`, montés en `/run/secrets`                                                        |
| Migrations       | appliquées par l'API au démarrage                                                                      | tâche `migrate` qui doit réussir avant l'API                                                              |
| Mails            | Mailpit (rien ne sort de la machine)                                                                   | SMTP réel (`.env.production`)                                                                             |
| Réseau `backend` | ouvert                                                                                                 | **interne** : la base et l'API n'ont aucun accès à Internet                                               |

Ce qui est commun aux deux : l'API tourne sans root (`USER 65532`), système de fichiers en lecture
seule, toutes les capacités Linux retirées, `no-new-privileges`, journaux tournants, et un
**healthcheck** : `/api -healthcheck` interroge `/api/v1/ready` (base comprise) depuis l'intérieur du
conteneur. Une image `FROM scratch` n'a ni shell ni `curl` : la sonde est donc dans le binaire du
serveur (`cmd/api`, `internal/server/probe.go`), délai réglable par `HEROS_HEALTHCHECK_TIMEOUT`.
Les sondes `/health` et `/ready` réussies ne sont journalisées qu'au niveau `debug` (`HEROS_LOG_LEVEL=debug` pour les
voir) ; une sonde en échec reste en `warn` ou `error`.

**Premier déploiement en production**

```bash
cd backend
cp .env.production.example .env.production   # DOMAIN, ACME_EMAIL, SMTP… (aucun secret ici)
make secrets                                  # génère secrets/ (ne remplace jamais un secret existant)
make up-prod                                  # (aussi disponible depuis la racine : make secrets, make up-prod)
make moderator-prod EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ   # seulement depuis backend/
```

`secrets/` et `.env.production` ne sont jamais versionnés. Les secrets ne passent pas par
l'environnement des conteneurs : l'API lit `HEROS_<NOM>_FILE` (chemin d'un fichier) à la place de
`HEROS_<NOM>`. Définir les deux est une erreur, et le contenu d'un fichier n'apparaît jamais dans
un message d'erreur. Renseigner `secrets/omdb_api_key` avec une clé OMDb après `make secrets`;
la clé est obligatoire en production. **`secrets/crypto_keys` se sauvegarde hors du serveur.**

- **PostgreSQL 18** : le volume se monte sur `/var/lib/postgresql` (et non plus sur
  `/var/lib/postgresql/data`). Un volume créé avec PostgreSQL 17 n'est pas réutilisable : en
  développement, `make down-dev V=1` le supprime, avec ses données.
- **Adresse IP des visiteurs** : le proxy et le front ont des adresses fixes (`PROXY_IP`, `WEB_IP`,
  réseau `EDGE_SUBNET`), et seules elles sont dans `HEROS_TRUSTED_PROXIES`. Quand le serveur Next.js
  appelle l'API (Server Actions, rendu serveur), il doit **transmettre l'en-tête `X-Forwarded-For`**
  reçu du proxy. Sinon l'API ne voit que l'adresse du conteneur front, et tous les visiteurs
  partagent les mêmes limites de débit.
- **Nettoyage** : le front de développement garde `node_modules` et `.next` dans des volumes
  anonymes ; `make up-dev` et `make down-dev` les suppriment avec le conteneur.
- `make compose-check` valide la syntaxe des deux fichiers Compose, sans aucune configuration
  préalable (`.env`, `.env.production` ni `secrets/` n'ont besoin d'exister). À lancer après toute
  modification des fichiers Compose : le CI ne le fait pas encore.
- **Journaux** : chaque conteneur écrit sur sa sortie standard, que `make logs-dev` ou
  `make logs-prod` suit. Ils tournent pour **tous** les services : au plus `LOG_MAX_FILE` fichiers de
  `LOG_MAX_SIZE` par conteneur (3 × 10 Mo par défaut), faute de quoi Docker les garde indéfiniment.
  L'API écrit en texte en développement et en JSON en production, au niveau `HEROS_LOG_LEVEL` ; le
  proxy journalise chaque requête en JSON.
- Les variables des conteneurs (`POSTGRES_*`, `RABBITMQ_*`, `LOG_MAX_*`, ports, adresses) sont
  documentées dans `.env.example` et `.env.production.example`.

**RabbitMQ n'est pas encore utilisé par le code.** Le service est démarré (en développement, et en
production avec `--profile rabbitmq`) pour être prêt, mais aucun paquet Go ne s'y connecte : les
mails passent aujourd'hui par `mail.Queue`, une file en mémoire dans le processus de l'API (bornée,
avec reprises, vidée à l'arrêt propre). RabbitMQ n'interviendra qu'avec un worker de mails, derrière
la même interface `mail.Mailer`.

## Production

- `HEROS_ENV=production` active des contrôles stricts au démarrage : cookie `Secure`, SMTP chiffré,
  base en TLS, origines en HTTPS, paramètres argon2id minimaux. Une configuration incohérente
  empêche le démarrage.
- Derrière un reverse proxy, renseigner `HEROS_TRUSTED_PROXIES` : sinon tous les clients partagent
  l'adresse IP du proxy, et donc les mêmes limites de débit (déjà fait par les fichiers Compose).
- Le limiteur de débit est en mémoire : exact pour une seule instance de l'API.
- Les mails en file sont perdus si le processus est tué brutalement (un worker RabbitMQ les rendra
  durables sans changer le code appelant : seul `mail.Mailer` est connu des autres paquets).
- Sauvegarder `HEROS_CRYPTO_KEYS` séparément de la base ; pour une rotation, ajouter une clé puis
  changer `HEROS_CRYPTO_ACTIVE_KEY_ID` (les anciennes valeurs restent lisibles). La clé qui signe les
  liens (confirmation de la newsletter, confirmation d'un changement d'email, désinscription) en est
  dérivée : les liens déjà envoyés restent valables tant que leur clé est configurée.
- **Photos de profil** : le seul endroit où l'API écrit sur le disque, dans le volume `avatars`
  (`HEROS_AVATAR_DIR`, fixé par l'image). À sauvegarder avec la base : une photo référencée par un
  compte mais absente du volume répond 404. Un envoi interrompu au mauvais moment peut laisser un
  fichier orphelin, sans effet visible.
- **Formulaire de contact** : le mail est la seule trace du message, et la file d'envoi est interne
  au processus (voir plus haut). Régler `HEROS_CONTACT_RECIPIENT` sur la boîte de l'équipe.
- **Newsletter** : l'abonnement (double opt-in), la désinscription et l'envoi des numéros existent. Les
  modérateurs rédigent des brouillons sous `/admin/newsletters` (HTML assaini) et les envoient à tous les
  abonnés confirmés ou à un seul ; un numéro envoyé est en lecture seule. Les mails partent en
  arrière-plan par la file d'envoi (perdus si le processus s'arrête brutalement, sans suivi par
  destinataire). `mail.SendNewsletter` est la seule façon d'en envoyer un et refuse un message sans lien de
  désinscription. Tous les mails HTML partagent la mise en page `internal/mail/templates/layout.html.tmpl`
  (logo `frontend/public/mail/logo.png` servi depuis l'adresse publique du frontend, thèmes clair et sombre).
  Le consentement est daté dans `newsletter_subscriptions.confirmed_at`.
- **Double authentification des modérateurs** : `HEROS_MFA_MODERATOR_REQUIRED=true` est imposé (le serveur
  refuse de démarrer avec `false`).
  - *Activation, pour chaque modérateur* : l'exploitation crée le compte (`make moderator-prod`, depuis
    `backend/`), puis la personne se connecte elle-même. Sa session est limitée : le front l'envoie sur
    `/connexion/double-authentification`, où elle ajoute la clé à une application d'authentification (clé à
    saisir ou lien `otpauth://`, pas de QR code), saisit un premier code et **note ses codes de secours**,
    affichés une seule fois. L'exploitation ne voit jamais le secret.
  - *Codes de secours presque épuisés* : tant qu'il a son appareil, le modérateur en régénère depuis
    `/admin/securite` (un code de l'application est exigé).
  - *Réinitialisation (appareil et codes de secours perdus)* : après avoir vérifié l'identité de la personne
    par un autre moyen que le compte lui-même, lancer `make resetmfa-prod EMAIL=…` **depuis `backend/`**. La
    commande supprime sa 2FA, ferme toutes ses sessions et écrit `auth.mfa_reset` dans le journal d'audit ;
    à sa prochaine connexion, la personne réactive sa 2FA comme à la première fois. Elle passe par la base,
    pas par l'API : elle marche aussi quand tous les modérateurs sont verrouillés.
  - **Pas `make resetmfa`** : cette cible (à la racine, `backend/` n'en a pas) vise la pile de développement.
    La lancer en production ne débloquerait personne.
- **Changement d'adresse email** : régler `HEROS_EMAIL_CHANGE_CONFIRM_PATH` sur la page du frontend
  (`/changement-email/confirmation`) ; le lien reçu vaut `HEROS_EMAIL_CHANGE_CONFIRM_TTL` (48 h par défaut).
