#!/usr/bin/env sh
set -eu

MAX_NUM_ATTEMPTS="${MAX_NUM_ATTEMPTS:-15}"
SLEEP_DURATION_SECONDS="${SLEEP_DURATION_SECONDS:-2}"

num_attempts=0
until /client_check_db 2>/dev/null; do
    num_attempts=$((num_attempts+1))
    if [ $num_attempts -ge "${MAX_NUM_ATTEMPTS}" ]; then
        echo "ERROR: max attempts exceeded waiting for SQL Server" >&2
        exit 1
    fi
    echo "SQL Server is unavailable now, sleeping..." >&2
    sleep "${SLEEP_DURATION_SECONDS}"
done
echo "SQL Server is up" >&2
