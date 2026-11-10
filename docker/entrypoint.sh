#!/bin/sh
# Apply database migrations, then start the app.
set -e
goose -dir /app/migrations postgres "$PBW_POSTGRES_CONN_STRING" up
exec /app/app
