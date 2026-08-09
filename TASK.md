# TASK.md — План проекта «Календарь бронирований»

## Описание
Проект web приложения `Календарь бронирований`. Упрощённый аналог cal.com без регистрации и авторизации.

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

### Этап 2 — TypeSpec-спецификация и генерация OpenAPI v3
Статус: выполнено.

- [x] Инструменты: `@typespec/http`, `@typespec/rest`, `@typespec/openapi`, эмиттер `@typespec/openapi3`.
- [x] Модели Owner/Event/Booking/Slot/Schedule + схемы запросов/ответов/ошибок (400/404/409), базовый путь `/api`.
- [x] Настройка эмиттера в `tspconfig.yaml`: генерация `spec/dist/openapi.yaml` и `openapi.json`.
- [x] Цель `make spec` = `npm ci` + `tsp compile .` → генерация `spec/dist/openapi.yaml` (+ `openapi.json`).
- [x] Критерий готовности: `make spec` проходит, сгенерированный OpenAPI v3 валиден (проверка валидатором Redocly), спецификация покрывает все пункты.

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
Статус: выполнено.

- [x] Реализация по сгенерированному `spec/dist/openapi.yaml` (ручная сверка путей/схем/ошибок с контрактом).
- [x] Миграции (SQLite), слой repository/service, HTTP-хендлеры, сид владельца.
- [x] Генератор слотов, антиконкурентная запись (уникальный индекс + транзакция с проверкой пересечений).
- [x] Валидация полей (email, длительность 15–480 мин).
- [x] Юнит-тесты на занятость/прошлое/окно/выходные/шаг сетки + интеграционный тест на 409.
- [x] Чек-лист готовности: `make test` проходит; каждый путь из OpenAPI реализован и совпадает с контрактом по схеме/ошибкам; сид владельца создаётся; 409 воспроизводится интеграционным тестом.

Реализация (Этап 4):
- Структура `backend/`: `main.go` + пакеты `internal/model` (типы Owner/Event/Booking/Slot/Schedule с JSON-тегами по контракту), `internal/store` (SQLite: миграции, repository, сид владельца, антиконкурентная запись), `internal/service` (бизнес-правила: генерация слотов, валидация, бронирование), `internal/httpapi` (HTTP-хендлеры по OpenAPI, middleware CORS/recover/logging).
- СУБД: `modernc.org/sqlite` (чистый Go, не требует CGO — Dockerfile собирается с `CGO_ENABLED=0`); DSN с `_pragma=busy_timeout(10000)` и `journal_mode=WAL`.
- Схема: `owners` (schedule в JSON), `events` (`duration_minutes` 15–480), `bookings` с уникальным индексом `UNIQUE (date, start_at)`; внешние ключи с `ON DELETE CASCADE`.
- Антиконкурентная запись: транзакция на отдельном соединении (`db.Conn` + `BEGIN IMMEDIATE`), внутри SELECT бронирований, пересекающих интервал (`date = ? AND start_at < ? AND end_at > ?`), затем INSERT; конфликт = 409, уникальный индекс ловит гонку на одинаковый `startAt`.
- Генерация слотов: шаг = длительность события, старт следующего слота сразу после конца предыдущего; слот целиком в рабочем времени дня из графика; исключаются слоты с `startAt <= now` и пересекающиеся с любым бронированием (любого события) за дату.
- Валидация бронирования: имя, email по `@pattern` из контракта, `startAt` в будущем, в окне `[сегодня, сегодня+13]`, день рабочий, слот умещается в рабочие часы и стоит на сетке `(минута - начало дня) % длительность == 0`.
- Ошибки: тело `{"error":{"code","message"}}`; 400 (`BAD_REQUEST`), 404 (`NOT_FOUND`), 409 (`CONFLICT`); все 8 путей контракта реализованы под `/api` (+ `/api/health`).
- Сид владельца: id `1` (совпадает с дефолтом на главной странице фронта), имя «Владелец календаря», график Пн–Пт 09:00–18:00, выходные закрыты; `INSERT ... ON CONFLICT(id) DO NOTHING`.
- Таймзона владельца = серверная: `time.Local` (в контейнере — UTC), переопределяется env `APP_TIMEZONE`; `now` инжектится в сервис для детерминированных тестов.
- Тесты: юнит-тесты сервиса (окно, выходные, прошлое, шаг сетки, занятость/соседние слоты, валидация 400, 409 конфликт и частичное наложение между разными событиями, конкурентная запись — ровно 1 успех из N), интеграционные тесты HTTP (полный сценарий, 409, 400, 404, формат тела ошибок), `go test -race ./...` проходит.
- Docker: `docker-compose.yml` — сервис `db` (alpine) теперь выполняет `chown -R 65534:65534 /data`, чтобы бэкенд (`USER nobody`) мог писать в volume; заглушка Этапа 3 заменена реальным бэкендом.
- Проверка: `make test` (go test ./...), `go vet`, `gofmt`, `CGO_ENABLED=0 go build`, `make build`; `docker compose up` поднимает 3 сервиса, сценарий «владелец создаёт событие → гость бронирует слот → дубль даёт 409 → бронь видна в /bookings» проходит через nginx-proxy; данные сохраняются в volume после рестарта бэкенда.

