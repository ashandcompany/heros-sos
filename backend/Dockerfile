# syntax=docker/dockerfile:1

# Image de l'API Heros-SOS : compilation statique, puis image `scratch` (ni shell, ni paquets, ni
# gestionnaire d'identités) exécutée par un utilisateur non privilégié. La surface d'attaque se
# réduit à trois binaires et aux certificats racine.

ARG GO_IMAGE=golang:1.26-alpine3.23

# ---- Compilation ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS builder
WORKDIR /src

# Les dépendances d'abord : cette couche n'est reconstruite que si go.mod ou go.sum changent.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# CGO désactivé : binaire statique, indispensable pour `scratch`. -trimpath retire les chemins de
# la machine de build, -s -w la table des symboles (binaire plus petit, build reproductible).
# Les migrations SQL et les modèles de mails sont embarqués : le binaire est autonome.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/createmoderator ./cmd/createmoderator && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed && \
    mkdir /out/avatars

# ---- Image finale -----------------------------------------------------------------------------
FROM scratch

# Certificats racine : nécessaires au SMTP chiffré (STARTTLS, TLS) et aux futurs appels HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# `api` est le serveur ; `createmoderator` crée le premier modérateur :
#   docker compose run --rm --entrypoint /createmoderator api -email … -first-name … -last-name … -birth-date …
# `seed` remplit une base de démonstration (refusé en production) : make seed
COPY --from=builder /out/api /out/createmoderator /out/seed /

LABEL org.opencontainers.image.title="heros-sos-api" \
      org.opencontainers.image.description="API REST de la plateforme Heros-SOS" \
      org.opencontainers.image.source="https://github.com/ashandcompany/heros-sos"

# Dossier des photos de profil : le seul endroit où l'API écrit. Il appartient à l'utilisateur non
# privilégié, et un volume monté dessus reprend ces droits à sa création (voir docker-compose.yml).
# Le reste du système de fichiers peut rester en lecture seule (read_only: true).
COPY --from=builder --chown=65532:65532 /out/avatars /var/lib/heros-sos/avatars
ENV HEROS_AVATAR_DIR=/var/lib/heros-sos/avatars

# Utilisateur non privilégié, par identifiants numériques : `scratch` n'a pas de /etc/passwd.
USER 65532:65532

EXPOSE 8080

# L'image n'a pas de curl : le binaire se sonde lui-même (/api/v1/ready, base de données incluse).
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 CMD ["/api", "-healthcheck"]

# ENTRYPOINT (et non CMD) pour passer des options sans répéter le binaire : `-migrate`.
ENTRYPOINT ["/api"]
