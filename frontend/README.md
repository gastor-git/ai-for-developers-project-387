# Frontend — Календарь бронирований

Клиентская часть приложения: TypeScript + Vite + React + shadcn/ui (Tailwind CSS + Radix UI).
Реализована вручную по контракту `spec/dist/openapi.yaml` (генерация кода не применяется).

## Стек

- Роутинг: `react-router-dom`
- Иконки: `lucide-react`
- Календарь: `react-day-picker` (обёртка `calendar` из shadcn/ui)
- Даты: `date-fns`

## Запуск

```sh
npm ci
npm run dev
```

Dev-сервер доступен на http://localhost:5173. Все запросы к API идут на относительный
путь `/api/...`; Vite proxy в `vite.config.ts` проксирует их на бэкенд
(`VITE_API_PROXY_TARGET`, по умолчанию `http://localhost:8080`).

Базовый URL API берётся из `VITE_API_BASE_URL` (по умолчанию пусто — относительные пути).

## Работа с Prism (без бэкенда)

Эмулятор API по контракту (мок-ответы из OpenAPI):

```sh
# из spec/
npx @stoplight/prism-cli mock dist/openapi.yaml -p 8080

# из frontend/ (другой терминал)
VITE_API_PROXY_TARGET=http://localhost:8080 \
VITE_API_PROXY_STRIP_PREFIX=true \
npm run dev
```

Prism не учитывает базовый путь `servers.url` (`/api`) из спецификации, поэтому для
эмулятора Vite proxy срезает префикс `/api` (`VITE_API_PROXY_STRIP_PREFIX=true`).

## Сборка и линт

```sh
npm run build   # tsc -b && vite build
npm run lint    # oxlint
```

Docker-образ собирается многостадийным Dockerfile (node → nginx). В проде nginx
отдаёт статику и проксирует `/api` на сервис `backend` (см. `nginx.conf`).