### Этап 5 — E2E-тестирование (Playwright), CI и автоматизация релизов
Статус: выполнено.

- [x] Описаны и зафиксированы основные пользовательские сценарии для проверки (см. «E2E-сценарии»).
- [x] Подключён Playwright; E2E-тесты покрывают основной сценарий бронирования и конфликт 409.
- [x] Настроен запуск тестов в CI через GitHub Actions.
- [x] Описан формат коммитов по Conventional Commits.
- [x] Подключён release-please как GitHub Actions workflow.
- [x] После мёджа в основную ветку release-please создаёт или обновляет release-PR с changelog и предложенной версией.

План (Этап 5):
- **E2E — Playwright**. Тесты живут в `frontend/` (`@playwright/test`, `playwright.config.ts`, сценарии в `frontend/e2e/`). Стек для тестов поднимается самим Playwright через `webServer`: бэкенд (`go run .` в `backend/`, свежая БД в `/tmp/booking-e2e.db`) + фронтенд (`npm run dev` с Vite proxy на бэкенд). `workers: 1`, `fullyParallel: false` (общий бэкенд), `locale: 'en-US'` — для детерминированных селекторов дат в календаре. Сценарии:
  1. Владелец создаёт событие через UI (`/owners/1`: название, описание, длительность 15–480 мин) и видит его в списке.
  2. Гость бронирует слот: страница события → следующий рабочий день в календаре → свободный слот → имя/email → подтверждение «Готово!».
  3. Владелец видит бронь в `/owners/1/bookings` (время, имя, email гостя).
  4. Конфликт 409: бронь того же слота дважды — вторая попытка отклоняется («Выбранный слот уже занят»).
  Детерминизм: «следующий рабочий день» (слоты строго в будущем, выходные закрыты), выбор дня по `data-day` календаря, при необходимости перелистывание месяца.
- **CI — GitHub Actions** (`.github/workflows/ci.yml`, не трогаем `hexlet-check.yml`): джобы `contract` (`make spec` + Redocly validate), `backend` (`make test`), `frontend` (`npm ci` + lint + build), `e2e` (`npm ci`, `playwright install --with-deps chromium`, `npm run test:e2e`). Для Node — setup-node 22, для Go — setup-go по `backend/go.mod`.
- **Conventional Commits**. Формат `type(scope): subject`, subject — на русском; `feat:` → minor, `fix:` → patch, `feat!:`/`BREAKING CHANGE` → major. Описание — в новом `CONTRIBUTING.md`, правило в `AGENTS.md` заменяет прежний формат «выполнен Этап N — …».
- **release-please** (`.github/workflows/release.yml`, `googleapis/release-please-action@v4`). Версионируется весь проект через корневой `package.json` (`booking-calendar`, `release-type: node`); тег `vX.Y.Z`, changelog генерируется автоматически. Токен — дефолтный `GITHUB_TOKEN`; если потребуется прогон CI на самом release-PR — нужен PAT-секрет (см. release-please-action).

