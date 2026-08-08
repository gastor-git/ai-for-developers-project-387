# TASK.md — План проекта «Календарь бронирований»

## Описание
Проект web приложения `Календарь бронирований` по PLAN.md. Упрощённый аналог cal.com без регистрации и авторизации.

Роли: `Владелец календаря` (один фиксированный профиль) и `Гость` (бронирует слоты без аккаунта).

## Стек
- REST API
- Бэкенд: Go
- Фронтенд: TypeScript + Vite + React + shadcn/ui
- БД: SQLite
- Запуск: docker (backend, frontend, db — отдельные сервисы)

## Структура репозитория
```
hexlet/
├── PLAN.md            # исходное описание (не трогаем)
├── TASK.md            # этот файл: подробный план + чек-листы
├── Makefile           # make spec / dev / test / build
├── docker-compose.yml # backend + frontend + db (3 сервиса)
├── .env.example       # переменные окружения для docker compose
├── .gitignore         # игнор: node_modules/, spec/dist/, .env и др.
├── spec/              # TypeSpec-спецификация
│   ├── package.json   # зависимости @typespec/*, @typespec/openapi, @redocly/cli + скрипты compile/validate
│   ├── package-lock.json
│   ├── tspconfig.yaml # конфиг компиляции + эмиттер OpenAPI v3 (openapi.yaml + openapi.json)
│   ├── .redocly.yaml  # настройка валидации сгенерированного OpenAPI (Redocly, правила recommended)
│   ├── main.tsp       # сервис, базовые декораторы, теги
│   ├── models.tsp     # модели Owner/Event/Booking/Slot + валидация полей
│   ├── errors.tsp     # типы ошибок (400/404/409)
│   ├── operations.tsp # маршруты API (вложенные namespace)
│   └── dist/          # сгенерированные openapi.yaml / openapi.json (.gitignore)
├── backend/           # Go (REST API, SQLite)
└── frontend/          # TypeScript + Vite + React + shadcn/ui
```

## Доменная модель
- **Owner** — один фиксированный профиль (id, имя) + настраиваемый график работы (по дням недели: начало/конец рабочего дня, выходные).
- **Event** — id, ownerId, название, описание, длительность в минутах.
- **Booking** — id, eventId, дата, startAt/endAt (UTC), имя и email гостя, createdAt.

## API-контракт (для TypeSpec)
| Метод | Путь | Назначение |
|---|---|---|
| GET | `/api/owners/{ownerId}` | профиль владельца + график |
| PATCH | `/api/owners/{ownerId}/schedule` | обновление графика (без авторизации, т.к. её нет) |
| GET | `/api/owners/{ownerId}/events` | события владельца |
| POST | `/api/owners/{ownerId}/events` | создать событие |
| GET | `/api/owners/{ownerId}/events/{eventId}` | страница события для гостя |
| GET | `/api/owners/{ownerId}/events/{eventId}/slots?date=YYYY-MM-DD` | свободные слоты на дату |
| POST | `/api/owners/{ownerId}/events/{eventId}/bookings` | создать бронирование (name, email, startAt выбранного слота) |
| GET | `/api/owners/{ownerId}/bookings` | все бронирования (владелец) |

Ошибки: `400` (невалидный запрос, в т.ч. нарушение валидации полей), `404` (не найдено), `409` (слот занят).

Валидация (ответ `400`): email гостя обязателен и валиден по формату; длительность события — целое число от 15 до 480 минут.

## Артефакт контракта (OpenAPI v3)
- Из TypeSpec-спецификации эмиттером `@typespec/openapi3` генерируется документ `spec/dist/openapi.yaml` (+ `openapi.json`) — эталон API-контракта.
- Бэкенд и фронтенд разрабатываются по этому документу вручную (генерация кода по OpenAPI не применяется).
- `spec/dist/` в `.gitignore`; воспроизводимость артефакта обеспечивает `make spec` (проверяется в CI).

## Бизнес-правила генерации слотов
- Окно: `[сегодня, сегодня + 13]` — ровно 14 календарных дней; даты вне окна не принимаются (отказ `400` или пустой список).
- `date=YYYY-MM-DD` интерпретируется в таймзоне владельца (она же серверная).
- Слот длиной = длительности события; старт каждого следующего слота — сразу после конца предыдущего; слот должен целиком умещаться в рабочее время из графика владельца.
- Слот свободен, только если весь интервал не пересекается ни с одним бронированием (любого события) и его `startAt` в будущем (`> now`).
- Антиконкурентная запись: уникальный индекс `(date, startAt)` + транзакция `BEGIN IMMEDIATE`, в которой перед INSERT выполняется SELECT бронирований, пересекающих интервал (ловит и частичные наложения при разных `startAt`).

## Этапы

### Этап 1 — Каркас
Статус: выполнено.

- [x] Структура репозитория (spec/, backend/, frontend/).
- [x] Makefile, docker-compose.yml, `.gitignore` (в т.ч. `spec/dist/`), `.env.example`.
- [x] В `spec/`: `package.json` (зависимости `@typespec/*`), `tspconfig.yaml` с настройкой эмиттера `@typespec/openapi3`.

