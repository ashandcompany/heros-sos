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

.PHONY: seed
seed: ## Remplit la base de démonstration (univers Marvel) ; RESET=1 efface d'abord TOUTES les données
	@$(WITH_ENV) go run ./cmd/seed $(if $(RESET),-reset)

# --- Mesures de performance : 5 millions d'incidents ---------------------------------------------------
# Tout se passe dans une base dédiée (PERF_DB_NAME) : jamais celle du développement quotidien, qui recevrait
# plusieurs Go de données fictives. Ordre : perf-db (si la base est dans Docker), perf-prepare, seed-perf,
# puis perf-run dans un terminal et perf dans un autre.
PERF_DB_NAME      ?= herossos_perf
PERF_DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(PERF_DB_NAME)?sslmode=disable
PERF_INCIDENTS    ?= 5000000
PERF_MAX_SIZE_GB  ?= 0
PERF_API_ADDR     ?= :8090
PERF_API_URL      ?= http://127.0.0.1$(PERF_API_ADDR)/api/v1
PERF_WORKERS      ?= 16
PERF_DURATION     ?= 20s
PERF_P95          ?= 350ms
PERF_USER         ?= jane.foster@heros-sos.local

# Les index de lecture sont seuls dans leur migration : les retirer puis les recréer (sans toucher aux données)
# rejoue en direct la lenteur d'origine, puis le gain. psql lit l'URL de la base de mesures.
PERF_PSQL         ?= psql '$(PERF_DATABASE_URL)' -v ON_ERROR_STOP=1 -q
PERF_INDEXES_SQL  ?= migrations/20261022132000_add_incident_perf_indexes

.PHONY: perf-indexes-off
perf-indexes-off: ## Démonstration : retire les index de lecture de la base de mesures (lenteur d'origine)
	@$(PERF_PSQL) -f $(PERF_INDEXES_SQL).down.sql && echo "Index de lecture retirés : lancez make perf pour voir la lenteur d'origine."

.PHONY: perf-indexes-on
perf-indexes-on: ## Démonstration : recrée les index de lecture de la base de mesures
	@$(PERF_PSQL) -f $(PERF_INDEXES_SQL).up.sql && echo "Index de lecture recréés."

.PHONY: perf-db
perf-db: ## Crée la base de mesures dans la base de la pile de développement (Docker)
	@$(COMPOSE_DEV) exec -T db psql -U $(DB_USER) -d postgres -tc "SELECT 1 FROM pg_database WHERE datname='$(PERF_DB_NAME)'" | grep -q 1 || \
		$(COMPOSE_DEV) exec -T db createdb -U $(DB_USER) $(PERF_DB_NAME)
	@echo "Base $(PERF_DB_NAME) prête."

.PHONY: perf-prepare
perf-prepare: ## Migre la base de mesures et y charge les données Marvel (héros, vilains, citoyens)
	@$(WITH_ENV) export HEROS_DATABASE_URL='$(PERF_DATABASE_URL)'; go run ./cmd/api -migrate && go run ./cmd/seed

.PHONY: seed-perf
seed-perf: ## Ajoute PERF_INCIDENTS (5 000 000) incidents à la base de mesures ; PERF_MAX_SIZE_GB=8 borne sa taille
	@$(WITH_ENV) export HEROS_DATABASE_URL='$(PERF_DATABASE_URL)'; go run ./cmd/seedperf -incidents $(PERF_INCIDENTS) -max-size-gb $(PERF_MAX_SIZE_GB)

.PHONY: perf-run
perf-run: ## Lance l'API sur la base de mesures (migrations comprises), à laisser tourner pendant `make perf`
	@$(WITH_ENV) export HEROS_DATABASE_URL='$(PERF_DATABASE_URL)' HEROS_HTTP_ADDR='$(PERF_API_ADDR)'; go run ./cmd/api

.PHONY: perf
perf: ## Mesure les temps de réponse (p95 ≤ PERF_P95) ; LOADTEST_PASSWORD=… ajoute « mes incidents » (compte PERF_USER)
	@$(WITH_ENV) go run ./cmd/loadtest -base-url '$(PERF_API_URL)' -workers $(PERF_WORKERS) -duration $(PERF_DURATION) \
		-p95 $(PERF_P95) $(if $(LOADTEST_PASSWORD),-identifier '$(PERF_USER)') $(if $(SCENARIOS),-scenarios '$(SCENARIOS)')

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

# Options de golangci-lint en plus. Le CI y met --new-from-merge-base=origin/develop : seules les remarques
# qu'une PR introduit la font échouer, celles qui existent déjà n'arrêtent pas les autres développeurs.
GOLANGCI_FLAGS ?=

.PHONY: golangci
golangci: ## Analyse plus poussée (conventions Go, erreurs courantes) : voir .golangci.yml
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run $(GOLANGCI_FLAGS) ./...

.PHONY: vuln
vuln: ## Cherche les vulnérabilités connues atteignables dans le code
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tidy
tidy: ## Nettoie go.mod et go.sum
	go mod tidy

# --- Docker : piles de développement et de production -----------------------------------------------

