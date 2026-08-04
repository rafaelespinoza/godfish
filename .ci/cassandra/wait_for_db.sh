#!/usr/bin/env sh
set -eu

MAX_NUM_ATTEMPTS="${MAX_NUM_ATTEMPTS:-12}"
SLEEP_DURATION_SECONDS="${SLEEP_DURATION_SECONDS:-5}"
dbhost="${1:?dbhost required}"
keyspace="${2:?keyspace required}"

num_attempts=0
until /client_setup_keyspace "${dbhost}" "${keyspace}" 2>/dev/null; do
    num_attempts=$((num_attempts+1))
    if [ $num_attempts -gt "${MAX_NUM_ATTEMPTS}" ]; then
        echo "ERROR: max attempts exceeded waiting for Cassandra" >&2
        exit 1
    fi
    echo "Cassandra is unavailable now, sleeping..." >&2
    sleep "${SLEEP_DURATION_SECONDS}"
done
echo "Cassandra is up" >&2
