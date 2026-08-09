.PHONY: help spec spec-validate dev test test-e2e build

help:
	@echo "Usage:"
	@echo "  make spec          Generate OpenAPI contract from TypeSpec (spec/dist)"
	@echo "  make spec-validate Validate generated OpenAPI with Redocly"
	@echo "  make dev           Run all services via docker compose"
	@echo "  make test          Run backend tests"
	@echo "  make test-e2e      Run Playwright E2E tests (frontend/e2e)"
	@echo "  make build         Build docker images"

spec:
	cd spec && npm ci && npx tsp compile .

spec-validate:
	cd spec && npm run validate

dev:
	docker compose up --build

test:
	cd backend && go test ./...

test-e2e:
	cd frontend && npm run test:e2e

build:
	docker compose build
