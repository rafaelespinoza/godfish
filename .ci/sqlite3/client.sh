#!/usr/bin/env sh

set -eu

./bin/godfish_sqlite3 version

echo "testing godfish"
just test '-v -count=1 -coverprofile=/tmp/cover.out'

echo "testing godfish against live db"
just test-sqlite3 '-v -count=1 -coverprofile=/tmp/cover_driver.out'
