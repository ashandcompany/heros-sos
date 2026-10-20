# Architecture du frontend

Ce document explique **comment les pièces s'articulent**. Pour démarrer le projet, voir le
[README](../README.md) ; pour les composants d'interface, voir [ui.md](./ui.md).

## Vue d'ensemble

```
Navigateur
   │  requête
   ▼
src/proxy.ts ──────────────► redirige si l'accès n'est pas autorisé (rôle, visiteur)
   │
   ▼
src/app/**/page.tsx  (Server Components, rendus sous <Suspense>)
   │  lisent la session avec lib/server-session.ts
   │  lisent les données avec lib/api/
   ▼
src/components/**    (Server Components par défaut, "use client" seulement si nécessaire)
   │  <form action={…}>
   ▼
src/app/actions/*.ts (Server Actions)
   │  requireXxxSession() → validation (lib/validation/) → lib/api/
   ▼
src/lib/api/*.ts     (seule couche qui connaît les mocks et l'API Go)
   ▼                          ▼
src/mocks/*.ts         API REST Go, via lib/api/http.ts (si API_URL est renseignée :
(données fictives)     connexion, session, déconnexion aujourd'hui)
```

Tout le code qui touche aux données ou à la session s'exécute **côté serveur**. Les composants
clients (`"use client"`) ne gèrent que l'interaction (état local, clavier, `useActionState`).

## Rôles et espaces

Trois rôles (`UserRole`, dans [src/types/auth.ts](../src/types/auth.ts)) :

| Rôle          | Espace personnel                | Layout                      | Création du compte                       |
| ------------- | ------------------------------- | --------------------------- | ---------------------------------------- |
| `citoyen`     | `/profil`                       | `app/profil` (`citizenNav`) | Inscription libre                        |
| `super-heros` | `/heros` (→ `/heros/dashboard`) | `app/heros` (`heroNav`)     | Après validation de la demande par admin |
| `admin`       | `/admin`                        | `app/admin` (`adminNav`)    | Hors inscription publique                |

- La page d'accueil de chaque rôle est dans `accountHomeByRole` ([src/lib/routes.ts](../src/lib/routes.ts)).
- Les sous-navigations (`citizenNav`, `heroNav`, `adminNav`) sont aussi dans `routes.ts` et
  affichées par `components/layout/SpaceNav`.
- Les layouts d'espace posent `robots: { index: false }` : ces pages ne sont jamais indexées.
- Un compte `super-heros` n'existe qu'une fois sa demande validée (`reviewHeroApplication`) :
  avoir le rôle suffit donc à garantir un compte validé.

### Routes

**Ne jamais écrire un chemin en dur.** Tout passe par `routes` et les fonctions dérivées
(`heroIncidentRoute(id)`, `blogArticleRoute(slug)`, `heroSheetRoute(id)`…). Les URL sont en
français (`/encyclopedie`, `/connexion`, `/mot-de-passe-oublie`). Ajouter une page = ajouter
sa route dans `routes.ts` (et dans la sous-navigation si elle doit y figurer).

## Authentification et session

### Fichiers