Чек-лист готовности: `make test` зелёный; фронтенд собирается (`npm run build`, lint); `make spec` проходит; `npm run test:e2e` проходит локально и в CI — сценарии 1–4 покрывают основной сценарий бронирования; после пуш в `main` release-please создаёт/обновляет release-PR с changelog и версией; релизы формируются автоматически по истории коммитов.

Реализация (Этап 5):
- **E2E — Playwright**: `@playwright/test` ^1.62.1 добавлен в devDependencies `frontend/`; `frontend/playwright.config.ts` (`testDir: e2e/`, `workers: 1`, `fullyParallel: false`, `locale: 'en-US'`, `reporter: list`, скриншот при падении). Стек поднимает сам Playwright через `webServer` (массив из двух серверов): бэкенд `rm -f /tmp/booking-e2e.db* && go run .` (свежая БД, `DB_PATH=/tmp/booking-e2e.db`, `ADDR=:8080`, health-check `/api/health`) + фронтенд `npm run dev -- --strictPort` (Vite proxy на `http://localhost:8080`). Для локальных прогонов без скачанного Chromium предусмотрен `PLAYWRIGHT_USE_SYSTEM_CHROME=1` (канал `chrome`); в CI ставится штатный Chromium (`playwright install --with-deps chromium`).
- **Сценарии** в `frontend/e2e/booking-flow.spec.ts` (`test.describe.serial`) + хелперы в `frontend/e2e/utils.ts`: (1) владелец создаёт событие через UI и видит его в списке; (2) гость бронирует слот: «следующий рабочий день» → первый свободный слот → имя/email → «Готово!»; (3) владелец видит бронь в `/owners/1/bookings`; (4) конфликт 409: две страницы выбирают один и тот же слот, вторая попытка отклоняется («Выбранный слот уже занят»). Детерминизм: `nextWorkingDay()` (слоты строго в будущем, пн–пт), выбор дня по `data-day` (значение `toLocaleDateString()` в локали `en-US`, совпадает в Node и браузере), перелистывание месяца по aria-label «Go to the Next Month» при переходе через границу месяца (окно ≤ 14 дней → не больше одного листания). Конфликт 409 воспроизводится через две вкладки с «устаревшим» списком слотов: первая бронирует, вторая получает 409.
- **CI — GitHub Actions** (`.github/workflows/ci.yml`; `hexlet-check.yml` не тронут): джобы `contract` (`make spec` + `make spec-validate`), `backend` (`make test`, setup-go по `backend/go.mod`), `frontend` (`npm ci` + `npm run lint` + `npm run build`, setup-node 22 с кэшем по `frontend/package-lock.json`), `e2e` (`npm ci` → `playwright install --with-deps chromium` → `npm run test:e2e`; при падении загружается артефакт `frontend/test-results/`). Триггеры: push в `main` и pull_request; `concurrency` с отменой вхолостую запусков.
- **Conventional Commits**: новый `CONTRIBUTING.md` (формат `type(scope): subject`, subject — на русском, таблица типов и влияния на версию: `feat:` → minor, `fix:` → patch, `feat!:`/`BREAKING CHANGE` → major; перечень проверок перед коммитом). Правило в `AGENTS.md` заменяет прежний формат «выполнен Этап N — …».
- **release-please** (`.github/workflows/release.yml`, `googleapis/release-please-action@v4`): версионируется весь проект через корневой `package.json` (`booking-calendar`, версия `0.1.0`, `release-type: node`, `package-name: booking-calendar`); тег `vX.Y.Z`, changelog генерируется автоматически; токен — дефолтный `GITHUB_TOKEN`, права `contents: write` + `pull-requests: write`. Работает на push в `main`.
- **Makefile**: добавлены цели `spec-validate` и `test-e2e` (в `help` описаны). README (`./README.md` — бейдж CI + раздел «Запуск и проверки»; `frontend/README.md` — раздел про E2E-тесты) и `.gitignore` (артефакты Playwright: `test-results/`, `playwright-report/` и др. в `frontend/.gitignore`) обновлены.
- Проверка: `make test` зелёный; `npm run lint` (только прежние неблокирующие warnings в ui-компонентах) и `npm run build` проходят; `make spec` + `make spec-validate` проходят (1 неблокирующий warning про `info.license`); локально `npm run test:e2e` — 4/4 зелёные (~7.5 s).

