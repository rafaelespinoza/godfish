#!/usr/bin/env sh

set -eu

: "${TEST_COVERAGE_BASE_DIR:?TEST_COVERAGE_BASE_DIR is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"

./bin/godfish_sqlite3 version

echo "testing godfish"
just test -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover.out"

echo "testing godfish upgrade path"
DB_DRIVER=sqlite3 bats ./.ci/test_upgrade.sh
go tool covdata textfmt -i="${GOCOVERDIR}" -o="${TEST_COVERAGE_BASE_DIR}/integration.out"

echo "testing godfish against live db"
just test-sqlite3 -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover_driver.out"