Реализация (Этап 1):
- Версии TypeSpec: `@typespec/compiler`, `@typespec/http`, `@typespec/openapi3` — `^1.14.0`; `@typespec/rest` — `^0.84.0`. Требуется Node >= 22.
- `spec/package-lock.json` закоммичен для воспроизводимости; `make spec` = `npm ci` + `npx tsp compile .`.
- `tspconfig.yaml` изначально использовал блок `emitters`; в Этапе 2 переведён на синтаксис TypeSpec 1.14 (`emit` + `options`, см. ниже).
- `docker-compose.yml`: сервис `db` (alpine, держит volume `sqlite-data` в `/data`), `backend` пишет в тот же volume через env `DB_PATH`, `frontend`. Порты берутся из `.env` (дефолты: `BACKEND_PORT=8080`, `FRONTEND_PORT=5173`); `docker compose config` проходит.
- `backend/` и `frontend/` пока пустые (`.gitkeep`).

### Этап 2 — TypeSpec-спецификация и генерация OpenAPI v3 (главный этап по PLAN.md)
Статус: выполнено.

- [x] Инструменты: `@typespec/http`, `@typespec/rest`, `@typespec/openapi`, эмиттер `@typespec/openapi3`.
- [x] Модели Owner/Event/Booking/Slot/Schedule + схемы запросов/ответов/ошибок (400/404/409), базовый путь `/api`.
- [x] Настройка эмиттера в `tspconfig.yaml`: генерация `spec/dist/openapi.yaml` и `openapi.json`.
- [x] Цель `make spec` = `npm ci` + `tsp compile .` → генерация `spec/dist/openapi.yaml` (+ `openapi.json`).
- [x] Критерий готовности: `make spec` проходит, сгенерированный OpenAPI v3 валиден (проверка валидатором Redocly), спецификация покрывает все пункты PLAN.md.

Реализация (Этап 2):
- Файлы: `main.tsp` (сервис, `@service`/`@info`/`@useAuth`/`@server`/`@tagMetadata`), `models.tsp`, `errors.tsp`, `operations.tsp`.
- `tspconfig.yaml` на синтаксисе TypeSpec 1.14: `output-dir: {project-root}/dist` + `emit: [@typespec/openapi3]` + `options` (блок `emitters` устарел). Для вывода обоих форматов задан `emitter-output-dir: {project-root}/dist` и `file-type: [yaml, json]`.
- Нюансы TypeSpec 1.14: объектные литералы в аргументах декораторов — синтаксис `#{...}`; версия API задаётся `@info` из `@typespec/openapi` (в `@service` только `title`); вложенные `interface` запрещены — маршруты строятся через вложенные `namespace` с `@route`.
- API без авторизации зафиксирован явно: `@useAuth(NoAuth)` → в OpenAPI эмитится `security: [{ }]`.
- Валидация в контракте: email гостя — `@pattern` (обязателен, валидный формат); длительность события — `@minValue(15)`/`@maxValue(480)`; дата — `@pattern(^\d{4}-\d{2}-\d{2}$)`.
- Валидация артефакта: `npm run validate` = `redocly lint dist/openapi.yaml` (`.redocly.yaml`, правила `recommended`). Проверка проходит; остаётся 1 неблокирующий warning про `info.license` (лицензия проекта не задана).
- В пакет добавлены зависимости `@typespec/openapi` (^1.14.0) и `@redocly/cli` (^1.34.0) + скрипт `validate`; `package-lock.json` пересобран.
- Требуется Node >= 22 (в системе установлен локально, `~/.local/opt/node`).

### Этап 3 — Фронтенд (TypeScript + Vite + React + shadcn/ui)
Статус: выполнено.

- [x] Реализация по сгенерированному `spec/dist/openapi.yaml` (ручная сверка с контрактом).
- [x] Стек: TypeScript + Vite + React + shadcn/ui (Tailwind CSS + Radix UI); роутинг — react-router-dom; иконки — lucide-react; календарь — react-day-picker (обёртка `calendar` из shadcn/ui); даты — date-fns.
- [x] API-клиент пишется вручную под каждый путь OpenAPI (генерация кода не применяется): единый модуль с функциями под owner/events/slots/bookings; базовый URL берётся из `VITE_API_BASE_URL` (в dev — Vite proxy на отдельно запущенный бэкенд).
- [x] Для разработки и проверки интерфейса без запущенного бэкенда используется Stoplight Prism — эмулятор API по контракту: `npx @stoplight/prism-cli mock spec/dist/openapi.yaml` (мок-ответы по сгенерированному OpenAPI).
- [x] Фронтенд — отдельная часть приложения: получает данные и выполняет действия только через API по контракту; корректно работает с отдельно запущенным бэкендом.
- [x] Страница владельца (`/owners/:id`): профиль владельца, создание события (название, описание, длительность 15–480 мин), список событий, копирование ссылки на событие в буфер обмена.
- [x] Страница события (`/owners/:id/events/:eventId`): название/описание/длительность, календарь на 14 дней, свободные слоты на выбранную дату, форма бронирования (имя, email), сообщение об успехе.
- [x] Страница `/owners/:id/bookings`: список всех бронирований владельца.
- [x] Ошибки API (400/404/409) отображаются пользователю; локальная валидация форм (email, длительность) до отправки запроса.
- [x] Чек-лист готовности: создание события и копирование ссылки работают; календарь на 14 дней показывает только свободные слоты (даты вне окна недоступны); бронирование проходит и появляется в `/bookings`; ошибки API (400/404/409) отображаются пользователю; фронт собирается без ошибок (`make build`); интерфейс корректно работает с отдельно запущенным бэкендом и с Prism-эмулятором по контракту.

