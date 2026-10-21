# syntax=docker/dockerfile:1

# Site de documentation Docusaurus : image de DÉVELOPPEMENT (rechargement à chaud), lancée par
# `make up` ou `make docs` (docker-compose.dev.yml, target: dev). La publication du site, elle, se fait
# sur un hébergement de fichiers statiques (voir README) : pas d'image de production.
#
# Les dépendances sont installées ici, une fois, et mises en cache. Le code de docs-site/ est monté par
# dessus par Compose. Elles sont dans /app/node_modules, UN NIVEAU AU-DESSUS du site (/app/docs-site) :
# Node les y trouve en remontant l'arborescence, et le volume qui masque le node_modules de l'hôte
# (binaires d'un autre système) démarre vide. Installées dans /app/docs-site/node_modules, elles seraient
# recopiées (1,4 Go) dans ce volume à chaque création du conteneur : de quoi remplir le disque de Docker.
# Le site lit aussi backend/STACK.md, MVP.md, TABLES.md et openapi/openapi.yaml : le service `docs-sync`
# les recopie dans un volume monté en lecture seule (/app/backend). Jamais copiés dans l'image, et jamais
# avec le reste de backend/ (le .env n'a rien à faire dans ce conteneur).
#
# Même version de Node que le front (frontend/Dockerfile).

ARG NODE_IMAGE=node:24-alpine

# ── Dépendances ──────────────────────────────────────────────
FROM ${NODE_IMAGE} AS deps
RUN mkdir -p /app && chown node:node /app
WORKDIR /app
# Utilisateur non privilégié dès l'installation : node_modules lui appartient (pas de chown -R coûteux).
USER node

COPY --chown=node:node package.json package-lock.json ./
# Cache npm partagé entre les builds. --ignore-scripts : aucun script d'installation d'un paquet tiers ne
# s'exécute pendant le build de l'image.
RUN --mount=type=cache,target=/home/node/.npm,uid=1000,gid=1000 \
    npm ci --ignore-scripts

# ── Développement ────────────────────────────────────────────
FROM deps AS dev
# Dossiers générés (cache Docusaurus, pages d'API issues de l'OpenAPI) et node_modules (vide, il masque
# celui de l'hôte) : créés ici pour appartenir à `node`. docker-compose.dev.yml les remplace par des
# volumes qui reprennent ces droits, ce qui évite de polluer docs-site/ sur la machine.
RUN mkdir -p /app/docs-site/node_modules /app/docs-site/.docusaurus /app/docs-site/docs/api /app/docs-site/build
WORKDIR /app/docs-site

EXPOSE 3001
# `npm start` régénère les pages d'API (prestart), puis lance le serveur sur le port 3001 (voir package.json).
# --host 0.0.0.0 : sinon le serveur n'écoute que dans le conteneur et le port publié n'atteint rien.
CMD ["npm", "start", "--", "--host", "0.0.0.0"]
