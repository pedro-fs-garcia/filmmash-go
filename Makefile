MIGRATION_NAME ?= "add_migration"

.DEFAULT_GOAL := help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/app ./cmd/filmmash-go

docker-build: ## Build the filmmash-go image
	docker build -t filmmash-go .

docker-run: ## Run the image on port 8000
	docker rm filmmash-go
	docker run --env-file .env -p 8000:8080 --name filmmash-go -v filmmash_logs:/logs filmmash-go

docker-shell: ## Open a shell in the running container
	docker exec -it filmmash-go sh

docker-stop: ## Stop the running container
	docker stop filmmash-go

# Full local stack; monitoring services need COMPOSE_PROFILES=monitoring in .env
compose-up: ## Start the full local stack
	docker compose up -d

# Core + Alloy shipping to Grafana Cloud (what the VPS runs); needs GRAFANA_CLOUD_* in .env
compose-prod: ## Start the prod stack (core + Alloy)
	docker compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d

compose-down: ## Stop the stack and remove orphans
	docker compose --profile "*" down --remove-orphans

new-migration: ## Create a migration (MIGRATION_NAME=...)
	goose create $(MIGRATION_NAME) sql

migrate: ## Apply migrations to the dev database
	goose up

test-db-up: ## Start the postgres container the tests need
	docker compose up -d --wait postgres

migrate-test-db: ## Apply migrations to the test database
	goose -env=".env.test" up

create-test-db: ## Create the test database if it does not exist
	@set -a; . ./.env.test; set +a; \
	export PGPASSWORD="$$POSTGRES_PASSWORD"; \
	psql -h "$$POSTGRES_HOST" -p "$$POSTGRES_PORT" -U "$$POSTGRES_USER" -d postgres \
		-tAc "SELECT 1 FROM pg_database WHERE datname = '$$POSTGRES_DB_NAME'" | grep -q 1 \
	|| psql -h "$$POSTGRES_HOST" -p "$$POSTGRES_PORT" -U "$$POSTGRES_USER" -d postgres \
		-c "CREATE DATABASE \"$$POSTGRES_DB_NAME\""

test: create-test-db migrate-test-db ## Run the test suite
	go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

sqlc: ## Regenerate sqlc code
	sqlc generate

# Run a step silently: print one status line, and its output only when it fails.
define run_step
@printf '  %-6s ' "$(1)"; \
if out=$$($(2) 2>&1); then \
	printf '\033[32mok\033[0m\n'; \
else \
	printf '\033[31mFAIL\033[0m\n\n'; \
	printf '%s\n' "$$out"; \
	exit 1; \
fi
endef

pre-commit: ## Check sqlc is current, then lint, test and build
	$(call run_step,sqlc,sqlc diff)
	$(call run_step,lint,$(MAKE) --no-print-directory lint)
	$(call run_step,test,$(MAKE) --no-print-directory test)
	$(call run_step,build,$(MAKE) --no-print-directory build)
