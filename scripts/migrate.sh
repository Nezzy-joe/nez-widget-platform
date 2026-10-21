#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if [[ ! -f .env ]]; then
  echo "Missing .env. Copy .env.example to .env and configure it first."
  exit 1
fi

set -a
source ./.env
set +a

: "${POSTGRES_DB:?POSTGRES_DB is required in .env}"
: "${POSTGRES_USER:?POSTGRES_USER is required in .env}"

docker compose up -d postgres

ready=0
for _ in $(seq 1 30); do
  if docker compose exec -T postgres \
    pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done

if [[ "$ready" != 1 ]]; then
  echo "PostgreSQL did not become ready within 30 seconds."
  exit 1
fi

for file in migrations/*.sql; do
  case "$(basename "$file")" in
    001_init.sql)
      check="SELECT to_regclass('public.tenants') IS NOT NULL
        AND to_regclass('public.widgets') IS NOT NULL
        AND to_regclass('public.widget_fields') IS NOT NULL
        AND to_regclass('public.submissions') IS NOT NULL;"
      ;;
    002_users.sql)
      check="SELECT to_regclass('public.users') IS NOT NULL;"
      ;;
    003_background_jobs.sql)
      check="SELECT to_regclass('public.background_jobs') IS NOT NULL;"
      ;;
    *)
      echo "No migration check defined for $file"
      exit 1
      ;;
  esac

  applied="$(docker compose exec -T postgres \
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "$check" |
    tr -d '[:space:]')"

  if [[ "$applied" == "t" ]]; then
    echo "Already applied: $file"
  else
    echo "Applying: $file"
    docker compose exec -T postgres \
      psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
      -v ON_ERROR_STOP=1 < "$file"
  fi
done

echo "Database migration check complete."
