# heros-sos

Plateforme web qui rapproche les citoyens et les super-héros pour signaler et traiter les incidents causés par des super-vilains.

## Structure du dépôt

| Dossier | Contenu |
|---|---|
| `frontend/` | Application Next.js 16 (App Router, React 19, TypeScript strict) avec Tailwind CSS v4 |
| `backend/` | API REST en Go (GORM) avec PostgreSQL 18, contrat OpenAPI dans `backend/openapi/` |
| `.github/workflows/` | CI du frontend et du backend |

## Prérequis

- **Node.js** ≥ 22 (version utilisée par la CI) et **npm** ≥ 10
- **Go** : version fixée par `backend/go.mod` (1.26)
- **Docker** : pour PostgreSQL et la pile de développement complète
- **openssl** : pour générer les secrets du backend (`make env`)

## Installation et lancement

### Tout dans Docker (recommandé)

```bash
cd backend
make env      # crée .env avec des secrets aléatoires
make up-dev   # proxy, front, API, PostgreSQL, Mailpit, RabbitMQ
```

| Service | Adresse |
|---|---|
| Front + API (via le proxy) | <http://localhost:8088> |
| Documentation de l'API (Swagger) | <http://localhost:8088/api/v1/docs> |
| API en direct | <http://localhost:8081/api/v1> |
| Mails reçus (Mailpit) | <http://localhost:8025> |
| Adminer | <http://localhost:8082> |

Le code du front est monté dans le conteneur et rechargé à chaud. Pour créer le premier modérateur :

```bash
make moderator-dev EMAIL=modo@example.org FIRST=Nick LAST=Fury BIRTH=1950-12-04
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

Les commandes, les tests et la configuration sont détaillés dans [le README du backend](backend/README.md) (`make help` liste toutes les cibles).

## État du branchement front ↔ API

Quand `API_URL` est renseignée, **inscription, activation, connexion, session, déconnexion et réinitialisation du mot de passe** passent par l'API. Tout le reste fonctionne encore sur les mocks côté frontend. Le backend expose désormais `GET /heroes` et `GET /heroes/{heroId}`, mais les fiches héroïques du frontend ne sont pas encore branchées. Le détail, domaine par domaine, est tenu à jour dans [frontend/docs/backend-contract.md](frontend/docs/backend-contract.md).

## Variables d'environnement du frontend

Copier `frontend/.env.example` vers `frontend/.env.local` (ignoré par Git), puis redémarrer `npm run dev` après chaque modification.

| Variable | Rôle |
|---|---|
| `API_URL` | URL de base de l'API REST, préfixe compris : `http://localhost:8081/api/v1` avec `make up-dev`, `http://localhost:8080/api/v1` avec `make run`. Lue par le serveur Next uniquement. Vide : tout le frontend tourne sur les mocks. |
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

## Liens utiles

- **Maquettes Figma** : à compléter
- **Contrat de l'API** : [backend/openapi/openapi.yaml](backend/openapi/openapi.yaml)
- **Environnement de recette** : à compléter
