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
