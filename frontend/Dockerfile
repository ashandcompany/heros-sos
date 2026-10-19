# syntax=docker/dockerfile:1

# Front Next.js : un seul fichier, plusieurs cibles.
#   dev      rechargement à chaud           backend/docker-compose.dev.yml  (target: dev)
#   runtime  production, dernière étape     backend/docker-compose.prod.yml (cible par défaut)
#
# Node 24 : version de développement du projet (voir README) et version LTS en cours. Node 20, utilisé
# dans d'autres projets de l'équipe, n'est plus maintenu depuis avril 2026.

ARG NODE_IMAGE=node:24-alpine

# ── Étape 1 : Dépendances ────────────────────────────────────
FROM ${NODE_IMAGE} AS deps
WORKDIR /app

COPY package.json package-lock.json ./
# Cache npm partagé entre les étapes et les builds : les paquets déjà téléchargés (~/.npm) sont
# réutilisés au lieu d'être téléchargés de nouveau. --ignore-scripts : aucun script d'installation
# d'un paquet tiers ne s'exécute pendant le build (les binaires natifs de Next.js, de Tailwind et
# de sharp sont livrés par des dépendances optionnelles, sans script).
RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts

# ── Étape 0 : Dev (rechargement à chaud) ─────────────────────
# L'image ne contient que les dépendances ; docker-compose.dev.yml monte le code source par-dessus.
FROM deps AS dev
ENV NEXT_TELEMETRY_DISABLED=1
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--hostname", "0.0.0.0", "--port", "3000"]

# ── Étape 2 : Dépendances de production seules ───────────────
FROM ${NODE_IMAGE} AS prod-deps
WORKDIR /app

COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --omit=dev --ignore-scripts

# ── Étape 3 : Build ──────────────────────────────────────────
FROM ${NODE_IMAGE} AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Les variables NEXT_PUBLIC_* sont figées dans le code du navigateur au moment du build : elles se
# passent donc ici (build.args dans docker-compose.prod.yml), pas à l'exécution. Vide tant que le
# front travaille sur des données mockées (voir src/lib/api/).
ARG NEXT_PUBLIC_API_URL=
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

# `next/font/google` télécharge les polices pendant la compilation : le build a besoin d'internet.
RUN npm run build

# ── Étape 4 : Image finale ───────────────────────────────────
# Next.js reste servi par Node (`next start`) : les Server Actions et le rendu serveur ne peuvent pas
# être servis par un nginx de fichiers statiques. Le TLS, le routage /api et les en-têtes de sécurité
# sont déjà assurés par le reverse proxy Caddy, devant ce conteneur.
FROM ${NODE_IMAGE} AS runtime
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

COPY --from=prod-deps --chown=node:node /app/node_modules ./node_modules
COPY --from=build --chown=node:node /app/.next ./.next
COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/package.json /app/next.config.ts ./

# Utilisateur non privilégié fourni par l'image Node (uid 1000).
USER node
EXPOSE 3000
CMD ["node_modules/.bin/next", "start"]
