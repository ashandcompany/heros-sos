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

## Brancher l'API Go

Renseigner `NEXT_PUBLIC_API_URL`, puis remplacer dans chaque fonction de `src/lib/api/`
l'appel à `mockResponse()` par un `fetch` vers l'endpoint indiqué en commentaire.
Les composants n'ont pas à être modifiés.
