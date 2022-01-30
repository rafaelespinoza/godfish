#!/usr/bin/env sh

set -eu

echo "building binary"
make build-sqlite3

echo "testing godfish"
make test ARGS='-v -count=1'

echo "testing godfish against live db"
make test-sqlite3 ARGS='-v -count=1'
