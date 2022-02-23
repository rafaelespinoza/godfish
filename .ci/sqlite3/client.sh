#!/usr/bin/env sh

set -eu

echo "building binary"
make build-sqlite3

echo "testing godfish"
make test ARGS='-v -count=1 -coverprofile=/tmp/cover.out'

echo "testing godfish against live db"
make test-sqlite3 ARGS='-v -count=1 -coverprofile=/tmp/cover_driver.out'

echo "vetting code"
make vet-sqlite3
