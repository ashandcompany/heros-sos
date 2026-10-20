# Heros-SOS — Frontend

Next.js (App Router, TypeScript strict) + Tailwind CSS v4.

## Démarrer

```bash
cp .env.example .env.local
npm install
npm run dev
```

| Script                 | Rôle                          |
| ---------------------- | ----------------------------- |
| `npm run dev`          | Serveur de développement      |
| `npm run build`        | Build de production           |
| `npm run lint`         | ESLint                        |
| `npm run format`       | Formate le code avec Prettier |
| `npm run format:check` | Vérifie le formatage (CI)     |

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

| Variable              | Obligatoire          | Défaut                          | Lue par                                   | Rôle                                                                             |
| --------------------- | -------------------- | ------------------------------- | ----------------------------------------- | -------------------------------------------------------------------------------- |
| `NEXT_PUBLIC_API_URL` | Non (pour l'instant) | vide → données mockées          | `src/lib/api/config.ts`                   | Base de l'API Go **avec son préfixe**, ex : `http://localhost:8080/api/v1`       |
| `PHOTON_URL`          | Non                  | `https://photon.komoot.io/api/` | `src/lib/geocoding.ts`                    | Géocodeur Photon (OpenStreetMap) de l'autocomplétion d'adresses                  |
| `NODE_ENV`            | Non                  | défini par Next                 | `lib/cookies.ts`, `actions/newsletter.ts` | `production` active le drapeau `secure` des cookies. Ne pas le définir à la main |

- `NEXT_PUBLIC_*` est **embarqué dans le JavaScript envoyé au navigateur** : n'y mettre aucun
  secret. Elle est aujourd'hui lue uniquement côté serveur ; un renommage en `API_URL` éviterait
  de l'exposer inutilement.
- Vide ou absente, `NEXT_PUBLIC_API_URL` laisse toute l'application sur les mocks. Une fois les
  fonctions branchées (voir [docs/backend-contract.md](docs/backend-contract.md)), elle deviendra
  obligatoire.
- `PHOTON_URL` : l'instance publique de Komoot est soumise à un usage raisonnable ; en
  production, pointer vers une instance dédiée (ex : `http://photon:2322/api/`).
- Côté backend, `HEROS_ALLOWED_ORIGINS` doit contenir l'origine du frontend
  (`http://localhost:3000` en développement), sinon les écritures sont refusées
  (voir [backend/.env.example](../backend/.env.example)).

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

Renseigner `NEXT_PUBLIC_API_URL`, puis remplacer dans chaque fonction de `src/lib/api/`
l'appel à `mockResponse()` par un `fetch` vers l'endpoint indiqué en commentaire.
Les composants n'ont pas à être modifiés.
