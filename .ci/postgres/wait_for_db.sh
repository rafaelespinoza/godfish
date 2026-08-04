#!/usr/bin/env sh
set -eu

MAX_NUM_ATTEMPTS="${MAX_NUM_ATTEMPTS:-10}"
SLEEP_DURATION_SECONDS="${SLEEP_DURATION_SECONDS:-1}"
dbhost="${1:?dbhost required}"
dbname="${2:?dbname required}"
dbuser="${3:?dbuser required}"

num_attempts=0
until PGPASSWORD="${POSTGRES_PASSWORD:?missing POSTGRES_PASSWORD}" psql -h "${dbhost}" -U "${dbuser}" "${dbname}" -c '\q' 2>/dev/null; do
    num_attempts=$((num_attempts+1))
    if [ $num_attempts -gt "${MAX_NUM_ATTEMPTS}" ]; then
        echo "ERROR: max attempts exceeded waiting for Postgres" >&2
        exit 1
    fi
    echo "Postgres is unavailable now, sleeping..." >&2
    sleep "${SLEEP_DURATION_SECONDS}"
done
echo "Postgres is up" >&2