# Deux projets Compose distincts : conteneurs, réseaux et volumes séparés. Un `up-prod` ne peut jamais
# toucher aux données de développement, ni l'inverse, même lancés sur la même machine.
#
# Les fichiers Compose sont à la racine du dépôt (back et front se lancent ensemble) : Compose prend
# alors la racine comme dossier du projet. Son `.env` automatique serait donc celui de la racine, d'où
# --env-file explicite. Depuis la racine, `make up`, `make up-prod`… appellent ces cibles.
COMPOSE_DEV  = docker compose -p heros-sos-dev --env-file .env -f ../docker-compose.yml -f ../docker-compose.dev.yml
COMPOSE_PROD = docker compose -p heros-sos-prod --env-file .env.production -f ../docker-compose.yml -f ../docker-compose.prod.yml

# Le disque de la machine virtuelle de Docker (Docker Desktop) se remplit vite : images, cache de build,
# volumes. Faute de place, un build échoue et un conteneur peut redémarrer en boucle (ENOSPC) : on vérifie
# donc l'espace libre AVANT de toucher aux conteneurs.
# MSYS_NO_PATHCONV : sous Windows, Git Bash réécrirait le `/` de `df -k /` en `C:/Program Files/Git/`, df
# échouerait et la cible annoncerait toujours 0 Mo libres. Sans effet sous Linux et macOS.
DISK_MIN_MB ?= 1536
.PHONY: disk-check
disk-check:
	@free=$$(MSYS_NO_PATHCONV=1 docker run --rm alpine:3 df -k / 2>/dev/null | awk 'NR==2 {print int($$4/1024)}'); \
	if [ -z "$$free" ] || { [ "$$free" -lt $(DISK_MIN_MB) ] && [ -z "$(FORCE)" ]; }; then \
		echo "Le disque de Docker est presque plein : $${free:-0} Mo libres, $(DISK_MIN_MB) Mo au moins sont nécessaires."; \
		echo "Sinon un build échoue, ou un conteneur ne peut plus écrire et redémarre en boucle."; \
		echo "  - libérer le cache de build (sans danger) : docker builder prune -a -f"; \
		echo "  - agrandir le disque, durablement         : Docker Desktop > Settings > Resources > Virtual disk limit"; \
		echo "  - continuer malgré tout                   : FORCE=1 make up"; \
		exit 1; \
	fi

.PHONY: up-dev
up-dev: disk-check ## Démarre la pile de développement (proxy, API, base, Mailpit, RabbitMQ, front, documentation)
	@test -f .env || { echo "Créez .env d'abord : make env"; exit 1; }
	@set -a; . ./.env; set +a; \
	for port in $${PROXY_PORT:-8088} $${API_PORT:-8081} $${MAILPIT_PORT:-8025} $${POSTGRES_PORT:-5432} $${RABBITMQ_PORT:-5672} $${RABBITMQ_UI_PORT:-15672} $${DOCS_PORT:-3001}; do \
		if ! $(COMPOSE_DEV) ps --status running -q 2>/dev/null | grep -q . && command -v lsof >/dev/null && lsof -nP -iTCP:$$port -sTCP:LISTEN >/dev/null 2>&1; then \
			echo "Le port $$port est déjà utilisé sur cette machine. Choisissez-en un autre dans .env (PROXY_PORT, API_PORT, MAILPIT_PORT, POSTGRES_PORT, RABBITMQ_PORT, RABBITMQ_UI_PORT, DOCS_PORT)."; exit 1; \
		fi; \
	done
	@# D'abord construire les images : si le build échoue (disque plein, réseau…), les conteneurs en cours
	@# ne sont pas touchés. Supprimer web et docs avant aurait laissé l'application cassée.
	$(COMPOSE_DEV) build
	@# Le front et la documentation gardent leurs node_modules (et le cache .next, les pages d'API) dans
	@# des volumes anonymes (voir docker-compose.dev.yml). `rm -v` supprime l'ancien conteneur ET ces
	@# volumes : sans cela chaque démarrage en abandonnerait une copie (~700 Mo) et les dépendances
	@# resteraient celles d'avant.
	$(COMPOSE_DEV) rm -sfv web docs
	$(COMPOSE_DEV) up -d
	@set -a; . ./.env; set +a; echo "Application : http://localhost:$${PROXY_PORT:-8088}   API : http://localhost:$${PROXY_PORT:-8088}/api/v1/docs   Mails : http://localhost:$${MAILPIT_PORT:-8025}   Documentation : http://localhost:$${DOCS_PORT:-3001}/heros-sos/"

.PHONY: down-dev
down-dev: ## Arrête la pile de développement (V=1 supprime aussi les données)
	$(COMPOSE_DEV) rm -sfv web docs
	$(COMPOSE_DEV) down $(if $(V),-v)

.PHONY: logs-dev
logs-dev: ## Suit les journaux de la pile de développement
	$(COMPOSE_DEV) logs -f --tail=100

.PHONY: moderator-dev
moderator-dev: ## Crée un modérateur dans la pile de développement : EMAIL=… FIRST=… LAST=… BIRTH=AAAA-MM-JJ
	$(call create_moderator,$(COMPOSE_DEV))

