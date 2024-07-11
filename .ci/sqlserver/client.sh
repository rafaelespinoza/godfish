#!/usr/bin/env sh

set -eu

./bin/godfish_sqlserver version

echo "testing godfish"
make test ARGS='-v -count=1 -coverprofile=/tmp/cover.out'

# Wait for db server to be ready, with some limits.

num_attempts=0

until /client_check_db ; do
	num_attempts=$((num_attempts+1))
	if [ $num_attempts -ge 15 ]; then
		>&2 echo "ERROR: max attempts exceeded"
		exit 1
	fi

	>&2 echo "db is unavailable now, sleeping"
	sleep 2
done
>&2 echo "db is up"

echo "testing godfish against live db"
make test-sqlserver ARGS='-v -count=1 -coverprofile=/tmp/cover_driver.out'
