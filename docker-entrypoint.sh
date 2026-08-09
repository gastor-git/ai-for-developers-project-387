#!/bin/sh
set -e

backend_pid=""
nginx_pid=""

log() {
  echo "[entrypoint] $*"
}

stop_all() {
  log "получен сигнал, останавливаю процессы"
  [ -n "$backend_pid" ] && kill "$backend_pid" 2>/dev/null || true
  [ -n "$nginx_pid" ] && kill "$nginx_pid" 2>/dev/null || true
  exit 0
}
trap stop_all TERM INT QUIT

log "запускаю бэкенд на $ADDR (db=$DB_PATH)"
/usr/local/bin/backend &
backend_pid=$!

tries=0
until wget -qO- http://127.0.0.1:8081/api/health >/dev/null 2>&1; do
  tries=$((tries + 1))
  if [ "$tries" -ge 30 ]; then
    log "бэкенд не поднялся за 30 секунд"
    kill "$backend_pid" 2>/dev/null || true
    exit 1
  fi
  sleep 1
done
log "бэкенд готов на 127.0.0.1:8081"

/docker-entrypoint.sh nginx -g "daemon off;" &
nginx_pid=$!

wait "$nginx_pid"
