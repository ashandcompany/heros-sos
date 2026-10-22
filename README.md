# heros-sos

Plateforme web qui rapproche les citoyens et les super-héros pour signaler et traiter les incidents causés par des super-vilains.

## Structure du dépôt

| Dossier | Contenu |
|---|---|
| `frontend/` | Application Next.js 16 (App Router, React 19, TypeScript strict) avec Tailwind CSS v4 |
| `backend/` | API REST en Go (GORM) avec PostgreSQL 18, contrat OpenAPI dans `backend/openapi/` |
| `docs-site/` | Site de documentation Docusaurus (son propre `package.json`, tourne dans Docker) : architecture, MVP, stack et tables du backend, contrat d'API généré depuis l'OpenAPI |
| `docker-compose*.yml`, `Makefile` | Piles Docker de développement et de production (back, front, documentation) et leurs raccourcis (`make help`) |
| `.github/workflows/` | CI du frontend, du backend et de la documentation |

## Prérequis

- **Docker** : pour la pile de développement complète (`make up`). Prévoir de la place pour sa machine virtuelle : **16 Go de disque recommandés** (Docker Desktop, Settings → Resources), car la pile, ses images et le cache de build dépassent 8 Go. Avec un disque plein, les builds échouent et PostgreSQL ne peut plus écrire.
- **openssl** : pour générer les secrets du backend (`make env`, lancé par `make up`)
- **Node.js** ≥ 22 (version utilisée par la CI) et **npm** ≥ 10 : seulement pour travailler sur le frontend hors Docker. `make up` n'en a pas besoin, la documentation non plus.
- **Go** : version fixée par `backend/go.mod` (1.26), seulement pour lancer l'API hors Docker

## Installation et lancement

### Tout dans Docker (recommandé)

```bash
make up   # proxy, front, API, PostgreSQL, Mailpit, RabbitMQ, documentation (crée backend/.env au premier lancement)
```

Les fichiers `docker-compose*.yml` sont à la racine : un seul endroit pour lancer le back, le front et la documentation. Les commandes ci-dessous délèguent à `backend/Makefile`, qui reste utilisable directement (`cd backend && make up-dev`). L'environnement (`.env`, `.env.production`) et les secrets restent dans `backend/`.

