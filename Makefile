.PHONY: help up down logs build fmt test pounce nap stalk groom sniff init-env \
        db-shell schema-dryrun schema-apply schema-export \
        telemetry-fixture telemetry-loop \
        telemetry-browser-deps telemetry-browser-once telemetry-browser-forever

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
	@echo "New Relic load generator:"
	@echo "  make telemetry-fixture           Generate frontend/src/__tests__/fixtures/sample.mp4"
	@echo "  make telemetry-loop              Backend APM loop via vitest (HTTP-only)"
	@echo "  make telemetry-browser-deps      One-time: install host node_modules + Chromium"
	@echo "  make telemetry-browser-once      Single Playwright batch (real Chromium, host)"
	@echo "  make telemetry-browser-forever   Loop forever — Browser+Video Agent continuous data"
	@echo "                                   (override: TELEMETRY_ITERS=, TELEMETRY_PAUSE=, TELEMETRY_PLAYBACK_MS=)"
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

# --- NR telemetry loop (load generator) ---
# Generates a small MP4 once inside the backend container and copies it to the
# frontend test fixtures dir.
TELEMETRY_FIXTURE = frontend/src/__tests__/fixtures/sample.mp4
TELEMETRY_ITERS  ?= 10
TELEMETRY_PAUSE  ?= 1500

telemetry-fixture:
	@mkdir -p $(dir $(TELEMETRY_FIXTURE))
	docker compose exec -T backend ffmpeg -y -f lavfi \
	  -i color=c=cornflowerblue:size=640x360:r=15:d=8 \
	  -vf "drawtext=text='cat-loop':fontsize=72:fontcolor=white:x=(w-tw)/2:y=(h-th)/2" \
	  -c:v libx264 -pix_fmt yuv420p -movflags +faststart \
	  /tmp/telemetry-loop.mp4
	docker compose cp backend:/tmp/telemetry-loop.mp4 $(TELEMETRY_FIXTURE)
	@ls -la $(TELEMETRY_FIXTURE)

# Runs the load loop inside the frontend container so it can reach `backend`
# on the compose network. Override ITERS/PAUSE on the CLI:
#   make telemetry-loop TELEMETRY_ITERS=50 TELEMETRY_PAUSE=500
telemetry-loop:
	@test -f $(TELEMETRY_FIXTURE) || { echo "Run 'make telemetry-fixture' first"; exit 1; }
	docker compose exec \
	  -e LOAD=1 \
	  -e ITERS=$(TELEMETRY_ITERS) \
	  -e PAUSE_MS=$(TELEMETRY_PAUSE) \
	  -e API_BASE=http://backend:8080 \
	  -T frontend npx vitest run telemetry-loop

# --- Browser-driven NR loop (Playwright, host-side) ---
# Drives real Chromium against the running dev app (http://localhost:5173 +
# http://localhost:8080) so NR Browser Agent + Video Agent actually fire.
# Runs on the HOST — no Docker — to avoid compose-internal networking issues.
#
# Requirements (one-time): node + npm on the host, then:
#   make telemetry-browser-deps   # installs node_modules and Chromium
TELEMETRY_PLAYBACK_MS ?= 8000

telemetry-browser-deps:
	cd frontend && npm install --no-audit --no-fund
	cd frontend && npx playwright install chromium

# Single batch run — useful for quick verification.
telemetry-browser-once:
	@test -f $(TELEMETRY_FIXTURE) || { echo "Run 'make telemetry-fixture' first"; exit 1; }
	cd frontend && \
	  ITERS=$(TELEMETRY_ITERS) \
	  PLAYBACK_MS=$(TELEMETRY_PLAYBACK_MS) \
	  PAUSE_MS=$(TELEMETRY_PAUSE) \
	  API_BASE=http://localhost:8080 \
	  E2E_BASE_URL=http://localhost:5173 \
	  npx playwright test e2e/telemetry-loop.spec.ts --reporter=line

# Forever loop — keeps NR data flowing. Ctrl+C to stop.
telemetry-browser-forever:
	@test -f $(TELEMETRY_FIXTURE) || { echo "Run 'make telemetry-fixture' first"; exit 1; }
	ITERS_PER_BATCH=$(TELEMETRY_ITERS) \
	  PLAYBACK_MS=$(TELEMETRY_PLAYBACK_MS) \
	  PAUSE_MS=$(TELEMETRY_PAUSE) \
	  ./scripts/telemetry-browser-forever.sh

pounce: up
nap: down
stalk: logs
groom: fmt
sniff: test
