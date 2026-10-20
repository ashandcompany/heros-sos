# Guide des composants d'interface

Composants réutilisables de `src/components/ui/`. Avant d'écrire un nouveau champ ou bouton,
vérifier qu'il n'existe pas déjà ici. Pour le contexte (formulaires, Server Actions), voir
[architecture.md](./architecture.md#formulaires-et-server-actions).

## Quel composant pour quel besoin ?

| Besoin                              | Composant                                                          | Type    |
| ----------------------------------- | ------------------------------------------------------------------ | ------- |
| Saisie d'une ligne de texte         | `TextField`                                                        | serveur |
| Mot de passe (afficher / masquer)   | `PasswordField`                                                    | client  |
| Adresse avec suggestions            | `AddressField`                                                     | client  |
| Texte long                          | `TextAreaField`                                                    | serveur |
| Choix dans une liste                | `SelectField`                                                      | serveur |
| Choix unique entre quelques options | `RadioGroup`                                                       | serveur |
| Bouton d'envoi d'un formulaire      | `SubmitButton`                                                     | client  |
| Lien ayant l'apparence d'un bouton  | `ButtonLink`                                                       | serveur |
| Classes d'un `<button>` stylé       | `buttonClasses()`                                                  | —       |
| Message global (succès / erreur)    | `FormMessage`                                                      | serveur |
| Ouvrir une fenêtre modale           | `DialogTrigger`                                                    | client  |
| Étiquette de statut ou de priorité  | `Badge`                                                            | serveur |
| Note sur 5 (lecture seule)          | `RatingStars`                                                      | serveur |
| Photo de profil                     | `Avatar`                                                           | serveur |
| Titre de section                    | `SectionHeading`                                                   | serveur |
| Largeur maximale de page            | `Container`                                                        | serveur |
| Défilement horizontal de cartes     | `Carousel`                                                         | client  |
| Combiner des classes CSS            | `cn()`                                                             | —       |
| Décoratifs                          | `Logo`, `WallPoster`, `Typewriter`, `DigitalClock` / `GeneratedOn` | mixte   |

« Serveur » = utilisable dans un Server Component sans `"use client"`. Un composant « client »
peut être importé depuis un Server Component, mais pas l'inverse pour les données.

## Principes communs

- **Accessibilité d'abord.** Chaque champ a un `<label>` relié par `htmlFor`, et ses messages
  d'aide / d'erreur sont reliés par `aria-describedby`. `aria-invalid` est posé quand `error` est
  fourni. Ne jamais remplacer un label par un placeholder.
- **Ids automatiques.** Les champs génèrent leur `id` avec `useId()` ; passer `id` seulement si
  un autre élément doit le référencer.
- **Props natives conservées.** Les champs étendent les props de l'élément HTML (`name`,
  `required`, `defaultValue`, `autoComplete`, `maxLength`…) et les transmettent telles quelles.
- **Taille tactile.** Cibles d'au moins 44 px (`min-h-11` / `min-h-12`).
- **Mouvement réduit.** Les animations respectent `prefers-reduced-motion`.
- **Pas de logique métier.** Ces composants ne connaissent ni l'API ni les rôles.

## Champs de formulaire

Tous les champs affichent l'erreur sous le champ via `error` et passent la bordure en rouge.

### TextField

```tsx
<TextField
  label="Adresse email"
  name="email"
  type="email"
  autoComplete="email"
  defaultValue={state.values?.email}
  error={state.fieldErrors.email}
  hint="Nous ne la partagerons jamais."
/>
```

| Prop         | Rôle                                                             |
| ------------ | ---------------------------------------------------------------- |
| `label`      | Obligatoire                                                      |
| `hint`       | Aide sous le champ                                               |
| `error`      | Message d'erreur (celui de `state.fieldErrors.<champ>`)          |
| `labelAside` | Contenu à droite du label (ex : lien « Mot de passe oublié ? »)  |
| `trailing`   | Contenu dans le champ, à droite (ex : bouton afficher / masquer) |

### PasswordField

`TextField` sans `type` ni `trailing`, avec un bouton œil (`aria-pressed`). Mêmes props sinon.
Ne jamais pré-remplir sa valeur après une erreur.

