---
title: Fonctionnement du site
sidebar_label: Fonctionnement du site
---

## Une seule source

Rien n'est recopié dans ce site, pour qu'il ne dérive jamais de ce qui est réellement livré :

- **MVP, Stack et Tables** sont lus sur place dans `backend/MVP.md`, `backend/STACK.md` et `backend/stack/`,
  `backend/TABLES.md` et `backend/tables/`. Modifier un de ces fichiers met le site à jour, et ils restent lisibles sur GitHub.
- **Frontend** est lu sur place dans `frontend/README.md` et `frontend/docs/*.md`. Ces fichiers restent
  écrits pour GitHub : les alertes `> [!NOTE]` y deviennent des encadrés, et les liens vers le code
  (`../src/…`) mènent au fichier sur GitHub.
- **L'API** est générée à chaque build depuis `backend/openapi/openapi.yaml`, le fichier que les tests du
  backend vérifient route par route (« spec d'abord »). Les pages générées ne sont pas versionnées.
- **La page de présentation, l'architecture, cette page** et la configuration du site sont dans `docs-site/`.

## Travailler sur la documentation

Ce site tourne dans Docker : `make up`, depuis la racine du dépôt, le démarre avec le reste de la pile (ou
`make docs` pour lui seul), sur http://localhost:3001/heros-sos/. Les modifications de `docs-site/` et des
documents de `backend/` sont rechargées à chaud. Sans Docker, depuis `docs-site/` :

```bash
npm ci          # une seule fois
npm start       # http://localhost:3001/heros-sos/ ; régénère les pages d'API puis recharge à chaud
npm run build   # ce que vérifie le CI : un lien cassé fait échouer le build
```

Pour ajouter une page du site, créer un fichier dans `docs-site/docs/` puis l'ajouter à
`docs-site/sidebars.ts`. Pour un document de conception du backend, l'ajouter à `include` dans
`docusaurus.config.ts` et à `sidebarsBackend.ts`. Pour un document du frontend, le créer dans
`frontend/docs/` (il est pris automatiquement) puis l'ajouter à `sidebarsFrontend.ts`.
