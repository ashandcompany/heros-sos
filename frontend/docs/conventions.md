# Conventions du frontend

Règles de travail du projet. Pour le fonctionnement d'ensemble, voir
[architecture.md](./architecture.md) ; pour les composants, [ui.md](./ui.md).

## Next.js : lire la doc de la version installée

Le projet utilise Next 16 avec `cacheComponents` et `partialPrefetching` activés
([next.config.ts](../next.config.ts)). Les API et conventions diffèrent de ce qu'on trouve dans
la plupart des tutoriels. Avant d'écrire du code qui touche au rendu, au cache, aux routes ou
au proxy, lire le guide correspondant dans `node_modules/next/dist/docs/` (voir
[AGENTS.md](../AGENTS.md)) et tenir compte des avis de dépréciation.

Conséquence pratique de `cacheComponents` : tout ce qui est dynamique (cookies, données de
session, lecture de `lib/api/` dépendante de l'utilisateur) doit se trouver sous un `<Suspense>`.
Les `page.tsx` restent donc minces (voir [architecture.md](./architecture.md#rendu--server-components-et-suspense)).

## Server Components ou Client Components ?

Par défaut, **tout est Server Component**. Ajouter `"use client"` seulement si le composant a
besoin d'au moins un de ces éléments :

- état local ou effets (`useState`, `useEffect`, `useRef`) ;
- événements navigateur (`onClick`, clavier, `matchMedia`, `localStorage`) ;
- `useActionState` (formulaires) ou `useFormStatus` (`SubmitButton`) ;
- une librairie purement client (éditeur Tiptap).

Règles associées :

- Un composant client **n'importe jamais `@/lib/api`** ni `server-session` : les données lui
  arrivent par props ou par une Server Action.
- **Garder le `"use client"` le plus bas possible** dans l'arbre : extraire la partie
  interactive (un bouton, un champ) plutôt que de marquer toute une page.
- Un formulaire = un composant client qui appelle une action de `src/app/actions/`.
- Aucun secret ni jeton dans un composant client ou dans une variable `NEXT_PUBLIC_*`.

## Nommage et organisation

| Élément                | Convention                                           | Exemple                                   |
| ---------------------- | ---------------------------------------------------- | ----------------------------------------- |
| Composants             | `PascalCase.tsx`, un composant principal par fichier | `IncidentCard.tsx`                        |
| Dossiers de composants | Par espace ou domaine, pas par type                  | `components/hero/`, `components/admin/`   |
| Utilitaires, lib       | `kebab-case.ts`                                      | `server-session.ts`, `form-state.ts`      |
| Server Actions         | `xxxAction`, dans `app/actions/<domaine>.ts`         | `rateInterventionAction`                  |
| Types métier           | `src/types/<domaine>.ts`, réexportés par `index.ts`  | `import type { AuthUser } from "@/types"` |
| Dossiers de routes     | **En français**, `kebab-case`                        | `/encyclopedie`, `/mot-de-passe-oublie`   |
| Identifiants de code   | En anglais                                           | `getHeroIncident`, `declareIncident`      |

- Imports via l'alias `@/` (jamais de `../../..`).
- `import type` pour les types.
- Les données métier restent dans la langue du produit : valeurs d'énumération en français
  (`"citoyen"`, `"super-heros"`, `"temoin"`), libellés dans `lib/format.ts`.
- Les chemins de pages viennent toujours de `routes` ([src/lib/routes.ts](../src/lib/routes.ts)),
  jamais écrits en dur. Voir [architecture.md](./architecture.md#routes).
- Tout texte affiché à l'utilisateur est en français, sans bibliothèque d'internationalisation.
  Dates et nombres passent par `lib/format.ts` (`fr-FR`, fuseau `Europe/Paris`).

### Pages

- Chaque `page.tsx` exporte `metadata` avec un `title` court : le gabarit du layout ajoute
  « · Heros-SOS » (ou « · Back-office », « · Espace pro »).
- Les espaces `/admin`, `/heros` et `/profil` ne sont pas indexés (`robots: { index: false }` dans
  leur layout).
- Les props de routes dynamiques utilisent les types globaux de Next (`PageProps<"/blog/[slug]">`,
  `LayoutProps<"/heros">`).

## Style et design

- **Tailwind v4, tokens d'abord.** Utiliser les utilitaires issus du bloc `@theme`
  (`bg-surface`, `text-muted`, `border-edge`…). Pas de couleur hexadécimale dans un composant :
  elle ne suivrait pas le thème clair. Voir [ui.md](./ui.md#design-tokens).
  - Exceptions actuelles : un survol de bouton `danger` et la bordure des commandes du
    `Carousel` ([ButtonLink.tsx](../src/components/ui/ButtonLink.tsx),
    [Carousel.tsx](../src/components/ui/Carousel.tsx)). À migrer vers des tokens si on les touche.
- **Pas d'arrondis** (supprimés dans `@theme`), bordures épaisses et **ombres dures décalées**.
- Les classes se combinent avec `cn()`, pas de concaténation manuelle de chaînes.
- CSS global dans [globals.css](../src/app/globals.css) (tokens) et
  [retro.css](../src/app/retro.css) (effets décoratifs). N'y ajouter que du CSS qui ne s'exprime
  pas proprement en utilitaires.
- Les deux thèmes doivent être vérifiés à chaque changement visuel.

## Accessibilité

Exigence de base, pas une finition :

- Un `SkipLink` est présent dans le layout ; garder un seul `<h1>` par page et des titres
  hiérarchisés (`SectionHeading` produit des `<h2>`).
- Chaque champ a un label visible ; erreurs et aides sont reliées par `aria-describedby`
  (composants de [ui.md](./ui.md)).
- Tout se fait au clavier : préférer les éléments natifs (`<button>`, `<dialog>`, `<fieldset>`)
  aux `<div onClick>`.
- Cibles tactiles d'au moins 44 px.
- Les éléments décoratifs sont `aria-hidden` ; ne jamais y mettre d'information.
- Respecter `prefers-reduced-motion` pour toute animation.
- Les statuts ne reposent pas sur la couleur seule : un `Badge` contient toujours du texte.

## Formulaires, validation et messages

- Motif unique `FormState` + `useActionState` + Server Action
  ([architecture.md](./architecture.md#formulaires-et-server-actions)).
- Validation **côté serveur obligatoire** ; le HTML (`required`, `maxLength`, `type`) n'est
  qu'un confort.
- Un message d'erreur dit **quoi corriger**, en français, à la deuxième personne du pluriel
  (« Veuillez saisir… », « Choisissez une note de 1 à 5. »).
- Erreur d'authentification : message **identique** pour « compte inconnu » et « mauvais mot de
  passe » (ne pas révéler quels comptes existent). Même principe pour la réinitialisation du mot
  de passe.
- Erreur par champ → `fieldErrors` ; erreur globale → `message` affiché par `FormMessage`.
- Jamais de mot de passe dans `values`.

## Gestion des erreurs

- Fonctions de `lib/api/` : un échec attendu (identifiants invalides, e-mail déjà utilisé, rejet
  métier) est un **résultat** `{ ok: false, error }`, pas une exception. Une exception signale un
  bug ou une panne.
- Une ressource introuvable renvoie `null` côté `lib/api/`, et la page appelle `notFound()`.
- `app/error.tsx` (erreur inattendue, bouton « Réessayer ») et `app/not-found.tsx` (404, y
  compris `notFound()`) existent à la racine, sur le gabarit `StatusPage`. Pas de `loading.tsx` :
  les pages utilisent `<Suspense>` + squelettes. Une erreur dans le layout racine (en-tête) n'est
  pas couverte : il faudrait un `global-error.tsx`.

## Sécurité côté frontend

- Autorisation : `requireRole()` dans chaque action et chaque composant de données protégées ;
  le proxy ne suffit pas.
- HTML riche (articles de blog) : toujours passer par `sanitizeArticleHtml`
  ([src/lib/sanitize.ts](../src/lib/sanitize.ts)) avant affichage. N'utiliser
  `dangerouslySetInnerHTML` que pour une sortie de cette fonction (ou, dans le layout racine,
  pour le script anti-flash du thème, qui est une constante du code).
- Cookies d'identification : `httpOnly`, `sameSite: lax`, `secure` en production.
- Fichiers envoyés : taille vérifiée (`AVATAR_MAX_BYTES`, 2 Mo) ; la limite des Server Actions
  (`bodySizeLimit: "3mb"`) doit rester supérieure à la plus grande pièce jointe autorisée.

## Code, formatage et vérifications

- **Prettier** décide du format : 100 colonnes, guillemets doubles, points-virgules, virgules
  finales ([.prettierrc.json](../.prettierrc.json)). Lancer `npm run format` avant de commiter.
- **ESLint** (`eslint-config-next` + TypeScript) : `npm run lint`.
- TypeScript en mode strict : pas de `any`, pas de `@ts-ignore` sans commentaire expliquant
  pourquoi.
- **Commentaires** : en français, ils expliquent le **pourquoi** (contrainte, choix, piège),
  pas ce que le code dit déjà. Les fonctions de `lib/api/` documentent l'endpoint futur.
  Un composant exporté a un commentaire `/** … */` d'une ou deux lignes.
- La CI ([frontend-ci.yml](../../.github/workflows/frontend-ci.yml)) exécute sur chaque PR vers
  `develop` : audit des dépendances (high/critical), `prettier --check`, `npm run lint` (aucun
  avertissement toléré : `next build` ne lance plus ESLint depuis Next 16), `npm test`, types de
  l'API à jour, `npm run build`, puis les parcours Playwright dans un job séparé.

## Tests

- **Unitaires** (Vitest, `npm test`, `npm run test:watch`) : fichiers `*.test.ts` à côté du
  module testé, pour le code pur de `src/lib/` (validation, formatage, pagination). Les règles
  de validation doivent rester celles du backend : un test fixe chaque borne.
- **Conversions de l'API** : les `fromApi*` restent privées, on les teste à travers les fonctions
  publiques avec l'API activée et `fetch` simulé, grâce à
  [src/lib/api/test-utils.ts](../src/lib/api/test-utils.ts) (`withApi`, `mockFetch`,
  `jsonResponse`). Voir `blog.test.ts` comme modèle.
- **Parcours** (Playwright, `npm run test:e2e`, dossier `e2e/`) : build de production en mode
  mock sur le port 3100 (jamais un `npm run dev` branché sur l'API). Les mocks étant partagés
  en mémoire, les parcours tournent l'un après l'autre et ne supposent pas un état neuf (ils
  prennent « le premier incident en attente », pas un identifiant fixe). Sélecteurs par rôle
  et par libellé accessible (`getByRole`, `getByLabel`). Première fois :
  `npx playwright install chromium`.
- Limite connue du mode mock : `proxy.ts` est un bundle à part, avec sa propre copie des mocks.
  Un compte créé par l'inscription ne passe donc pas le proxy (`/profil` renvoie à la
  connexion) ; les parcours utilisent les comptes de `src/mocks/users.ts`.

## Git

- Branches : `feat/<sujet>`, `docs/<sujet>` (ex : `feat/hero-space`, `feat/character-sheets`),
  fusionnées dans `develop` par pull request.
- Messages de commit en [Conventional Commits](https://www.conventionalcommits.org/) :
  `feat:`, `fix:`, `docs:`, `style:`, `refactor:`, `ci:`, avec une portée facultative
  (`feat(docker):`, `docs(openapi):`).
- Un commit = un changement cohérent ; le formatage de masse va dans un commit `style:` séparé.
