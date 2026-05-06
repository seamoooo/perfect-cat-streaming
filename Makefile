.PHONY: help up down logs build fmt test pounce nap stalk groom sniff init-env

help:
	@echo "perfect-cat-streaming -- Bincho × Kanpachi HLS streamer"
	@echo ""
	@echo "Standard targets:"
	@echo "  make up        Build & start all services (docker compose up --build)"
	@echo "  make down      Stop & remove containers"
	@echo "  make logs      Tail logs"
	@echo "  make build     Build images only"
	@echo "  make fmt       Format Go + frontend"
	@echo "  make test      Run go test + vitest"
	@echo "  make init-env  Copy .env.example -> .env if missing"
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

pounce: up
nap: down
stalk: logs
groom: fmt
sniff: test