### AddressField

Champ d'adresse en combobox ARIA 1.2 (flèches, Entrée, Échap). Interroge `/api/adresses` après
300 ms et 3 caractères minimum. La saisie libre reste possible : les suggestions sont une aide.

```tsx
<AddressField
  label="Adresse"
  name="address"
  kind="postal"
  fill={{ postalCode: "postalCode", city: "city", country: "country" }}
  error={state.fieldErrors.address}
/>
```

| Prop   | Rôle                                                                                                                                                                                                        |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `kind` | `"postal"` (numéro + rue, défaut) ou `"place"` (commerce, gare…)                                                                                                                                            |
| `fill` | Associe `postalCode` / `city` / `country` à l'attribut `name` des autres champs du même formulaire, remplis au choix d'une suggestion. Un `<select>` n'est modifié que si la valeur existe dans ses options |

### TextAreaField, SelectField

Même contrat que `TextField` (`label`, `error`, et `hint` pour `TextAreaField`).
`TextAreaField` : `rows` vaut 4 par défaut. `SelectField` reçoit ses `<option>` en `children`.

### RadioGroup

Pour **2 à 4 options** courtes. Le groupe est un `<fieldset>` avec `legend`.

```tsx
<RadioGroup
  legend="Vous êtes"
  name="reporterRole"
  options={reporterRoleOptions}
  defaultValue={state.values?.reporterRole}
  error={state.fieldErrors.reporterRole}
  required
/>
```

> Pour une saisie spécifique (ex : note en étoiles de `RatingForm`), le motif est le même :
> vrais `<input type="radio">` masqués visuellement (`sr-only`), pour garder clavier et lecteurs
> d'écran. Ne pas construire de widget à base de `<div onClick>`.

## Boutons et liens

### ButtonLink

Un `<Link>` Next avec l'apparence d'un bouton. À utiliser pour **naviguer**.

| Prop      | Valeurs                                                       | Usage                                                                       |
| --------- | ------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `variant` | `primary` (défaut), `secondary`, `danger`, `ghost`, `console` | `danger` pour refuser / supprimer ; `console` pour le gros appel à l'action |
| `size`    | `md` (défaut), `lg`                                           |                                                                             |

`buttonClasses(variant, size)` renvoie les mêmes classes pour styler un `<button>` natif.

### SubmitButton

À placer **à l'intérieur** d'un `<form>` : il lit l'état du formulaire parent
(`useFormStatus`) et se désactive pendant l'envoi.

```tsx
<SubmitButton pendingLabel="Envoi en cours…">Envoyer</SubmitButton>
```

`children` est une chaîne ; `variant` : `primary`, `secondary` ou `danger`.

### DialogTrigger

Ouvre un `<dialog>` natif rendu côté serveur (focus piégé, Échap et retour du focus gérés par
le navigateur) :

```tsx
<DialogTrigger dialogId="details-42" className={buttonClasses("secondary")}>
  Voir le détail
</DialogTrigger>
<dialog id="details-42">…</dialog>
```

## Retours utilisateur

### FormMessage

Affiche `state.message` d'une Server Action : `role="status"` en succès, `role="alert"` en
erreur. Ne rend rien s'il n'y a pas de message. Les erreurs **par champ** vont dans le `error`
du champ concerné, pas ici.

### Badge

`tone` : `neutral`, `paper`, `accent`, `success`, `warning`, `danger`, `sage`.
Convention pour les incidents : `paper` = signalé, `warning` / `accent` = en cours,
`success` = réussi, `danger` = échoué ou priorité haute, `sage` = priorité basse. Les
correspondances statut → teinte sont centralisées (voir `IncidentBadges`, `ModerationBadge`) :
les réutiliser plutôt que de choisir une teinte à la main.

### RatingStars

Note sur 5 en lecture seule, avec un libellé complet pour les lecteurs d'écran
(« Note moyenne : 4,2 sur 5 »). Les étoiles sont décoratives. Pour **saisir** une note, voir
`RatingForm`.

## Contenu et mise en page

