# Heros-SOS — Frontend

Next.js (App Router, TypeScript strict) + Tailwind CSS v4.

## Démarrer

```bash
cp .env.example .env.local
npm install
npm run dev
```

| Script                    | Rôle                                                                     |
| ------------------------- | ------------------------------------------------------------------------ |
| `npm run dev`             | Serveur de développement                                                 |
| `npm run build`           | Build de production                                                      |
| `npm run lint`            | ESLint                                                                   |
| `npm run format`          | Formate le code avec Prettier                                            |
| `npm run format:check`    | Vérifie le formatage (CI)                                                |
| `npm run api:types`       | Régénère `src/lib/api/schema.d.ts` depuis `backend/openapi/openapi.yaml` |
| `npm run api:types:check` | Vérifie que ces types sont à jour (CI)                                   |

Sans configuration, toute l'application tourne sur les mocks (comptes de test dans
[src/mocks/users.ts](src/mocks/users.ts), ex : `citoyen@heros-sos.fr` / `Citoyen123!`).

### Avec l'API Go

Identité (inscription, activation, connexion, session, mot de passe), demandes de super-héros,
modération des comptes, compte, types d'incident, incidents et interventions, vidéothèque, avis,
blog, contact et newsletter passent par l'API dès que `API_URL` est renseignée ; le reste reste
mocké (voir [docs/backend-contract.md](docs/backend-contract.md)).
La recherche de films IMDb exige une clé OMDb côté API (`HEROS_OMDB_API_KEY` dans
`backend/.env`). Après une mise à jour du backend, relancer `make up-dev` pour reconstruire
l'image de l'API (sinon les nouvelles routes répondent 404). Les comptes
sont alors ceux de l'API : à créer par `POST /auth/register`, puis à activer avec le lien reçu
par mail (Mailpit, `http://localhost:8025`, avec `make up`).

- **Tout dans Docker** (recommandé) : depuis la racine du dépôt, `make up` (crée `backend/.env` au
  premier lancement). Front et API derrière le même proxy : <http://localhost:8088>. `API_URL` est
  déjà fixée dans `docker-compose.dev.yml` (à la racine) ; le code du front est monté et rechargé à
  chaud.
- **Front sur la machine** : lancer l'API (`make up` à la racine, ou `go run` dans `backend/`), puis
  renseigner `API_URL` dans `.env.local` (`http://localhost:8081/api/v1` avec `make up`) et
  relancer `npm run dev`.

## Organisation

- `src/app/` : routes, layout racine, Server Actions (`actions/`)
- `src/components/layout|home|ui/` : composants (Server Components par défaut)
- `src/lib/api/` : couche d'accès aux données, seule à connaître les mocks
- `src/mocks/` : données fictives
- `src/types/` : types métier
- `src/app/globals.css` (bloc `@theme`) : design tokens Tailwind (couleurs, typos)

## Variables d'environnement

Copier `.env.example` vers `.env.local` (jamais versionné). Next.js ne charge les variables qu'au
démarrage : redémarrer `npm run dev` après modification.

| Variable   | Obligatoire | Défaut                 | Lue par                                   | Rôle                                                                                             |
| ---------- | ----------- | ---------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `API_URL`  | Non         | vide → données mockées | `src/lib/api/config.ts`                   | Base de l'API Go **avec son préfixe**, ex : `http://localhost:8081/api/v1`                       |
| `NODE_ENV` | Non         | défini par Next        | `lib/cookies.ts`, `actions/newsletter.ts` | `production` : cookies `secure`, cookie de session `__Host-session`. Ne pas le définir à la main |

- `API_URL` est lue **par le serveur, au démarrage** : elle n'est jamais envoyée au navigateur et
  n'est pas figée au build (la même image Docker sert dans tous les environnements). Dans la pile
  Docker, elle vaut `http://api:8080/api/v1` (réseau interne, voir `docker-compose.*.yml` à la racine).
- Le nom du cookie de session doit être celui de l'API : `session` en développement,
  `__Host-session` en production. Côté backend, c'est `HEROS_COOKIE_SECURE` (vrai par défaut en
  production) : un front en mode production face à une API de développement fait échouer la
  connexion avec une erreur explicite.
- L'autocomplétion des adresses n'a besoin d'aucune variable : elle interroge le service public
  de la Géoplateforme de l'IGN (Base Adresse Nationale), sans clé. Voir `src/lib/geocoding.ts`.

## Documentation détaillée

- [docs/architecture.md](docs/architecture.md) : rôles, session et accès, couche `lib/api`,
  formulaires et Server Actions, rendu serveur.
- [docs/ui.md](docs/ui.md) : composants `components/ui/`, accessibilité, design tokens.
- [docs/conventions.md](docs/conventions.md) : Server/Client Components, nommage, style,
  accessibilité, erreurs, sécurité, CI et Git.
- [docs/backend-contract.md](docs/backend-contract.md) : écarts entre le frontend et le contrat
  OpenAPI, checklist de branchement.

## Back-office (modération)

Espace `/admin`, réservé au rôle `admin` (compte mock : `admin@heros-sos.fr` / `Admin123!`).

| Page                  | Rôle                                                             |
| --------------------- | ---------------------------------------------------------------- |
| `/admin`              | Tableau de bord : éléments en attente dans chaque section        |
| `/admin/incidents`    | Valider / rejeter / corriger les incidents, en signaler un       |
| `/admin/comptes`      | Contrôler les nouveaux comptes (suspension), changements d'email |
| `/admin/super-heros`  | Valider / refuser les demandes de super-héros, référentiel       |
| `/admin/videotheque`  | Publier / refuser / supprimer les films des vidéothèques         |
| `/admin/commentaires` | Publier / masquer les commentaires des évaluations               |
| `/admin/blog`         | Créer, modifier, supprimer les articles (éditeur riche Tiptap)   |

Les listes à modérer affichent par défaut les éléments en attente (`?statut=valide|rejete|tous`
pour les autres). Le HTML des articles doit être assaini côté API avant tout affichage public.

## Brancher l'API Go

Plusieurs domaines sont déjà branchés ([src/lib/api/auth.ts](src/lib/api/auth.ts) et
[src/lib/api/videos.ts](src/lib/api/videos.ts) servent de modèles). Pour une autre fonction de `src/lib/api/` : appeler `apiFetch()`
([src/lib/api/http.ts](src/lib/api/http.ts)) quand `isApiEnabled`, garder le mock sinon, et
conserver la signature. Les composants n'ont pas à être modifiés. Checklist complète :
[docs/backend-contract.md](docs/backend-contract.md#checklist-de-branchement-dune-fonction).
