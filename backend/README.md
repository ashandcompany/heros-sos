# Backend Heros-SOS

API REST en Go (PostgreSQL, GORM). Cette partie couvre l'**identité et la sécurité** : comptes,
sessions, demande du statut de super-héros, modération des comptes, chiffrement des données
personnelles, mails.

## Démarrage rapide

Prérequis : Go (voir `go.mod`), Docker, `openssl`.

**Toute l'application dans Docker** (proxy, front, API, PostgreSQL 18, Mailpit, RabbitMQ) :

```bash
cd backend
make env             # crée .env avec des secrets aléatoires
make up-dev          # http://localhost:8088 (front + API) · http://localhost:8025 (mails reçus)
make moderator-dev EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
```

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

## Organisation du code

```
cmd/api                 serveur HTTP (-migrate pour appliquer les migrations)
cmd/createmoderator     crée un modérateur (mot de passe par HEROS_MODERATOR_PASSWORD)
migrations/             SQL versionné par horodatage (golang-migrate), embarqué dans le binaire
openapi/openapi.yaml    contrat de l'API (référence)
internal/
  config/               TOUTE la configuration : variables d'environnement, valeurs par défaut,
                        validation (plus stricte en production)
  server/               assemblage : dépendances, routes, middlewares, tâches de fond, arrêt propre
  auth/                 mots de passe (argon2id), sessions, CSRF, jetons, endpoints /auth/*
  user/                 comptes ; chiffrement des données personnelles dans le dépôt
  hero/                 demande du statut de super-héros
  moderation/           actions des modérateurs sur les comptes
  crypto/               cascade AES-256-GCM puis XChaCha20-Poly1305 (clé de 512 bits)
  mail/                 construction MIME, SMTP, file d'envoi asynchrone, modèles
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
  route `/api/*` vers l'API et le reste vers Next.js.
- **Session** : cookie `HttpOnly` posé par `POST /auth/login` (`__Host-session` en production,
  `session` en développement). Le serveur Next relaie le `Set-Cookie` au navigateur et l'en-tête
  `Cookie` à l'API ; l'API n'accepte **que** le cookie (jamais `Authorization: Bearer`).
- **CSRF** : toute écriture authentifiée porte `X-CSRF-Token` (valeur de `csrfToken`, renvoyée par
  `/auth/login` et `/auth/me`).
- **Inscription** : `POST /auth/register` ne connecte pas l'utilisateur. Il faut activer le compte
  (lien reçu par mail, route `HEROS_ACTIVATION_PATH` du frontend, jeton dans `?token=`) puis se
  connecter.
- **Rôles** : `citizen`, `hero`, `moderator` (le frontend actuel utilise `citoyen` et `super-heros`).
- **2FA** : prévue en version 2 ; une session est complète dès la connexion.

## Environnements Docker

Un fichier de base (`docker-compose.yml`) décrit ce qui est commun ; `docker-compose.dev.yml` et
`docker-compose.prod.yml` s'y ajoutent. Chaque environnement est un projet Compose distinct
(`heros-sos-dev`, `heros-sos-prod`) : leurs conteneurs et leurs volumes ne se mélangent jamais.

| | Développement (`make up-dev`) | Production (`make up-prod`) |
|---|---|---|
| Ports publiés | proxy `8088`, API `8081`, Mailpit `8025`, PostgreSQL `5432`, RabbitMQ — **sur `127.0.0.1` uniquement** | **proxy seul** : 80 et 443 |
| Front | serveur de développement Next.js, code monté et rechargé à chaud | image de production (`next start`), sans dépendances de développement |
| TLS | non (HTTP) | automatique (Let's Encrypt, via Caddy) |
| Secrets | mots de passe publics de `.env.example` | fichiers dans `secrets/`, montés en `/run/secrets` |
| Migrations | appliquées par l'API au démarrage | tâche `migrate` qui doit réussir avant l'API |
| Mails | Mailpit (rien ne sort de la machine) | SMTP réel (`.env.production`) |
| Réseau `backend` | ouvert | **interne** : la base et l'API n'ont aucun accès à Internet |

Ce qui est commun aux deux : l'API tourne sans root (`USER 65532`), système de fichiers en lecture
seule, toutes les capacités Linux retirées, `no-new-privileges`, journaux tournants, et un
**healthcheck** : `/api -healthcheck` interroge `/api/v1/ready` (base comprise) depuis l'intérieur du
conteneur. Une image `FROM scratch` n'a ni shell ni `curl` : la sonde est donc dans le binaire du
serveur (`cmd/api`, `internal/server/probe.go`), délai réglable par `HEROS_HEALTHCHECK_TIMEOUT`.

**Premier déploiement en production**

```bash
cp .env.production.example .env.production   # DOMAIN, ACME_EMAIL, SMTP… (aucun secret ici)
make secrets                                  # génère secrets/ (ne remplace jamais un secret existant)
make up-prod
make moderator-prod EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ
```

`secrets/` et `.env.production` ne sont jamais versionnés. Les secrets ne passent pas par
l'environnement des conteneurs : l'API lit `HEROS_<NOM>_FILE` (chemin d'un fichier) à la place de
`HEROS_<NOM>`. Définir les deux est une erreur, et le contenu d'un fichier n'apparaît jamais dans
un message d'erreur. **`secrets/crypto_keys` se sauvegarde hors du serveur.**

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
- Les variables des conteneurs (`POSTGRES_*`, `RABBITMQ_*`, ports, adresses) sont documentées dans
  `.env.example` et `.env.production.example`.

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
  changer `HEROS_CRYPTO_ACTIVE_KEY_ID` (les anciennes valeurs restent lisibles).