### Этап 6 — Docker и деплой
Статус: выполнено.

- [x] В корне репозитория файл `Dockerfile`: по нему собирается образ и запускается приложение (одинаково локально, в CI и в облаке).
- [x] При запуске контейнера из собранного образа приложение стартует автоматически.
- [x] Приложение слушает порт из переменной окружения `PORT` (используется при деплое и в автоматической проверке проекта).
- [x] После деплоя у приложения есть публичная ссылка.
- [x] Собран Docker-образ приложения и проверена его работа.
- [x] Настроен деплой на Render (запуск по `PORT`, проверка работы приложения); если Render требует оплату или недоступен — тот же образ деплоится на Railway (запуск по `PORT`, публичная ссылка).
- [x] В репозиторий добавлена ссылка на опубликованное приложение.

Результат:
- Есть `Dockerfile` для сборки образа; приложение запускается в контейнере по `PORT`.

План (Этап 6):
- **Единый Dockerfile** в корне репозитория (многостадийный): stage `node:22-alpine` — сборка фронтенда (`npm ci` + `npm run build`, `dist/`); stage `golang:1.26-alpine` — сборка бэкенда (`CGO_ENABLED=0 go build`); runtime `nginx:1.27-alpine` — SPA + бинарь бэкенда, оба процесса в одном контейнере.
- **Запуск по `PORT`**: nginx-конфиг собирается из шаблона `nginx.conf.template` (`listen ${PORT};`, официальный envsubst-механизм nginx — переменные nginx `$uri`/`$host` не подставляются); `/api/` проксируется на `127.0.0.1:8080`, статика SPA отдаётся через `try_files`.
- **Entrypoint**: `docker-entrypoint.sh` стартует бэкенд на `127.0.0.1:8080` в фоне (с wait на `/api/health`) и запускает официальный `/docker-entrypoint.sh` nginx; trap на TERM/INT для корректной остановки. `HEALTHCHECK` на `/api/health`, `WORKDIR /srv` для записи `booking.db`.
- **Deploy**: `render.yaml` (blueprint: web-сервис, `runtime: docker`, `dockerfilePath: ./Dockerfile`, `healthCheckPath: /api/health`, план free) + деплой через Render MCP (API ключ для Render запроси во время выполнения); запасной вариант — Railway (тот же Dockerfile, `PORT`, публичная ссылка).
- **Публичная ссылка**: добавляется в `README.md` (блок «Демо»).

Чек-лист готовности: `docker build .` проходит; контейнер стартует автоматически и отвечает по `PORT` (`/` — SPA, `/api/health` — 200); сценарий «владелец создаёт событие → гость бронирует слот → дубль даёт 409» проходит через один контейнер; приложение задеплоено и доступно по публичной ссылке (Render, при проблемах — Railway); ссылка на приложение добавлена в README.md.

