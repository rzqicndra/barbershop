#!/usr/bin/env bash
set -euo pipefail
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v runtime_password="$APP_DB_PASSWORD" -v database="$POSTGRES_DB" <<'SQL'
CREATE ROLE gilir_app LOGIN PASSWORD :'runtime_password';
GRANT CONNECT ON DATABASE :"database" TO gilir_app;
GRANT USAGE ON SCHEMA public TO gilir_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gilir_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO gilir_app;
SQL