# Monitoring ELK (profil Compose `elk`) : Elasticsearch, Logstash, Kibana et Filebeat, qui lit les journaux JSON de l'API.
# Compter 3 à 4 Go de mémoire en plus : donner 6 à 8 Go à Docker Desktop. Les images pèsent environ 2 Go.
ELK_DISK_MIN_MB ?= 4096
ELK_SERVICES    = elasticsearch logstash kibana filebeat elk-setup

.PHONY: elk-dev
elk-dev: ## Démarre le monitoring ELK (journaux de l'API en JSON, Kibana sur http://localhost:5601)
	@$(MAKE) --no-print-directory disk-check DISK_MIN_MB=$(ELK_DISK_MIN_MB)
	@test -f .env || { echo "Créez .env d'abord : make env"; exit 1; }
	@# L'API est recréée avec des journaux JSON ; elle reprend le texte lisible au prochain `make up`.
	HEROS_LOG_FORMAT=json $(COMPOSE_DEV) --profile elk up -d api $(ELK_SERVICES)
	@echo "Kibana : http://localhost:$${KIBANA_PORT:-5601} (le dashboard apparaît une fois elk-setup terminé : make logs-dev)"

.PHONY: elk-down-dev
elk-down-dev: ## Arrête le monitoring ELK (V=1 supprime aussi ses données) et remet les journaux de l'API en texte
	$(COMPOSE_DEV) --profile elk stop $(ELK_SERVICES)
	$(COMPOSE_DEV) --profile elk rm -f $(if $(V),-v) $(ELK_SERVICES)
	$(if $(V),docker volume rm -f heros-sos-dev_esdata >/dev/null 2>&1 || true)
	$(COMPOSE_DEV) up -d api

.PHONY: seed-dev
seed-dev: ## Remplit la base de la pile de développement (univers Marvel) ; RESET=1 efface d'abord TOUTES les données
	$(COMPOSE_DEV) run --rm --entrypoint /seed api $(if $(RESET),-reset)

.PHONY: resetmfa-dev
resetmfa-dev: ## Réinitialise la 2FA d'un compte (appareil perdu) dans la pile de développement : EMAIL=…
	$(call reset_mfa,$(COMPOSE_DEV))

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
	create omdb_api_key ""; \
	create rabbitmq_password "$$(openssl rand -hex 24)"; \
	echo; echo "Sauvegardez secrets/crypto_keys HORS du serveur : sans elle, les données chiffrées sont perdues."; \
	echo "Renseignez secrets/smtp_password si votre serveur SMTP demande un mot de passe."; \
	echo "Renseignez secrets/omdb_api_key avec votre clé OMDb avant de lancer l'API."

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

.PHONY: resetmfa-prod
resetmfa-prod: ## Réinitialise la 2FA d'un compte (appareil perdu) en production : EMAIL=… (vérifier l'identité d'abord)
	$(call reset_mfa,$(COMPOSE_PROD))

.PHONY: compose-check
compose-check: ## Valide la syntaxe des fichiers Compose (développement et production), sur un clone neuf aussi
	@# Les fichiers manquants (.env, .env.production, secrets/) sont créés le temps du contrôle, puis supprimés.
	@set -e; created=""; \
	[ -f .env ] || { cp .env.example .env; created="$$created .env"; }; \
	[ -f .env.production ] || { cp .env.production.example .env.production; created="$$created .env.production"; }; \
	[ -d secrets ] || { mkdir secrets; for f in database_url postgres_password crypto_keys csrf_secret smtp_password omdb_api_key rabbitmq_password; do : > secrets/$$f; done; created="$$created secrets"; }; \
	trap 'rm -rf $$created' EXIT; \
	$(COMPOSE_DEV) config -q && echo "développement : valide"; \
	$(COMPOSE_PROD) config -q && echo "production : valide"

# Lance la commande resetmfa (incluse dans l'image) : elle supprime la double authentification d'un compte et ferme
# ses sessions. L'API ne le permet jamais, pour qu'un compte piraté ne puisse pas retirer sa propre protection.
define reset_mfa
	@test -n "$(EMAIL)" || { echo "EMAIL est obligatoire"; exit 1; }
	$(1) run --rm --entrypoint /resetmfa api -email "$(EMAIL)"
endef

# Lance la commande createmoderator (incluse dans l'image) ; le mot de passe est demandé, jamais passé en argument.
define create_moderator
	@test -n "$(EMAIL)" -a -n "$(FIRST)" -a -n "$(LAST)" -a -n "$(BIRTH)" || { echo "EMAIL, FIRST, LAST et BIRTH sont obligatoires"; exit 1; }
	@read -rsp "Mot de passe : " password; echo; \
	$(1) run --rm -e HEROS_MODERATOR_PASSWORD="$$password" --entrypoint /createmoderator api \
		-email "$(EMAIL)" -first-name "$(FIRST)" -last-name "$(LAST)" -birth-date "$(BIRTH)"
endef

.PHONY: check
check: lint test vuln ## Tout ce que vérifie le CI
