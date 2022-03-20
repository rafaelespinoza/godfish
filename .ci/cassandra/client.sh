#!/usr/bin/env sh

set -eu

dbhost="${1:?missing dbhost}"

echo "building binary"
make build-cassandra
echo "testing godfish"
make test ARGS='-v -count=1 -coverprofile=/tmp/cover.out'

# Wait for db server to be ready, with some limits.
num_attempts=0

until /client_setup_keyspace "${dbhost}" godfish_test ; do
	num_attempts=$((num_attempts+1))
	if [ $num_attempts -gt 12 ]; then
		>&2 echo "ERROR: max attempts exceeded"
		exit 1
	fi

	>&2 echo "db is unavailable now, sleeping"
	sleep 5
done
>&2 echo "db is up"

echo "testing godfish against live db"
make test-cassandra ARGS='-v -count=1 -coverprofile=/tmp/cover_driver.out'
