#!/usr/bin/env sh

set -eu

echo "building binary"
make sqlite3

echo "testing godfish"
make test ARGS='-v -count=1'

echo "testing godfish against live db"
make sqlite3-test ARGS='-v -count=1'
