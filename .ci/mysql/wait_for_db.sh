#!/usr/bin/env sh
set -eu

MAX_NUM_ATTEMPTS="${MAX_NUM_ATTEMPTS:-12}"
SLEEP_DURATION_SECONDS="${SLEEP_DURATION_SECONDS:-1}"
dbhost="${1:?dbhost required}"
dbuser="${2:?dbuser required}"

num_attempts=0
until mariadb-admin --host="${dbhost}" --user="${dbuser}" --password="${MYSQL_PASSWORD:?missing MYSQL_PASSWORD}" --skip-ssl ping 2>/dev/null; do
    num_attempts=$((num_attempts+1))
    if [ $num_attempts -gt "${MAX_NUM_ATTEMPTS}" ]; then
        echo "ERROR: max attempts exceeded waiting for MySQL" >&2
        exit 1
    fi
    echo "MySQL is unavailable now, sleeping..." >&2
    sleep "${SLEEP_DURATION_SECONDS}"
done
echo "MySQL is up" >&2
