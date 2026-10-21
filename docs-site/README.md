# Documentation Heros-SOS

Site de documentation technique, construit avec [Docusaurus](https://docusaurus.io/). Il a son propre
`package.json` et ne dépend ni du front Next.js (`frontend/`) ni du backend Go (`backend/`).

| Section        | Source                                                                                                                         |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Accueil        | `docs-site/docs/intro.mdx`                                                                                                     |
| Architecture   | `docs-site/docs/architecture.md` et `architecture/` (schémas Mermaid)                                                          |
| Fonctionnement | `docs-site/docs/fonctionnement.md`                                                                                             |
| Backend        | `backend/MVP.md`, `backend/STACK.md` et `backend/stack/`, `backend/TABLES.md` et `backend/tables/` (lus sur place, non copiés) |
| API            | `backend/openapi/openapi.yaml` (pages générées, non versionnées)                                                               |

## Commandes

Le site tourne **dans Docker** (service `docs`, image `docs-site/Dockerfile`) : aucun Node n'est nécessaire sur la
machine. `make up` le démarre avec le reste de la pile ; `make docs` ne démarre que lui. Le code de `docs-site/` est
monté dans le conteneur et rechargé à chaud : http://localhost:3001/heros-sos/ (3001 : le 3000 est celui du front).

Le conteneur ne lit de `backend/` que `STACK.md`, `stack/*.md`, `MVP.md`, `TABLES.md`, `tables/*.md` et `openapi/openapi.yaml`, recopiés par un
petit service `docs-sync` : jamais `backend/` en entier, qui contient le `.env`.

Sans Docker, le site se lance aussi avec Node (22 ou plus), par exemple pour le CI :

```bash
npm ci            # installe les dépendances (Node 22 ou plus)
npm start         # serveur de développement : http://localhost:3001/heros-sos/ (sans ouvrir de navigateur)
npm run build     # build de production dans build/ ; un lien cassé le fait échouer
npm run serve     # sert le build local
npm run typecheck # vérifie les types de la configuration
```

Les blocs ` ```mermaid ` des pages (schémas d'architecture, schémas des tables de `backend/tables/`) sont
rendus en diagrammes ; GitHub les rend aussi. Un schéma de plus de 10 à 12 éléments devient illisible : mieux
vaut en faire plusieurs, par domaine.

`npm start` et `npm run build` régénèrent d'abord les pages d'API (`npm run api:gen`). Lancer
`npm run api:clean` les supprime.

## Intégration continue et déploiement

Le workflow `.github/workflows/docs-ci.yml` construit le site à chaque pull request qui touche
`docs-site/`, les documents du backend ou l'OpenAPI. Il ne publie rien.

Le site se publie sur **GitHub Pages** (<https://ashandcompany.github.io/heros-sos/>, réglé par `url` et `baseUrl`
dans `docusaurus.config.ts`) avec `.github/workflows/docs-deploy.yml` : mêmes chemins et mêmes étapes que la CI,
à chaque push sur `develop` (ou à la demande, *Run workflow*). Le workflow ne publie rien tant que le dépôt n'est
pas réglé, une fois pour toutes :

1. *Settings → Pages → Source* : **GitHub Actions**.
2. *Settings → Environments → github-pages → Deployment branches* : autoriser `develop` (par défaut, seule la
   branche principale peut déployer).

Le dépôt est public : le site l'est aussi. Les revues de code sont demandées par `.github/CODEOWNERS`, qui
y inscrit `docs-site/`.
