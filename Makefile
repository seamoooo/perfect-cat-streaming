.PHONY: help up down logs build fmt test pounce nap stalk groom sniff init-env \
        db-shell schema-dryrun schema-apply schema-export

# --- DB connection (overridable; defaults match docker-compose.yml) ---
DB_HOST     ?= 127.0.0.1
DB_PORT     ?= 3306
DB_USER     ?= perfectcat
DB_PASSWORD ?= perfectcat
DB_NAME     ?= perfectcat
SCHEMA_FILE ?= infra/schema/schema.sql

help:
	@echo "perfect-cat-streaming -- Bincho × Kanpachi HLS streamer"
	@echo ""
	@echo "Standard targets:"
	@echo "  make up             Build & start all services (docker compose up --build)"
	@echo "  make down           Stop & remove containers"
	@echo "  make logs           Tail logs"
	@echo "  make build          Build images only"
	@echo "  make fmt            Format Go + frontend"
	@echo "  make test           Run go test + vitest"
	@echo "  make init-env       Copy .env.example -> .env if missing"
	@echo ""
	@echo "Schema (sqldef / mysqldef):"
	@echo "  make schema-dryrun  Show DDL diff without applying"
	@echo "  make schema-apply   Converge live DB to infra/schema/schema.sql"
	@echo "  make schema-export  Dump current live DDL"
	@echo "  make db-shell       Open mysql CLI inside the db container"
	@echo ""
	@echo "Cat-themed aliases:"
	@echo "  make pounce    = up         (cats pounce on prey)"
	@echo "  make nap       = down       (cats nap)"
	@echo "  make stalk     = logs       (cats stalk attentively)"
	@echo "  make groom     = fmt        (cats groom themselves)"
	@echo "  make sniff     = test       (cats sniff for trouble)"

init-env:
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; else echo ".env already exists"; fi

up: init-env
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

build:
	docker compose build

fmt:
	cd backend && go fmt ./...
	cd frontend && npm run fmt --if-present || true

test:
	cd backend && go test ./...
	cd frontend && npm test --if-present -- --run || true

# --- Schema management (sqldef / mysqldef) ---
# Requires `mysqldef` on PATH: brew install sqldef/sqldef/mysqldef
#                              or  go install github.com/sqldef/sqldef/cmd/mysqldef@latest

schema-dryrun:
	@command -v mysqldef >/dev/null 2>&1 || { echo "mysqldef not found. Install: brew install sqldef/sqldef/mysqldef"; exit 1; }
	mysqldef --dry-run -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) < $(SCHEMA_FILE)

schema-apply:
	@command -v mysqldef >/dev/null 2>&1 || { echo "mysqldef not found. Install: brew install sqldef/sqldef/mysqldef"; exit 1; }
	mysqldef -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME) < $(SCHEMA_FILE)

schema-export:
	@command -v mysqldef >/dev/null 2>&1 || { echo "mysqldef not found. Install: brew install sqldef/sqldef/mysqldef"; exit 1; }
	mysqldef --export -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME)

db-shell:
	docker compose exec db mysql -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME)

pounce: up
nap: down
stalk: logs
groom: fmt
sniff: test