Реализация (Этап 6):
- **Единый Dockerfile** в корне репозитория (многостадийный): stage `node:22-alpine` — сборка фронтенда (`npm ci` + `npm run build`); stage `golang:1.26-alpine` — сборка бэкенда (`CGO_ENABLED=0 go build`); runtime `nginx:1.27-alpine` — SPA и бинарь бэкенда в одном контейнере. Образ ~64MB, `HEALTHCHECK` на `/api/health`, `WORKDIR /srv` для записи `booking.db`.
- **Запуск по `PORT`**: `nginx.conf.template` (`listen ${PORT};`) обрабатывается официальным envsubst-механизмом nginx (шаблон кладётся в `/etc/nginx/templates/`; подставляются только определённые в окружении переменные — `$uri`/`$host` не трогаются); `/api/` проксируется на внутренний `127.0.0.1:8081`, статика SPA отдаётся через `try_files`. В образе задан дефолт `ENV PORT=8080`.
- **Нюанс портов**: бэкенд слушает внутренний `127.0.0.1:8081`, а не `8080`, — при дефолтном `PORT=8080` публичный порт nginx и бэкенд конфликтовали за один порт (`Address in use`). На Render `PORT` всегда назначается платформой, но образ обязан работать и с дефолтным значением.
- **Entrypoint**: `docker-entrypoint.sh` стартует бэкенд в фоне, ждёт `/api/health` (до 30 с), затем запускает официальный `/docker-entrypoint.sh nginx -g "daemon off;"`; trap `TERM/INT/QUIT` корректно останавливает оба процесса.
- **Нюанс STOPSIGNAL**: базовый образ nginx задаёт `STOPSIGNAL=SIGQUIT`, поэтому `docker stop` шлёт контейнеру `SIGQUIT`, а не `SIGTERM`; без trap на `QUIT` контейнер зависал на 10 с и убивался по `SIGKILL` (диагностировано на busybox/nginx-образах: trap срабатывает при прямом `kill -TERM 1`, но не при `docker stop`).
- **Deploy**: `render.yaml` (blueprint: web-сервис, `runtime: docker`, `dockerfilePath: ./Dockerfile`, `healthCheckPath: /api/health`, план free) + деплой через Render API (ключ из `APIKEY.md`, в `.gitignore`). На аккаунте уже существовал сервис `ai-for-developers-project-386` (создан из этого репо), его деплой на коммите до Этапа 6 падал `build_failed` (в репо не было корневого Dockerfile) — после пуша Этапа 6 передеплоен и поднялся.
- **Публичная ссылка**: https://ai-for-developers-project-386-qitn.onrender.com — добавлена в README.md (блок «Демо»).
- Проверка: `docker build .` проходит; контейнер стартует автоматически, отвечает по `PORT` (`/` — SPA, `/api/health` — 200, deep-link SPA-фолбэк 200); сценарий «создать событие → гость бронирует слот → дубль 409 → бронь в списке» проходит через один контейнер; `docker stop` — graceful exit (0); после деплоя приложение доступно по публичной ссылке и отвечает на `/api/health` 200.

## Открытые решения (дефолты)
- **Маршруты**: все операции событий и бронирований строятся по `ownerId` + `eventId` (соответствует фронт-URL).
- **Тело POST `/bookings`**: `name`, `email`, `startAt` — слот выбирает гость, сервер сам вычисляет `endAt` по длительности события.
- **Формат OpenAPI-артефакта**: основной — `yaml`, дополнительно `json`; `spec/dist/` генерируется и не коммитится.
- **Часовой пояс**: слоты считаются в таймзоне владельца (она же серверная), `date` интерпретируется в ней же; в API время передаётся в UTC ISO-8601.
- **Шаг сетки**: шаг = длительности события (старт следующего слота сразу после конца предыдущего).
- **Окно записи**: 14 календарных дней, включая текущий день (`[сегодня, сегодня + 13]`).
- **График по умолчанию**: Пн–Пт 09:00–18:00, выходные закрыты (настраивается владельцем).

Все этапы по первой части курса завершены.
