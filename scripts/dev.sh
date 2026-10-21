#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

for command in docker go npm curl; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Required command not found: $command"
    exit 1
  fi
done

if [[ ! -f .env ]]; then
  echo "Missing .env. Copy .env.example to .env and configure it first."
  exit 1
fi

for port in 8080 5173; do
  if ss -ltn "sport = :${port}" |
      awk 'NR > 1 { found = 1 } END { exit !found }'; then
    echo "Port ${port} is already in use."
    echo "Stop the existing server, then run ./scripts/dev.sh again."
    exit 1
  fi
done

mkdir -p tmp

docker compose up -d postgres
./scripts/migrate.sh

if [[ ! -d web/node_modules ]]; then
  (cd web && npm ci)
fi

go build -o tmp/nez-widget-api ./cmd/api

API_PID=""
WEB_PID=""

cleanup() {
  echo
  echo "Stopping development servers..."
  [[ -z "$WEB_PID" ]] || kill "$WEB_PID" 2>/dev/null || true
  [[ -z "$API_PID" ]] || kill "$API_PID" 2>/dev/null || true
  [[ -z "$WEB_PID" ]] || wait "$WEB_PID" 2>/dev/null || true
  [[ -z "$API_PID" ]] || wait "$API_PID" 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

PORT=8080 ./tmp/nez-widget-api > tmp/api.log 2>&1 &
API_PID=$!

ready=0
for _ in $(seq 1 40); do
  if ! kill -0 "$API_PID" 2>/dev/null; then
    echo "API exited before becoming healthy. See tmp/api.log"
    cat tmp/api.log
    exit 1
  fi

  if curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done

if [[ "$ready" != 1 ]]; then
  echo "API did not become healthy. See tmp/api.log"
  cat tmp/api.log
  exit 1
fi

(cd web && exec ./node_modules/.bin/vite --host 127.0.0.1 --strictPort) \
  > tmp/web.log 2>&1 &
WEB_PID=$!

sleep 2
if ! kill -0 "$WEB_PID" 2>/dev/null; then
  echo "Dashboard failed to start. See tmp/web.log"
  cat tmp/web.log
  exit 1
fi

echo
echo "Development environment is running."
echo "Dashboard: http://127.0.0.1:5173"
echo "API:       http://127.0.0.1:8080"
echo "Health:    http://127.0.0.1:8080/health"
echo "API logs:  tmp/api.log"
echo "Web logs:  tmp/web.log"
echo "Press Ctrl+C to stop both servers."

while kill -0 "$API_PID" 2>/dev/null &&
      kill -0 "$WEB_PID" 2>/dev/null; do
  sleep 1
done

echo "A development server stopped. Check the logs above."
exit 1