- **Container** : `max-w-6xl` centré, gouttières responsives. Enveloppe le contenu de chaque page.
- **SectionHeading** : `<h2>` avec `id` obligatoire, à référencer par `aria-labelledby` sur la
  `<section>` parente. Props : `eyebrow`, `section` (affiche « SECTION 02 »), `description`.
- **Avatar** : `name`, `src` (`null` → initiales sur dégradé), `size`, `square` (photo d'identité
  noir et blanc des fiches d'agent). Décoratif (`alt=""`) : **toujours afficher le nom à côté**.
  Affiché aussi dans la navigation (photo de l'utilisateur connecté, via `getCurrentUserAvatar`, même nom et même photo que la page de profil).
- **AvatarForm** (`components/citizen/AccountForms.tsx`) : recadrage rond avec `react-easy-crop`,
  puis l'image est réduite à 256 px et compressée en WebP dans le navigateur (`lib/avatar-image.ts`).
  Seule l'image recadrée est envoyée à la Server Action, qui garde sa validation (type, 2 Mo).
- **Carousel** : chaque enfant devient une diapositive ; `label` donne son nom accessible. Défilement
  natif (scroll-snap), boutons, pastilles et clavier (flèches, Début, Fin).

## Décoratifs

Masqués aux lecteurs d'écran, ne jamais y mettre d'information essentielle :

- `WallPoster` : affiche punaisée, cachée en dessous de `lg`.
- `Typewriter` : texte révélé lettre à lettre ; le texte complet reste dans un `sr-only`.
- `DigitalClock` / `GeneratedOn` : horloge et date, rendues côté client (affichent un
  placeholder au premier rendu pour éviter un décalage d'hydratation).
- `Logo` : monogramme SVG avec `aria-label="Heros-SOS"`.

## Design tokens

Définis dans le bloc `@theme` de [globals.css](../src/app/globals.css) (Tailwind v4) : chaque
`--color-xxx` génère `bg-xxx`, `text-xxx`, `border-xxx`… Direction artistique : rétro-futurisme
et brutalisme (angles droits, filets épais, ombres dures décalées). **Les arrondis sont
supprimés** (`rounded-full` reste utilisable).

| Famille          | Tokens                                                                        |
| ---------------- | ----------------------------------------------------------------------------- |
| Fonds            | `background`, `surface`, `surface-raised`, `border`                           |
| Texte            | `foreground`, `muted`                                                         |
| Marque           | `primary` (+ `-hover`, `-foreground`, `-ink`), `accent`, `mustard`            |
| Papier / encre   | `paper`, `ink` (identiques dans les deux thèmes)                              |
| Contours, ombres | `edge`, `shadow`, `shadow-deep`                                               |
| Statuts          | `success`, `danger`, `rating`, `brick`, `sage`, `terminal`                    |
| Polices          | `font-sans` (IBM Plex Sans), `font-display` (Jost), `font-mono`, `font-stamp` |

- **Thème sombre par défaut ; thème clair** via `<html data-theme="light">`
  (`:root[data-theme="light"]` redéfinit les tokens). Utiliser les tokens et non des couleurs
  hexadécimales, sinon le composant ne suivra pas le thème.
- Texte orange sur fond crème : utiliser `primary-ink`. Texte d'erreur sur fond sombre : `danger`.
- Pour une ombre dure : `shadow-[3px_3px_0_0_var(--color-shadow)]`.

## Écrire un nouveau composant UI

1. Un fichier par composant dans `components/ui/`, sans `"use client"` sauf nécessité.
2. Étendre `ComponentPropsWithoutRef<"élément">` et transmettre `...props` pour ne pas bloquer
   `name`, `required`, `aria-*`…
3. Accepter `className` et le fusionner avec `cn()`.
4. Relier label, aide et erreur par `useId()` et `aria-describedby` ; tester au clavier.
5. Utiliser les tokens du thème, vérifier les deux thèmes.
6. Un commentaire `/** … */` d'une ou deux lignes au-dessus du composant (comportement
   accessibilité inclus), comme les composants existants.
7. L'ajouter au tableau « Quel composant pour quel besoin ? » ci-dessus.
