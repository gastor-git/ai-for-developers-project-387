# syntax=docker/dockerfile:1

########################
# Frontend build (Vite)
########################
FROM node:22-alpine AS frontend-builder

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./

ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

########################
# Backend build (Go)
########################
FROM golang:1.26-alpine AS backend-builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend .

########################
# Runtime (nginx + backend)
########################
FROM nginx:1.27-alpine

ENV PORT=8080 \
    ADDR=127.0.0.1:8081 \
    DB_PATH=/srv/booking.db

WORKDIR /srv

RUN apk add --no-cache su-exec && \
    mkdir -p /srv && chown nginx:nginx /srv

COPY --from=backend-builder /out/backend /usr/local/bin/backend
COPY --from=frontend-builder /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
  CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/api/health" >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
