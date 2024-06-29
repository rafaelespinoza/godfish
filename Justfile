#!/usr/bin/env -S just -f

GO := "go"
BIN_DIR := justfile_directory() / "bin"
PKG_IMPORT_PATH := "github.com/rafaelespinoza/godfish"
_CORE_SRC_PKG_PATHS := PKG_IMPORT_PATH + " " + PKG_IMPORT_PATH / "internal" / "..."
_BASE_DRIVER_PATH := PKG_IMPORT_PATH / "drivers"
_LDFLAGS_BASE_PREFIX := "-X " + PKG_IMPORT_PATH + "/internal/cmd"
_LDFLAGS_DELIMITER := "\n\t"
LDFLAGS := (
  _LDFLAGS_BASE_PREFIX + ".versionBranchName=" + `git rev-parse --abbrev-ref HEAD` + _LDFLAGS_DELIMITER +
  _LDFLAGS_BASE_PREFIX + ".versionBuildTime=" + `date --utc +%FT%T%z` + _LDFLAGS_DELIMITER +
  _LDFLAGS_BASE_PREFIX + ".versionCommitHash=" + `git rev-parse --short=7 HEAD` + _LDFLAGS_DELIMITER +
  _LDFLAGS_BASE_PREFIX + ".versionGoVersion=" + `go version | awk '{ print $3 }'` + _LDFLAGS_DELIMITER +
  _LDFLAGS_BASE_PREFIX + ".versionTag=" + `git describe --tag 2>/dev/null || echo 'dev'` + _LDFLAGS_DELIMITER
)

# list available recipes
@default:
    {{ justfile() }} --list --unsorted

# run unit tests on core source packages
test ARGS='':
    {{ GO }} test {{ ARGS }} {{ _CORE_SRC_PKG_PATHS }}

# examine source code for suspicious constructs
vet ARGS='':
    {{ GO }} vet {{ ARGS }} {{ _CORE_SRC_PKG_PATHS }} {{ PKG_IMPORT_PATH }}/drivers/...

# Remove BIN_DIR
clean:
    rm -rf {{ BIN_DIR }}

GOSEC := "gosec"

# This Justfile won't install the scanner binary for you, so check out the
# gosec README for instructions: https://github.com/securego/gosec
#
# If necessary, specify the path to the built binary with the GOSEC variable.
#
# Also note, the package paths (last positional input to gosec command) should
# be a "relative" package path. That is, starting with a dot.
#
# Run a security scanner over the source code
gosec ARGS='':
    {{ GOSEC }} {{ ARGS }} . ./internal/... ./drivers/...

_CASSANDRA_PATH := _BASE_DRIVER_PATH / "cassandra"

# Compile binary for cassandra driver
build-cassandra: (_build_driver "cassandra" (_CASSANDRA_PATH / "godfish"))

# Run unit tests on a live cassandra instance at DB_DSN
test-cassandra ARGS='':
    {{ GO }} test {{ ARGS }} {{ _CASSANDRA_PATH }}/...

_MYSQL_PATH := _BASE_DRIVER_PATH / "mysql"

# Compile binary for mysql driver
build-mysql: (_build_driver "mysql" (_MYSQL_PATH / "godfish"))

# Run unit tests on a live mysql instance at DB_DSN
test-mysql ARGS='':
    {{ GO }} test {{ ARGS }} {{ _MYSQL_PATH }}/...

_POSTGRES_PATH := _BASE_DRIVER_PATH / "postgres"

# Compile binary for postgres driver
build-postgres: (_build_driver "postgres" (_POSTGRES_PATH / "godfish"))

# Run unit tests on a live postgres instance at DB_DSN
test-postgres ARGS='':
    {{ GO }} test {{ ARGS }} {{ _POSTGRES_PATH }}/...

_SQLSERVER_PATH := _BASE_DRIVER_PATH / "sqlserver"

# Compile binary for sqlserver driver
build-sqlserver: (_build_driver "sqlserver" (_SQLSERVER_PATH / "godfish"))

# Run unit tests on a live sqlserver instance at DB_DSN
test-sqlserver ARGS='':
    {{ GO }} test {{ ARGS }} {{ _SQLSERVER_PATH }}/...

_SQLITE3_PATH := _BASE_DRIVER_PATH / "sqlite3"

# Compile binary for sqlite3 driver
build-sqlite3: (_build_driver "sqlite3" (_SQLITE3_PATH / "godfish"))

# Run unit tests on a live sqlite3 instance at DB_DSN
test-sqlite3 ARGS='':
    {{ GO }} test {{ ARGS }} {{ _SQLITE3_PATH }}/...

_build_driver DRIVER_NAME SRC_PATH:
    #!/bin/sh
    set -eu
    bin={{ clean(BIN_DIR / "godfish_" + DRIVER_NAME) }}
    mkdir -pv {{ BIN_DIR }}
    ldflags="{{ LDFLAGS }}{{ _LDFLAGS_BASE_PREFIX }}.versionDriver={{ DRIVER_NAME }}"
    {{ GO }} build -o="${bin}" -v -ldflags="${ldflags}" {{ SRC_PATH }}
    "${bin}" version
    echo "built {{ DRIVER_NAME }} to ${bin}"
