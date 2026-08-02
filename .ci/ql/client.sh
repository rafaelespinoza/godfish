#!/usr/bin/env sh

set -eu

: "${TEST_COVERAGE_BASE_DIR:?TEST_COVERAGE_BASE_DIR is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"

./bin/godfish-ql version

echo "testing godfish"
just test -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover.out"

echo "testing godfish CLI"
# Unlike tests for other DB drivers, we are not testing the "upgrade" path here
# because this DB driver was introduced > v0.14.0.
DB_DRIVER=ql bats --abort --pretty --print-output-on-failure \
	./.ci/test_config.sh
go tool covdata textfmt -i="${GOCOVERDIR}" -o="${TEST_COVERAGE_BASE_DIR}/integration.out"

echo "testing godfish against live db"
just test-ql -v -count=1 -coverprofile="${TEST_COVERAGE_BASE_DIR}/cover_driver.out"
