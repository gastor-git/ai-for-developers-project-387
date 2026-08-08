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
- `make dev` — поднять все сервисы (docker compose up --build)
- `make test` — тесты бэкенда (cd backend && go test ./...)
- `make build` — сборка docker-образов
- `npm run validate` (в spec/) — валидация OpenAPI через Redocly

## Конвенции и правила
- Сообщения коммитов — на русском, формат «выполнен Этап N — краткое описание»
- Бэкенд и фронтенд разрабатываются строго по контракту spec/dist/openapi.yaml.
