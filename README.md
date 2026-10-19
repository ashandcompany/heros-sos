# heros-sos

Plateforme web qui rapproche les citoyens et les super-héros pour signaler et traiter les incidents causés par des super-vilains.

## Structure du dépôt

| Dossier | Contenu |
|---|---|
| `frontend/` | Application Next.js (App Router, TypeScript) avec Tailwind CSS v4 |
| `backend/` | API REST développée en Go, avec MySQL comme base de données |

## Prérequis

- **Node.js** ≥ 20 (développé avec Node.js 24)
- **npm** ≥ 10 (fourni avec Node.js)
- **Go** : version à préciser selon le `go.mod`
- **MySQL** : version à préciser selon la configuration du backend

## Installation et lancement

### Frontend

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

L'application est disponible à l'adresse suivante : <http://localhost:3000>.

Tant que l'API n'est pas connectée, les données du frontend proviennent de `frontend/src/mocks/`. Aucune configuration supplémentaire n'est nécessaire pour les utiliser.

### Backend

Le backend est développé en Go et utilise MySQL. Consulte le `README.md` du backend et sa configuration pour connaître les commandes de démarrage et les paramètres de connexion à la base de données.

## Variables d'environnement

Copier `frontend/.env.example` vers `frontend/.env.local`. Ce dernier fichier est ignoré par Git.

| Variable | Rôle |
|---|---|
| `NEXT_PUBLIC_API_URL` | URL de base de l'API REST (ex. : `http://localhost:8080/api/v1`). Cette variable n'est pas encore utilisée tant que les données proviennent des mocks. Son préfixe `NEXT_PUBLIC_` la rend accessible dans le navigateur : ne pas y placer de secret. |

## Scripts disponibles

À exécuter depuis le dossier `frontend/` :

| Script | Rôle |
|---|---|
| `npm run dev` | Lance le serveur de développement |
| `npm run build` | Génère le build de production |
| `npm run start` | Démarre le serveur de production |
| `npm run lint` | Analyse le code avec ESLint |
| `npm run format` | Formate le code avec Prettier |
| `npm run format:check` | Vérifie le formatage, notamment en CI |

Aucun script de test n'est actuellement configuré dans le frontend.

## Organisation du frontend

- `src/app/` : routes, layout racine et Server Actions (`actions/`)
- `src/components/layout/`, `src/components/home/` et `src/components/ui/` : composants de l'interface, Server Components par défaut
- `src/lib/api/` : couche d'accès aux données, centralisant l'utilisation de l'API et des mocks
- `src/mocks/` : données fictives utilisées en développement
- `src/types/` : types métier
- `src/app/globals.css` : styles globaux et design tokens Tailwind, définis dans le bloc `@theme`

Pour connecter le frontend à l'API Go, consulter [la documentation du frontend](frontend/README.md#brancher-lapi-go).

## Liens utiles

- **Maquettes Figma** : à compléter
- **Backend / API** : à compléter
- **Environnement de recette** : à compléter
