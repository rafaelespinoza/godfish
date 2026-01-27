#!/usr/bin/env sh

set -eu

: "${TEST_COVERAGE_BASE_DIR:?TEST_COVERAGE_BASE_DIR is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"

dbhost="${1:?missing dbhost}"

./bin/godfish_cassandra version

echo "testing godfish"
just test -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover.out"

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

echo "testing godfish upgrade path"
./.ci/upgrade_test.sh cassandra
go tool covdata textfmt -i="${GOCOVERDIR}" -o="${TEST_COVERAGE_BASE_DIR}/integration.out"

echo "testing godfish against live db"
just test-cassandra -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover_driver.out"
