.DEFAULT_GOAL := help
SHELL := /bin/bash

# Base PostgreSQL jetable pour le développement et les tests (remplacée par docker-compose).
DB_CONTAINER   ?= heros-sos-db
DB_PORT        ?= 5432
DB_USER        ?= heros
DB_PASSWORD    ?= heros
DB_NAME        ?= herossos
DB_TEST_NAME   ?= herossos_test
DB_IMAGE       ?= postgres:18-alpine
export HEROS_TEST_DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_TEST_NAME)?sslmode=disable

# Charge .env (s'il existe) avant de lancer une commande Go.
WITH_ENV = set -a; [ -f .env ] && . ./.env; set +a;

.PHONY: help
help: ## Affiche cette aide
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: env
env: ## Crée .env avec des secrets aléatoires (ne remplace jamais un .env existant)
	@test ! -f .env || { echo ".env existe déjà : rien n'a été modifié"; exit 1; }
	@sed -e "s|CHANGE_ME_BASE64_32_BYTES|$$(openssl rand -base64 32 | tr -d '\n')|" \
	     -e "s|CHANGE_ME_BASE64_64_BYTES|$$(openssl rand -base64 64 | tr -d '\n')|" \
	     .env.example > .env
	@echo ".env créé. Sauvegardez HEROS_CRYPTO_KEYS : sans elle, les données chiffrées sont perdues."

.PHONY: db-start
db-start: ## Démarre PostgreSQL dans Docker (bases herossos et herossos_test)
	@docker inspect $(DB_CONTAINER) >/dev/null 2>&1 && docker start $(DB_CONTAINER) >/dev/null || \
	docker run -d --name $(DB_CONTAINER) -p $(DB_PORT):5432 \
		-e POSTGRES_USER=$(DB_USER) -e POSTGRES_PASSWORD=$(DB_PASSWORD) -e POSTGRES_DB=$(DB_NAME) \
		$(DB_IMAGE) >/dev/null
	@# -h 127.0.0.1 (TCP) : pendant son initialisation, l'image démarre un serveur temporaire qui n'écoute
	@# que sur son socket. Tester le socket déclarerait la base prête juste avant son redémarrage.
	@until docker exec $(DB_CONTAINER) pg_isready -h 127.0.0.1 -U $(DB_USER) -d $(DB_NAME) >/dev/null 2>&1; do sleep 0.5; done
	@docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -tc \
		"SELECT 1 FROM pg_database WHERE datname='$(DB_TEST_NAME)'" | grep -q 1 || \
		docker exec $(DB_CONTAINER) createdb -U $(DB_USER) $(DB_TEST_NAME)
	@echo "PostgreSQL prêt sur le port $(DB_PORT)."

.PHONY: db-stop
db-stop: ## Arrête et supprime le conteneur PostgreSQL (les données sont perdues)
	@# -v : supprime aussi le volume anonyme de l'image PostgreSQL, sinon chaque cycle en abandonne un.
	@docker rm -fv $(DB_CONTAINER) >/dev/null 2>&1 || true

.PHONY: migrate
migrate: ## Applique les migrations SQL
	@$(WITH_ENV) go run ./cmd/api -migrate

.PHONY: run
run: ## Lance l'API
	@$(WITH_ENV) go run ./cmd/api

.PHONY: createmoderator
createmoderator: ## Crée un modérateur : make createmoderator EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ (mot de passe demandé)
	@test -n "$(EMAIL)" -a -n "$(FIRST)" -a -n "$(LAST)" -a -n "$(BIRTH)" || { echo "EMAIL, FIRST, LAST et BIRTH sont obligatoires"; exit 1; }
	@read -rsp "Mot de passe : " password; echo; \
	$(WITH_ENV) HEROS_MODERATOR_PASSWORD="$$password" go run ./cmd/createmoderator \
		-email "$(EMAIL)" -first-name "$(FIRST)" -last-name "$(LAST)" -birth-date "$(BIRTH)"

.PHONY: build
build: ## Compile tous les paquets
	go build ./...

.PHONY: test
test: ## Lance les tests (nécessite PostgreSQL : make db-start)
	go test ./... -race -count=1

.PHONY: cover
cover: ## Lance les tests avec le taux de couverture par paquet
	go test ./... -race -count=1 -cover

.PHONY: fmt
fmt: ## Formate le code
	gofmt -w .

.PHONY: lint
lint: ## Vérifie le formatage et lance go vet
	@test -z "$$(gofmt -l .)" || { echo "Fichiers mal formatés :"; gofmt -l .; exit 1; }
	go vet ./...

.PHONY: vuln
vuln: ## Cherche les vulnérabilités connues atteignables dans le code
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tidy
tidy: ## Nettoie go.mod et go.sum
	go mod tidy

# --- Docker : piles de développement et de production -----------------------------------------------

# Deux projets Compose distincts : conteneurs, réseaux et volumes séparés. Un `up-prod` ne peut jamais
# toucher aux données de développement, ni l'inverse, même lancés sur la même machine.
COMPOSE_DEV  = docker compose -p heros-sos-dev -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_PROD = docker compose -p heros-sos-prod --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml

