.DEFAULT_GOAL := help
SHELL := /bin/bash

# Base PostgreSQL jetable pour le développement et les tests (remplacée par docker-compose).
DB_CONTAINER   ?= heros-sos-db
DB_PORT        ?= 5432
DB_USER        ?= heros
DB_PASSWORD    ?= heros
DB_NAME        ?= herossos
DB_TEST_NAME   ?= herossos_test
DB_IMAGE       ?= postgres:17
export HEROS_TEST_DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_TEST_NAME)?sslmode=disable

# Charge .env (s'il existe) avant de lancer une commande Go.
WITH_ENV = set -a; [ -f .env ] && . ./.env; set +a;

.PHONY: help
help: ## Affiche cette aide
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

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
	@until docker exec $(DB_CONTAINER) pg_isready -U $(DB_USER) >/dev/null 2>&1; do sleep 0.5; done
	@docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -tc \
		"SELECT 1 FROM pg_database WHERE datname='$(DB_TEST_NAME)'" | grep -q 1 || \
		docker exec $(DB_CONTAINER) createdb -U $(DB_USER) $(DB_TEST_NAME)
	@echo "PostgreSQL prêt sur le port $(DB_PORT)."

.PHONY: db-stop
db-stop: ## Arrête et supprime le conteneur PostgreSQL (les données sont perdues)
	@docker rm -f $(DB_CONTAINER) >/dev/null 2>&1 || true

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

.PHONY: check
check: lint test vuln ## Tout ce que vérifie le CI
