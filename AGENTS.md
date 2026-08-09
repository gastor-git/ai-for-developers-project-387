# AGENTS.md

## Обзор
Web-приложение «Календарь бронирований» — упрощённый аналог cal.com.
Две роли: Владелец календаря (один фиксированный профиль) и Гость (бронирует без аккаунта).
Регистрации и авторизации нет.

Ссылки:
- TASK.md — подробный план, этапы, чек-листы, доменная модель, API-контракт
- README.md — статус CI

## Стек
- Бэкенд: Go (REST API, SQLite)
- Фронтенд: TypeScript + Vite + React + shadcn/ui
- Контракт: TypeSpec → OpenAPI v3 в spec/dist/openapi.yaml
- Запуск: Docker (backend, frontend, db)

## Команды
- `make spec` — сгенерировать OpenAPI из TypeSpec (npm ci + tsp compile)
- `make spec-validate` — валидация OpenAPI через Redocly
- `make dev` — поднять все сервисы (docker compose up --build)
- `make test` — тесты бэкенда (cd backend && go test ./...)
- `make test-e2e` — E2E-тесты Playwright (cd frontend && npm run test:e2e)
- `make build` — сборка docker-образов

## Конвенции и правила
- Сообщения коммитов — по Conventional Commits: `type(scope): subject`, subject — на русском; `feat:` → minor, `fix:` → patch, `feat!:`/`BREAKING CHANGE` → major (подробнее в CONTRIBUTING.md). История коммитов анализируется release-please для генерации релизов.
- Бэкенд и фронтенд разрабатываются строго по контракту spec/dist/openapi.yaml.