Реализация (Этап 3):
- Версии (frontend/package.json): Vite ^8, React ^19, react-router-dom ^7, react-day-picker ^10, date-fns ^4, Tailwind CSS v4; shadcn/ui инициализирован на Radix UI (пресет Nova, `style: radix-nova` в components.json).
- Типы и API-клиент в `src/lib/api.ts` по OpenAPI-контракту: `getOwner`, `updateSchedule`, `listEvents`, `createEvent`, `getEvent`, `getSlots`, `createBooking`, `listBookings`; ошибка — класс `ApiError` (статус + `error.code`/`error.message`); базовый URL — `VITE_API_BASE_URL` (по умолчанию относительный `/api/...`).
- Роуты: `/` (ввод id владельца), `/owners/:ownerId`, `/owners/:ownerId/events/:eventId`, `/owners/:ownerId/bookings`.
- Окно записи 14 дней (`[сегодня, сегодня + 13]`) — в календаре (react-day-picker) даты вне окна блокируются матчерами `{ before }` / `{ after }`; слоты на дату грузятся по GET `/slots?date=`.
- Локальная валидация: длительность события — целое 15–480; email гостя — по `@pattern` из контракта (тот же regex, что в OpenAPI).
- Vite dev: proxy `/api` → `VITE_API_PROXY_TARGET` (по умолчанию `http://localhost:8080`). Нюанс Prism 5.x: эмулятор игнорирует базовый путь `servers.url` (`/api`), поэтому для него задаётся `VITE_API_PROXY_STRIP_PREFIX=true` (прокси срезает префикс `/api`); с реальным бэкендом префикс не срезается.
- Dockerfile (многостадийный node → nginx, HEALTHCHECK), nginx.conf проксирует `/api` на сервис `backend:8080` и отдаёт SPA (`try_files`). Чтобы `make build` проходил на Этапе 3, в `backend/` добавлен заглушка-контейнер (Dockerfile + тривиальный `main.go` на 8080) — в Этапе 4 заменяется реальным бэкендом.
- Проверка: `npm run build` (tsc + vite), `npm run lint` (oxlint), `make build` (docker compose build), `make test` (пусто до Этапа 4); все 8 путей контракта отвечают через Vite proxy → Prism; `docker compose up` поднимает 3 сервиса, frontend healthy.

### Этап 4 — Бэкенд (Go)
- Реализация по сгенерированному `spec/dist/openapi.yaml` (ручная сверка путей/схем/ошибок с контрактом).
- Миграции (SQLite), слой repository/service, HTTP-хендлеры, сид владельца.
- Генератор слотов, антиконкурентная запись (уникальный индекс + транзакция с проверкой пересечений).
- Валидация полей (email, длительность 15–480 мин).
- Юнит-тесты на занятость/прошлое/окно/выходные/шаг сетки + интеграционный тест на 409.
- Чек-лист готовности: `make test` проходит; каждый путь из OpenAPI реализован и совпадает с контрактом по схеме/ошибкам; сид владельца создаётся; 409 воспроизводится интеграционным тестом.

### Этап 5 — Docker и приёмка
- Dockerfile для backend/frontend, docker-compose, CORS, README с инструкцией запуска.
- E2E-проверка по чек-листу PLAN.md, прогон Hexlet CI.
- Чек-лист готовности: `docker compose up` поднимает все 3 сервиса; сценарий «владелец создаёт событие → гость бронирует слот → бронь видна владельцу» проходит через полный стек; CI (Hexlet Check) зелёный.

## Открытые решения (дефолты)
- **Маршруты**: все операции событий и бронирований строятся по `ownerId` + `eventId` (соответствует PLAN.md и фронт-URL).
- **Тело POST `/bookings`**: `name`, `email`, `startAt` — слот выбирает гость, сервер сам вычисляет `endAt` по длительности события.
- **Формат OpenAPI-артефакта**: основной — `yaml`, дополнительно `json`; `spec/dist/` генерируется и не коммитится.
- **Часовой пояс**: слоты считаются в таймзоне владельца (она же серверная), `date` интерпретируется в ней же; в API время передаётся в UTC ISO-8601.
- **Шаг сетки**: шаг = длительности события (старт следующего слота сразу после конца предыдущего).
- **Окно записи**: 14 календарных дней, включая текущий день (`[сегодня, сегодня + 13]`).
- **График по умолчанию**: Пн–Пт 09:00–18:00, выходные закрыты (настраивается владельцем).
