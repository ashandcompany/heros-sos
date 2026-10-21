# syntax=docker/dockerfile:1

# Front Next.js : un seul fichier, plusieurs cibles.
#   dev      rechargement à chaud           docker-compose.dev.yml  (target: dev)
#   runtime  production, dernière étape     docker-compose.prod.yml (cible par défaut)
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

# ── Étape 2 : Build ──────────────────────────────────────────
FROM ${NODE_IMAGE} AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Aucune variable à figer au build : l'URL de l'API (API_URL) est lue par le serveur au démarrage
# (environment dans docker-compose.*.yml, à la racine), la même image sert donc partout.
# `next/font/google` télécharge les polices pendant la compilation : le build a besoin d'internet.
RUN npm run build

# ── Étape 3 : Image finale ───────────────────────────────────
# Sortie `standalone` (voir next.config.ts) : Next.js ne garde que les modules réellement utilisés et
# fournit un server.js autonome, d'où une image bien plus légère qu'avec node_modules + `next start`.
# Le rendu serveur et les Server Actions restent servis par Node ; le TLS, le routage /api et les
# en-têtes de sécurité sont assurés par le reverse proxy Caddy, devant ce conteneur.
FROM ${NODE_IMAGE} AS runtime
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static

# Utilisateur non privilégié fourni par l'image Node (uid 1000).
USER node
EXPOSE 3000
CMD ["node", "server.js"]
