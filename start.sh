#!/bin/sh

set -e

echo "run db migrations"
source /app/app.env
/app/migrate -path /app/db/migration -database "$DB_SOURCE" -verbose up

echo "Starting the application..."
exec "$@"
