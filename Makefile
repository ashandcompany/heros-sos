.DEFAULT_GOAL := help
SHELL := /bin/bash

# Point d'entrée à la racine : les fichiers Compose (docker-compose*.yml) sont ici, et chaque cible
# délègue à backend/Makefile (qui porte la logique : ports, secrets, modérateur…).
# Les variables passées en ligne de commande (EMAIL=…, V=1…) sont transmises telles quelles.
BACKEND := $(MAKE) --no-print-directory -C backend

# Même projet Compose que backend/Makefile (COMPOSE_DEV), pour les cibles qui n'y ont pas d'équivalent.
# L'environnement (.env) reste dans backend/ : Compose l'utilise pour les ${…} des fichiers Compose.
COMPOSE_DEV := docker compose -p heros-sos-dev --env-file backend/.env -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: help
help: ## Affiche cette aide
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: docker
docker:
	@docker info >/dev/null 2>&1 || { echo "Docker ne répond pas : lancez Docker Desktop (ou le démon) puis recommencez."; exit 1; }

# --- Développement ------------------------------------------------------------------------------------

.PHONY: up
up: docker ## Démarre tout en dev, doc comprise (crée backend/.env au premier lancement) : http://localhost:8088
	@test -f backend/.env || $(BACKEND) env
	@$(BACKEND) up-dev

.PHONY: down
down: docker ## Arrête la pile de dev (les données sont conservées)
	@$(BACKEND) down-dev

.PHONY: logs
logs: docker ## Suit les journaux de la pile de dev
	@$(BACKEND) logs-dev

.PHONY: ps
ps: docker ## Liste les conteneurs de dev, leur état et leurs ports
	@$(COMPOSE_DEV) ps

.PHONY: adminer
adminer: docker ## Ajoute Adminer (client SQL) à la pile de dev : http://localhost:8082
	@$(COMPOSE_DEV) --profile tools up -d adminer

.PHONY: moderator
moderator: docker ## Crée un modérateur en dev : make moderator EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ
	@$(BACKEND) moderator-dev

.PHONY: resetmfa
resetmfa: docker ## Réinitialise la 2FA d'un compte (appareil perdu) en dev : make resetmfa EMAIL=…
	@$(BACKEND) resetmfa-dev

.PHONY: reset
reset: docker ## Arrête la pile de dev ET SUPPRIME ses données (base, photos, RabbitMQ), après confirmation
	@read -rp "Supprimer les volumes de développement (base de données comprise) ? [o/N] " answer; \
	[ "$$answer" = "o" ] || { echo "Annulé : rien n'a été supprimé."; exit 1; }
	@$(BACKEND) down-dev V=1

# --- Production ---------------------------------------------------------------------------------------

.PHONY: secrets
secrets: ## Génère backend/secrets/ pour la production (ne remplace jamais un secret existant)
	@$(BACKEND) secrets

.PHONY: up-prod
up-prod: docker ## Démarre la pile de production (exige backend/.env.production et backend/secrets/)
	@$(BACKEND) up-prod

.PHONY: down-prod
down-prod: docker ## Arrête la pile de production (les données sont conservées)
	@$(BACKEND) down-prod

.PHONY: logs-prod
logs-prod: docker ## Suit les journaux de la pile de production
	@$(BACKEND) logs-prod

# --- Documentation ------------------------------------------------------------------------------------

# La documentation tourne dans Docker (service `docs`, voir docs-site/Dockerfile) : aucun Node sur la machine.
# `make up` la démarre déjà ; cette cible ne lance qu'elle, sans le reste de la pile. Port 3001 (le front en
# `npm run dev` hors Docker prendrait le 3000).
.PHONY: docs
docs: docker ## Démarre seulement la documentation (Docusaurus, rechargée à chaud) : http://localhost:3001/heros-sos/
	@test -f backend/.env || $(BACKEND) env
	@$(BACKEND) disk-check
	@$(COMPOSE_DEV) rm -sfv docs >/dev/null
	@$(COMPOSE_DEV) up -d --build docs
	@set -a; . ./backend/.env; set +a; echo "Documentation : http://localhost:$${DOCS_PORT:-3001}/heros-sos/   (journaux : make logs)"

# --- Vérifications ------------------------------------------------------------------------------------

.PHONY: check
check: docker ## Valide la syntaxe des fichiers Compose (dev et prod)
	@$(BACKEND) compose-check
