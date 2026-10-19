# Backend Heros-SOS

API REST en Go (PostgreSQL, GORM). Cette partie couvre l'**identité et la sécurité** : comptes,
sessions, demande du statut de super-héros, modération des comptes, chiffrement des données
personnelles, mails.

## Démarrage rapide

Prérequis : Go (voir `go.mod`), Docker (pour PostgreSQL), `openssl`.

```bash
cd backend
make db-start        # PostgreSQL dans Docker (bases herossos et herossos_test)
make env             # crée .env avec des secrets aléatoires
make migrate         # applique les migrations
make run             # API sur http://localhost:8080/api/v1
```

Créer le premier modérateur (il valide ensuite les super-héros) :

```bash
make createmoderator EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
```

En développement, les mails ne sont pas envoyés : le driver `log` les écrit dans la sortie du
serveur, avec les liens d'activation et de réinitialisation.

> **Sauvegardez `HEROS_CRYPTO_KEYS`.** Sans cette clé, les données personnelles chiffrées en base
> sont définitivement illisibles.

## Tests

```bash
make db-start   # une fois
make test       # go test ./... -race -count=1
make check      # tout ce que vérifie le CI : formatage, vet, tests, vulnérabilités
```

Les tests d'intégration utilisent un vrai PostgreSQL (`HEROS_TEST_DATABASE_URL`, défini par le
Makefile) et chacun travaille dans son propre schéma : ils tournent en parallèle sans se gêner. En
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

## Production

- `HEROS_ENV=production` active des contrôles stricts au démarrage : cookie `Secure`, SMTP chiffré,
  base en TLS, origines en HTTPS, paramètres argon2id minimaux. Une configuration incohérente
  empêche le démarrage.
- Derrière un reverse proxy, renseigner `HEROS_TRUSTED_PROXIES` : sinon tous les clients partagent
  l'adresse IP du proxy, et donc les mêmes limites de débit.
- Le limiteur de débit est en mémoire : exact pour une seule instance de l'API.
- Les mails en file sont perdus si le processus est tué brutalement (RabbitMQ les rendra durables
  sans changer le code appelant : seul `mail.Mailer` est connu des autres paquets).
- Sauvegarder `HEROS_CRYPTO_KEYS` séparément de la base ; pour une rotation, ajouter une clé puis
  changer `HEROS_CRYPTO_ACTIVE_KEY_ID` (les anciennes valeurs restent lisibles).