| Fichier                                                   | Rôle                                                                                                                                           |
| --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| [src/lib/cookies.ts](../src/lib/cookies.ts)               | Nom du cookie (`__Host-session` en production, `session` sinon, comme l'API) et attributs : `httpOnly`, `sameSite: lax`, `secure` en prod      |
| [src/lib/session.ts](../src/lib/session.ts)               | `getUserFromToken(token)` : jeton → `AuthUser \| null`. Utilisable partout (pas de `next/headers`)                                             |
| [src/lib/server-session.ts](../src/lib/server-session.ts) | Helpers qui lisent le cookie : `getCurrentUser()`, `requireRole()`, `requireAdminSession()`, `requireCitizenSession()`, `requireHeroSession()` |
| [src/proxy.ts](../src/proxy.ts)                           | Contrôle d'accès avant le rendu                                                                                                                |
| [src/app/actions/auth.ts](../src/app/actions/auth.ts)     | `loginAction` (pose le cookie) et `logoutAction` (révoque la session puis efface le cookie)                                                    |
| [src/lib/api/auth.ts](../src/lib/api/auth.ts)             | `login`, `getSessionUser`, `logout` (API ou mocks selon `API_URL`), `register`, `requestPasswordReset` (mocks)                                 |

Le mode dépend de `API_URL` ([config.ts](../src/lib/api/config.ts), `isApiEnabled`) :

- **vide** : comptes mock (voir [src/mocks/users.ts](../src/mocks/users.ts)), jeton `mock-token-<id>`,
  session de 7 jours ;
- **renseignée** : comptes réels de l'API Go, jeton opaque posé par l'API, durée fixée par l'API.
  Le détail (cookie recopié, CSRF, adresse IP, erreurs) est dans
  [backend-contract.md](./backend-contract.md#2-ce-qui-est-branché--la-session).

### Connexion

1. `LoginForm` appelle `loginAction` (via `useActionState`).
2. L'action valide l'identifiant (email, ou pseudo pour un super-héros) puis appelle `login()`.
3. En cas de succès, le jeton est écrit dans le cookie `SESSION_COOKIE`, qui expire avec la
   session (`expiresAt`), et l'utilisateur est redirigé vers `accountHomeByRole[role]`.
4. En cas d'échec, `loginErrorMessage` traduit l'erreur (`LoginError` : identifiants invalides,
   compte suspendu ou non activé, trop de tentatives, service indisponible) sans dire lequel
   des deux champs est faux.

### Déconnexion

`logoutAction` appelle `logout(token)` (révocation côté API, sans effet avec les mocks), puis
efface le cookie **avec ses attributs** : un cookie `__Host-` ne s'efface que par un `Set-Cookie`
`Secure`. La révocation ne lève jamais d'erreur : le cookie est effacé même si l'API ne répond pas.

### Protection des accès : deux niveaux, volontairement redondants

1. **`proxy.ts` (optimiste, rapide)** : il lit le cookie et redirige.
   - visiteur sur `/admin`, `/profil` ou `/heros` → `/connexion` ;
   - utilisateur du mauvais rôle → son propre espace ;
   - utilisateur connecté sur une page réservée aux visiteurs (`/mot-de-passe-oublie`) → son espace.
2. **`requireRole()` (autoritaire)** : chaque Server Action et chaque composant qui lit des
   données protégées revérifie le rôle au plus près des données. Le proxy n'est **pas** une
   barrière de sécurité suffisante à lui seul.

> Le `matcher` de `proxy.ts` doit contenir des **valeurs littérales** (analysées au build) : si
> vous ajoutez un espace protégé, ajoutez-le à la fois dans `protectedAreas` et dans `matcher`.

### Règle d'or

Une Server Action qui modifie des données **commence par `requireXxxSession()`**. C'est ce qui
fournit le jeton à transmettre à `lib/api/`, et c'est la vraie vérification d'autorisation.

## Couche de données : `src/lib/api/`

- Chaque domaine a son fichier (`incidents.ts`, `super-heroes.ts`, `videos.ts`…), tous
  réexportés par [src/lib/api/index.ts](../src/lib/api/index.ts). **Importer depuis `@/lib/api`**,
  jamais depuis `@/mocks`.
- Les fonctions sont `async`. Celles qui restent mockées renvoient `mockResponse(donnée)` (une
  copie `structuredClone`, pour ne jamais muter les mocks par accident).
- Les fonctions branchées choisissent selon `isApiEnabled` : `apiFetch()`
  ([http.ts](../src/lib/api/http.ts)) si `API_URL` est renseignée, le mock sinon. L'application
  démarre donc toujours sans backend.
- Chaque fonction mockée documente en commentaire l'endpoint futur
  (« Future implémentation : `GET ${API_URL}/incidents` … »). C'est la spec du branchement.
- Les mutations renvoient un résultat typé plutôt que de lever une exception :
  `{ ok: true, … } | { ok: false, error }` (ex : `LoginResult`). Les actions transforment `error`
  en message affiché.
- Les fonctions qui exigent une identité reçoivent le **jeton en premier argument**
  (`declareIncident(token, …)`).

[http.ts](../src/lib/api/http.ts) est le seul client de l'API :

| Élément             | Rôle                                                                                                             |
| ------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `apiFetch(path, …)` | Appel JSON avec `sessionToken` (envoyé dans le cookie attendu par l'API), `csrfToken`, délai de 10 s, sans cache |
| `ApiError`          | Erreur levée pour une réponse non 2xx : `status`, `code` stable de l'API, `retryAfterSeconds`                    |
| `readSessionCookie` | Extrait le jeton du `Set-Cookie` de l'API (connexion)                                                            |
| `fromApiUser`       | Convertit `UserView` vers `AuthUser` : seul endroit où les rôles de l'API (`citizen`, `hero`…) apparaissent      |

Le contrat d'API est décrit dans [backend/openapi/openapi.yaml](../../backend/openapi/openapi.yaml) ;
ce qui est branché et ce qui reste à faire, dans [backend-contract.md](./backend-contract.md).

> Avec l'API, les domaines encore mockés reçoivent une session réelle, mais leurs données sont
> indexées par les comptes mock : `asMockDataUser` donne au vrai compte l'identifiant du compte
> mock de son rôle (voir [backend-contract.md](./backend-contract.md#domaines-encore-mockés-avec-une-session-réelle)).

## Formulaires et Server Actions

Le motif utilisé partout :

```
composant client ("use client")        Server Action ("use server")
───────────────────────────────        ─────────────────────────────────────────
useActionState(action, idleFormState)  1. const token = await requireCitizenSession()
<form action={formAction}>             2. lire le FormData (readXxxValues)
  <TextField error={…} />              3. valider (validateXxx → fieldErrors)
  <SubmitButton>                       4. si erreurs → { status:"error", fieldErrors, values }
  <FormMessage state={state} />        5. appeler lib/api → si !ok → { status:"error", message }
</form>                                6. revalidatePath(…) puis { status:"success", message }
```

- **`FormState<Field>`** ([form-state.ts](../src/lib/validation/form-state.ts)) est le contrat
  entre action et composant : `status`, `message` global, `fieldErrors` par champ, `values` pour
  **re-remplir le formulaire** après une erreur (ne jamais y renvoyer un mot de passe).
- La validation vit dans `src/lib/validation/` (un fichier par domaine) : constantes de limites,
  `readXxxValues(formData)`, `validateXxx(values)`. Elle est **toujours exécutée côté serveur**,
  même si le navigateur a déjà contrôlé le champ.
- Après une mutation, `revalidatePath()` sur chaque page dont les données changent.
- `loginAction` utilise son propre état (`LoginFormState`) : c'est une exception historique,
  les nouveaux formulaires utilisent `FormState`.
- Les actions sont rangées par domaine dans `src/app/actions/` (`incidents.ts`, `admin.ts`,
  `hero.ts`, `account.ts`…). Plusieurs rôles peuvent partager une même logique via une fonction
  interne (voir `submitDeclaration` dans `incidents.ts`), chaque action publique imposant son rôle.

## Rendu : Server Components et `<Suspense>`

- Les `page.tsx` sont **légères** : métadonnées, en-tête, puis un composant de contenu enveloppé
  dans `<Suspense fallback={<…Skeleton />}>`. C'est le composant de contenu qui lit la session et
  les données (voir `app/heros/dashboard/page.tsx` + `HeroDashboardContent`).
- Lire `cookies()` rend la route dynamique : d'où l'obligation de le faire sous `<Suspense>`
  (voir le commentaire de [server-session.ts](../src/lib/server-session.ts)).
- `"use client"` uniquement pour : état local, événements navigateur, `useActionState`,
  `useFormStatus`, API navigateur. Un composant client ne doit pas importer `lib/api/`.
- Les props `children` des layouts sont typées avec les helpers globaux de Next
  (`LayoutProps<"/heros">`).

## Autres mécanismes à connaître

| Sujet                   | Où                                                                         | À savoir                                                                                                                                                                                                                                                                                                                                                           |
| ----------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Thème clair / sombre    | [lib/theme.ts](../src/lib/theme.ts), `ThemeToggle`                         | Choix en `localStorage`, appliqué par un script dans `<head>` (anti-flash). Sombre par défaut                                                                                                                                                                                                                                                                      |
| Design tokens           | bloc `@theme` de [globals.css](../src/app/globals.css)                     | Voir [ui.md](./ui.md#design-tokens)                                                                                                                                                                                                                                                                                                                                |
| Autocomplétion adresse  | `app/api/adresses/route.ts`, `lib/geocoding.ts`                            | Relais vers la Base Adresse Nationale (Géoplateforme IGN), cache 10 min, IP des visiteurs non transmise                                                                                                                                                                                                                                                            |
| Géocodage d'un incident | `geocodeLocation` dans `lib/geocoding.ts`                                  | À la déclaration et quand la modération change le lieu : adresse + ville → `location.coordinates` (BAN). Échec : incident sans position, déclaration acceptée quand même                                                                                                                                                                                           |
| Carte des incidents     | `app/carte` (publique), `app/heros/carte`, `components/map/`, `lib/map.ts` | Leaflet chargé côté navigateur uniquement. Fond de carte : `MAP_TILE_URL` (OpenStreetMap par défaut, voir la politique d'usage dans `lib/map.ts`). Espace pro : position exacte, adresse, héros en intervention (position = lieu de l'intervention, aucun suivi GPS). Carte publique : incidents validés seulement, position arrondie à ~1 km, ni adresse ni héros |
| Assainissement HTML     | [lib/sanitize.ts](../src/lib/sanitize.ts)                                  | Articles de blog (éditeur Tiptap) : à passer avant tout affichage public                                                                                                                                                                                                                                                                                           |
| Newsletter              | cookie `newsletter_subscribed`                                             | Mémorise l'inscription côté navigateur                                                                                                                                                                                                                                                                                                                             |

## Checklists

**Ajouter une page protégée**

1. Créer `src/app/<espace>/<page>/page.tsx` avec `metadata` et un contenu sous `<Suspense>`.
2. Ajouter la route dans `routes.ts` (et dans la sous-navigation du rôle).
3. Dans le composant de contenu, appeler `requireRole(...)` avant de lire des données.

**Ajouter un formulaire**

1. Types et limites dans `lib/validation/<domaine>.ts` (`Field`, `readValues`, `validate`).
2. Fonction d'accès dans `lib/api/<domaine>.ts` (avec le commentaire de l'endpoint futur).
3. Server Action dans `app/actions/<domaine>.ts` (voir le schéma ci-dessus).
4. Composant client avec les champs de [ui.md](./ui.md), `SubmitButton` et `FormMessage`.

**Brancher une fonction à l'API Go**

1. Garder la signature ; appeler `apiFetch()` si `isApiEnabled`, le mock sinon.
2. Mapper les `ApiError` vers le `{ ok: false, error }` existant.
3. Vérifier que le type de `src/types/` correspond au schéma de `openapi.yaml`.

Détail et modèle à suivre : [backend-contract.md](./backend-contract.md#checklist-de-branchement-dune-fonction).
