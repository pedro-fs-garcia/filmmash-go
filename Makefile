MIGRATION_NAME ?= "add_migration"

docker-build:
	docker build -t filmmash-go .

docker-run:
	docker rm filmmash-go
	docker run --env-file .env -p 8000:8080 --name filmmash-go -v filmmash_logs:/logs filmmash-go

docker-shell:
	docker exec -it filmmash-go sh

docker-stop:
	docker stop filmmash-go

# Full local stack; monitoring services need COMPOSE_PROFILES=monitoring in .env
compose-up:
	docker compose up -d

# Core + Alloy shipping to Grafana Cloud (what the VPS runs); needs GRAFANA_CLOUD_* in .env
compose-prod:
	docker compose -f docker-compose.yaml -f docker-compose.prod.yaml up -d

compose-down:
	docker compose --profile "*" down --remove-orphans

new-migration:
	goose create $(MIGRATION_NAME) sql

migrate:
	goose up

migrate-test-db:
	goose -env=".env.test" up

test:
	go test ./...

sqlc:
	sqlc generate