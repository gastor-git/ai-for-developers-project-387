.PHONY: help spec dev test build

help:
	@echo "Usage:"
	@echo "  make spec   Generate OpenAPI contract from TypeSpec (spec/dist)"
	@echo "  make dev    Run all services via docker compose"
	@echo "  make test   Run backend tests"
	@echo "  make build  Build docker images"

spec:
	cd spec && npm ci && npx tsp compile .

dev:
	docker compose up --build

test:
	cd backend && go test ./...

build:
	docker compose build