.PHONY: up-dev
up-dev: ## Démarre la pile de développement (proxy, API, base, Mailpit, RabbitMQ, front)
	@test -f .env || { echo "Créez .env d'abord : make env"; exit 1; }
	@set -a; . ./.env; set +a; \
	for port in $${PROXY_PORT:-8088} $${API_PORT:-8081} $${MAILPIT_PORT:-8025} $${POSTGRES_PORT:-5432} $${RABBITMQ_PORT:-5672} $${RABBITMQ_UI_PORT:-15672}; do \
		if ! $(COMPOSE_DEV) ps --status running -q 2>/dev/null | grep -q . && command -v lsof >/dev/null && lsof -nP -iTCP:$$port -sTCP:LISTEN >/dev/null 2>&1; then \
			echo "Le port $$port est déjà utilisé sur cette machine. Choisissez-en un autre dans .env (PROXY_PORT, API_PORT, MAILPIT_PORT, POSTGRES_PORT, RABBITMQ_PORT, RABBITMQ_UI_PORT)."; exit 1; \
		fi; \
	done
	@# Le front garde ses node_modules et son cache .next dans des volumes anonymes (voir
	@# docker-compose.dev.yml). `rm -v` supprime l'ancien conteneur ET ces volumes : sans cela chaque
	@# démarrage en abandonnerait une copie (~700 Mo) et les dépendances resteraient celles d'avant.
	$(COMPOSE_DEV) rm -sfv web
	$(COMPOSE_DEV) up -d --build
	@set -a; . ./.env; set +a; echo "Application : http://localhost:$${PROXY_PORT:-8088}   Mails : http://localhost:$${MAILPIT_PORT:-8025}"

.PHONY: down-dev
down-dev: ## Arrête la pile de développement (V=1 supprime aussi les données)
	$(COMPOSE_DEV) rm -sfv web
	$(COMPOSE_DEV) down $(if $(V),-v)

.PHONY: logs-dev
logs-dev: ## Suit les journaux de la pile de développement
	$(COMPOSE_DEV) logs -f --tail=100

.PHONY: moderator-dev
moderator-dev: ## Crée un modérateur dans la pile de développement : EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ
	$(call create_moderator,$(COMPOSE_DEV))

.PHONY: secrets
secrets: ## Génère secrets/ pour la production (ne remplace JAMAIS un secret existant)
	@set -e; [ -f .env.production ] && { set -a; . ./.env.production; set +a; } || true; \
	umask 022; mkdir -p secrets; chmod 700 secrets; \
	create() { if [ -e "secrets/$$1" ]; then echo "  conservé  secrets/$$1"; else printf '%s' "$$2" > "secrets/$$1"; echo "  créé      secrets/$$1"; fi; }; \
	[ -e secrets/postgres_password ] || printf '%s' "$$(openssl rand -hex 24)" > secrets/postgres_password; \
	create database_url "postgres://$${POSTGRES_USER:-heros}:$$(cat secrets/postgres_password)@db:5432/$${POSTGRES_DB:-herossos}?sslmode=prefer"; \
	create crypto_keys "1:$$(openssl rand -base64 64 | tr -d '\n')"; \
	create csrf_secret "$$(openssl rand -base64 32 | tr -d '\n')"; \
	create smtp_password ""; \
	create rabbitmq_password "$$(openssl rand -hex 24)"; \
	echo; echo "Sauvegardez secrets/crypto_keys HORS du serveur : sans elle, les données chiffrées sont perdues."; \
	echo "Renseignez secrets/smtp_password si votre serveur SMTP demande un mot de passe."

.PHONY: up-prod
up-prod: ## Démarre la pile de production (exige .env.production et secrets/)
	@test -f .env.production || { echo "Copiez .env.production.example vers .env.production et adaptez-le"; exit 1; }
	@test -d secrets || { echo "Générez les secrets d'abord : make secrets"; exit 1; }
	$(COMPOSE_PROD) up -d --build

.PHONY: down-prod
down-prod: ## Arrête la pile de production (les données sont conservées)
	$(COMPOSE_PROD) down

.PHONY: logs-prod
logs-prod: ## Suit les journaux de la pile de production
	$(COMPOSE_PROD) logs -f --tail=100

.PHONY: moderator-prod
moderator-prod: ## Crée un modérateur en production : EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ
	$(call create_moderator,$(COMPOSE_PROD))

.PHONY: compose-check
compose-check: ## Valide la syntaxe des fichiers Compose (développement et production), sur un clone neuf aussi
	@# Les fichiers manquants (.env, .env.production, secrets/) sont créés le temps du contrôle, puis supprimés.
	@set -e; created=""; \
	[ -f .env ] || { cp .env.example .env; created="$$created .env"; }; \
	[ -f .env.production ] || { cp .env.production.example .env.production; created="$$created .env.production"; }; \
	[ -d secrets ] || { mkdir secrets; for f in database_url postgres_password crypto_keys csrf_secret smtp_password rabbitmq_password; do : > secrets/$$f; done; created="$$created secrets"; }; \
	trap 'rm -rf $$created' EXIT; \
	$(COMPOSE_DEV) config -q && echo "développement : valide"; \
	$(COMPOSE_PROD) config -q && echo "production : valide"

# Lance la commande createmoderator (incluse dans l'image) ; le mot de passe est demandé, jamais passé en argument.
define create_moderator
	@test -n "$(EMAIL)" -a -n "$(FIRST)" -a -n "$(LAST)" -a -n "$(BIRTH)" || { echo "EMAIL, FIRST, LAST et BIRTH sont obligatoires"; exit 1; }
	@read -rsp "Mot de passe : " password; echo; \
	$(1) run --rm -e HEROS_MODERATOR_PASSWORD="$$password" --entrypoint /createmoderator api \
		-email "$(EMAIL)" -first-name "$(FIRST)" -last-name "$(LAST)" -birth-date "$(BIRTH)"
endef

.PHONY: check
check: lint test vuln ## Tout ce que vérifie le CI