| Commande | Effet |
|---|---|
| `make up` | Démarre tout en développement : proxy, front, API, PostgreSQL, Mailpit, RabbitMQ, documentation. Crée `backend/.env` au premier lancement et vérifie d'abord l'espace libre de Docker (1,5 Go minimum, `FORCE=1` pour passer outre). |
| `make down` | Arrête la pile ; les données sont conservées. |
| `make logs`, `make ps` | Suit les journaux de tous les conteneurs ; liste les conteneurs, leur état et leurs ports. |
| `make docs` | Démarre seulement la documentation (sans le reste de la pile). |
| `make adminer` | Ajoute Adminer, un client SQL. |
| `make moderator EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ` | Crée un modérateur (mot de passe demandé, jamais passé en argument). |
| `make resetmfa EMAIL=…` | Réinitialise la double authentification d'un compte qui a perdu son appareil, **en développement** (vérifier son identité d'abord). En production : `make resetmfa-prod EMAIL=…` depuis `backend/`. |
| `make elk`, `make elk-down` | Monitoring ELK (profil Compose `elk` : Elasticsearch, Logstash, Kibana, Filebeat) : les journaux JSON de l'API, le p95 par route, la chronologie des événements, le délai de synchronisation. Kibana : <http://localhost:5601>. Demande 6 à 8 Go de mémoire pour Docker. |
| `make perf-prepare`, `make seed-perf`, `make perf-run`, `make perf`, `make perf-indexes-off` / `-on` | Mesures sur 5 millions d'incidents, dans une base dédiée `herossos_perf` : `perf-prepare` la migre et y charge les données Marvel, `seed-perf` ajoute les incidents (`PERF_MAX_SIZE_GB=8` borne sa taille, 7 Go environ), `perf` échoue si le p95 dépasse 350 ms, `perf-indexes-off` / `-on` rejouent l'avant/après. Voir [la doc](backend/stack/performance-et-synchronisation.md). |
| `make seed` | Remplit la base avec des données de démonstration Marvel : modérateurs, héros, citoyens, vilains, incidents, interventions, avis, films, articles. Tous les comptes ont le mot de passe `Avengers2026!` (ex. `admin@heros-sos.local`, `peter.parker@heros-sos.local`, `mary.jane.watson@heros-sos.local`). Refusé sur une base déjà remplie ; `RESET=1` **efface d'abord toutes les données**. Refusé en production. |
| `make resetmfa EMAIL=…` | Réinitialise la double authentification d'un compte qui a perdu son appareil (vérifier son identité d'abord). |
| `make reset` | Arrête la pile **et supprime ses données** (base, photos, RabbitMQ), après confirmation. |
| `make secrets`, `make up-prod`, `make down-prod`, `make logs-prod` | Production : voir [le README du backend](backend/README.md#environnements-docker). |
| `make check` | Valide la syntaxe des fichiers Compose (développement et production). |

Tous les ports sont liés à `127.0.0.1` (rien n'est exposé sur le réseau) et se changent dans `backend/.env` (`PROXY_PORT`, `API_PORT`, `MAILPIT_PORT`, `ADMINER_PORT`, `POSTGRES_PORT`, `RABBITMQ_PORT`, `RABBITMQ_UI_PORT`, `DOCS_PORT`).

| Service | Adresse |
|---|---|
| Front + API (via le proxy) | <http://localhost:8088> |
| Documentation de l'API (Swagger) | <http://localhost:8088/api/v1/docs> |
| API en direct | <http://localhost:8081/api/v1> |
| Documentation technique (Docusaurus) | <http://localhost:3001/heros-sos/> |
| Mails reçus (Mailpit) | <http://localhost:8025> |
| Adminer (après `make adminer`) | <http://localhost:8082> |
| RabbitMQ (interface de gestion) | <http://localhost:15672> |
| PostgreSQL | `localhost:5432` |

Le code du front et celui de la documentation sont montés dans leurs conteneurs et rechargés à chaud. En production, seul le proxy publie des ports (80 et 443). Pour créer le premier modérateur :

```bash
make moderator EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
```

> **Sauvegardez `HEROS_CRYPTO_KEYS`** (dans `backend/.env`) : sans cette clé, les données personnelles chiffrées en base sont définitivement illisibles.

### Frontend seul

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

L'application est disponible sur <http://localhost:3000>. Sans `API_URL`, toutes les données proviennent de `frontend/src/mocks/` (comptes de test dans `src/mocks/users.ts`, ex. `citoyen@heros-sos.fr` / `Citoyen123!`).

### Backend seul

```bash
cd backend
make db-start   # PostgreSQL 18 dans Docker
make env
make migrate
make run        # API sur http://localhost:8080/api/v1
```

Les commandes, les tests et la configuration sont détaillés dans [le README du backend](backend/README.md) (`make help` liste toutes les cibles). Les documents de conception du backend sont `backend/MVP.md` (périmètre), `backend/STACK.md` (choix techniques, une page par sujet dans `backend/stack/`) et `backend/TABLES.md` (schéma de la base, une page par domaine dans `backend/tables/`).

## Documentation technique

Le site `docs-site/` (Docusaurus) rassemble la documentation technique : l'architecture en schémas, le MVP, les choix techniques, le schéma de la base de données et le contrat d'API (une page par route, générée depuis `backend/openapi/openapi.yaml`). Il tourne dans Docker : <http://localhost:3001/heros-sos/>, démarré par `make up` (ou seul, par `make docs`) et rechargé à chaud.

Rien n'est copié : les documents de `backend/` sont lus tels quels, donc le site ne dérive pas de ce qui est livré. Il n'est pas encore publié en ligne. Détails, dont l'ajout d'une page : [docs-site/README.md](docs-site/README.md).

## Journaux

`make logs` suit tous les conteneurs (`make logs-prod` en production). L'API écrit en texte en développement et en JSON en production, au niveau `HEROS_LOG_LEVEL` (`debug`, `info`, `warn` ou `error`). Chaque conteneur garde au plus `LOG_MAX_FILE` fichiers de `LOG_MAX_SIZE` (3 × 10 Mo par défaut), réglables dans `backend/.env` : sans cette limite, Docker les garderait indéfiniment.

## État du branchement front ↔ API

Quand `API_URL` est renseignée, **inscription, activation, connexion, session, déconnexion et réinitialisation du mot de passe** passent par l'API. Tout le reste fonctionne encore sur les mocks côté frontend. Le backend expose désormais `GET /heroes` et `GET /heroes/{heroId}`, mais les fiches héroïques du frontend ne sont pas encore branchées. Le détail, domaine par domaine, est tenu à jour dans [frontend/docs/backend-contract.md](frontend/docs/backend-contract.md).

## Variables d'environnement du frontend

Copier `frontend/.env.example` vers `frontend/.env.local` (ignoré par Git), puis redémarrer `npm run dev` après chaque modification.

| Variable | Rôle |
|---|---|
| `API_URL` | URL de base de l'API REST, préfixe compris : `http://localhost:8081/api/v1` avec `make up`, `http://localhost:8080/api/v1` avec `make run`. Lue par le serveur Next uniquement. Vide : tout le frontend tourne sur les mocks. |
| `MAP_TILE_URL` | Fond de la carte des incidents. Vide : tuiles publiques OpenStreetMap, réservées au développement. |
| `MAP_TILE_ATTRIBUTION` | Mention de la source des tuiles, affichée sur la carte. |
| `MAP_TILE_MAX_ZOOM` | Zoom maximal des tuiles. |

Les variables du backend sont décrites dans `backend/.env.example`.

## Scripts du frontend

À exécuter depuis `frontend/` :

| Script | Rôle |
|---|---|
| `npm run dev` | Lance le serveur de développement |
| `npm run build` | Génère le build de production |
| `npm run start` | Démarre le serveur de production |
| `npm run lint` | Analyse le code avec ESLint |
| `npm run format` | Formate le code avec Prettier |
| `npm run format:check` | Vérifie le formatage, notamment en CI |

Aucun test automatisé n'est configuré dans le frontend. Côté backend : `make test` (après `make db-start`) et `make check` pour tout ce que vérifie la CI.

## Organisation du frontend

- `src/app/` : routes (accueil, carte, héros, encyclopédie, vidéothèque, blog, FAQ, contact, à propos, pages légales, authentification, profil, back-office `admin/`), layout racine et Server Actions (`actions/`)
- `src/components/` : composants rangés par domaine (`layout/`, `ui/`, `home/`, `map/`, `admin/`…), Server Components par défaut
- `src/lib/api/` : couche d'accès aux données, seule à connaître l'API et les mocks
- `src/lib/validation/` : validation des formulaires
- `src/mocks/` : données fictives utilisées en développement
- `src/types/` : types métier
- `src/app/globals.css` : styles globaux et design tokens Tailwind (bloc `@theme`)

Documentation détaillée dans `frontend/docs/` : [architecture](frontend/docs/architecture.md), [conventions](frontend/docs/conventions.md), [interface](frontend/docs/ui.md) et [contrat avec le backend](frontend/docs/backend-contract.md).

## Intégration continue

- **Frontend** (`frontend-ci.yml`) : installation, audit des dépendances, vérification Prettier, build.
- **Backend** (`backend-ci.yml`) : vérification de `go.mod`/`go.sum`, recherche de vulnérabilités (`govulncheck`), tests avec PostgreSQL, build.
- **Documentation** (`docs-ci.yml`) : build du site `docs-site/` ; un lien cassé ou un OpenAPI illisible fait échouer le job. Rien n'est publié.
- **Publication de la documentation** (`docs-deploy.yml`) : mêmes étapes, puis déploiement sur GitHub Pages à chaque push sur `develop`. Il reste inactif tant que Pages n'est pas activé dans les réglages du dépôt : voir [docs-site/README.md](docs-site/README.md#intégration-continue-et-déploiement).
- **Relecteurs** : `.github/CODEOWNERS` désigne qui relire selon les fichiers touchés (frontend, backend, CI et Docker, `docs-site/`).

La CI ne construit pas encore les images Docker et ne vérifie pas les fichiers Compose : lancer `make check` (syntaxe Compose) avant de pousser une modification de `docker-compose*.yml`.

## Liens utiles

- **Maquettes Figma** : à compléter
- **Contrat de l'API** : [backend/openapi/openapi.yaml](backend/openapi/openapi.yaml)
- **Documentation technique** : <http://localhost:3001/heros-sos/> (après `make up` ou `make docs`), voir [docs-site/README.md](docs-site/README.md)
- **Environnement de recette** : à compléter
