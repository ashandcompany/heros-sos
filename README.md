# heros-sos

Plateforme web qui rapproche les citoyens et les super-héros pour signaler et traiter les incidents causés par des super-vilains.

## Structure du dépôt

| Dossier     | Contenu                                                         |
| ----------- | --------------------------------------------------------------- |
| `frontend/` | Application Next.js (App Router, TypeScript) + Tailwind CSS v4  |
| backend     | API REST (pas encore dans ce dépôt, données mockées en attendant) |

## Prérequis

- **Node.js** ≥ 20 (développé avec Node 24)
- **npm** ≥ 10 (livré avec Node ; le dépôt utilise `package-lock.json`)
- Docker : non requis pour l'instant (aucun `Dockerfile` ni `docker-compose`)

## Installation et lancement

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

L'application est disponible sur <http://localhost:3000>.

Tant que l'API n'est pas branchée, toutes les données viennent de `frontend/src/mocks/` : aucune configuration supplémentaire n'est nécessaire.

## Variables d'environnement

Copier `frontend/.env.example` vers `frontend/.env.local` (ce dernier est ignoré par Git).

| Variable              | Rôle                                                                                                                       |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL` | URL de base de l'API REST (ex : `http://localhost:8080/api/v1`). Inutilisée tant que les données sont mockées. Exposée au navigateur (préfixe `NEXT_PUBLIC_`) : n'y mettre aucun secret. |

## Scripts disponibles

À lancer depuis `frontend/` :

| Script                 | Rôle                                      |
| ---------------------- | ----------------------------------------- |
| `npm run dev`          | Serveur de développement                  |
| `npm run build`        | Build de production                       |
| `npm run start`        | Sert le build de production               |
| `npm run lint`         | ESLint                                    |
| `npm run format`       | Formate le code avec Prettier             |
| `npm run format:check` | Vérifie le formatage (utile en CI)        |

Il n'y a pas encore de script de test.

## Organisation du frontend

- `src/app/` : routes, layout racine, Server Actions (`actions/`)
- `src/components/layout|home|ui/` : composants (Server Components par défaut)
- `src/lib/api/` : couche d'accès aux données, seule à connaître les mocks
- `src/mocks/` : données fictives
- `src/types/` : types métier
- `src/app/globals.css` (bloc `@theme`) : design tokens Tailwind

Pour brancher l'API, voir [frontend/README.md](frontend/README.md#brancher-lapi-go).

## Liens

- Maquettes Figma : _à compléter_
- Backend / API : _à compléter_
- Environnement de recette : _à compléter_
