#!/usr/bin/env sh

set -eu

: "${TEST_COVERAGE_BASE_DIR:?TEST_COVERAGE_BASE_DIR is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"

./bin/godfish-sqlite3 version

echo "testing godfish"
just test -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover.out"

echo "testing godfish CLI"
# run the upgrade test first.
DB_DRIVER=sqlite3 bats --abort --pretty --print-output-on-failure \
	./.ci/test_upgrade.sh \
	./.ci/test_config.sh
go tool covdata textfmt -i="${GOCOVERDIR}" -o="${TEST_COVERAGE_BASE_DIR}/integration.out"

echo "testing godfish against live db"
just test-sqlite3 -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover_driver.out"
