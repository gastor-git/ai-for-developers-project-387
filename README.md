# Календарь бронирований

Web-приложение — упрощённый аналог cal.com. Гость бронирует слоты без регистрации,
владелец календаря настраивает события и график работы.

## Демо

Приложение развёрнуто на Render (единый контейнер с SPA и API):

<https://ai-for-developers-project-386-qitn.onrender.com>

> Free-план Render усыпляет сервис после простоя — первый запрос после паузы
> занимает 30–60 секунд.

## Статус

[![Hexlet tests and linter status](https://github.com/gastor-git/ai-for-developers-project-386/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/gastor-git/ai-for-developers-project-386/actions/workflows/hexlet-check.yml)
[![CI](https://github.com/gastor-git/ai-for-developers-project-386/actions/workflows/ci.yml/badge.svg)](https://github.com/gastor-git/ai-for-developers-project-386/actions/workflows/ci.yml)

## Документация

- `TASK.md` — план, этапы, чек-листы, API-контракт.
- `CONTRIBUTING.md` — правила оформления коммитов (Conventional Commits).
- `spec/dist/openapi.yaml` — эталонный OpenAPI-контракт (генерируется `make spec`).

## Стек

- Бэкенд: Go (REST API, SQLite)
- Фронтенд: TypeScript + Vite + React + shadcn/ui
- Контракт: TypeSpec → OpenAPI v3
- E2E: Playwright
- Запуск: Docker (backend, frontend, db)

## Запуск и проверки

```sh
make dev          # поднять все сервисы (docker compose up --build)
make spec         # сгенерировать OpenAPI из TypeSpec
make spec-validate# валидация контракта Redocly
make test         # тесты бэкенда
make test-e2e     # E2E-тесты Playwright
make build        # сборка docker-образов
```
