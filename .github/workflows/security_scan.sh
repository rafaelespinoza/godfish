#!/bin/sh

set -eu

# This script sets up a GitHub Action runner environment to run gosec on this
# project. As of 2025-12, gosec does not work very well with multiple module
# projects. See https://redirect.github.com/securego/gosec/pull/1100. The same
# can be said for the github action, securego/gosec@master. As a workaround,
# download a gosec binary and run the commands manually.

wget -O - -q https://raw.githubusercontent.com/securego/gosec/master/install.sh |
	sh -s -- -b /usr/local/bin

printf 'running gosec on core library...\n'
/usr/local/bin/gosec --tests -exclude-dir drivers/ ./...

for d in cassandra mysql postgres sqlite3 sqlserver; do
	cd "./drivers/${d}"
	printf 'running gosec on driver %s...\n' "${d}"
	/usr/local/bin/gosec --tests ./...
	cd -
done
