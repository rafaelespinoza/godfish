#!/usr/bin/env -S just -f

BASENAME := "godfish_test"
CI_DIR := justfile_directory() / ".ci"
BASE_BUILD_DIR := "/tmp" / BASENAME

#
# Build CI environment, run test suite against a live DB.
# NOTE: The client entrypoints require the other Justfile.
#

# List available recipes
@default:
    just --list --unsorted -f 'ci.Justfile'

[private]
_CASSANDRA_V3_FILE := CI_DIR / "cassandra" / "v3.yml"

# Setup, perform integration tests for cassandra driver, server v3
[group('driver-cassandra')]
cassandra3-up: (_up "cassandra_v3" _CASSANDRA_V3_FILE)

# Cleanup integration test environment for cassandra driver, cassandra server v3
[group('driver-cassandra')]
cassandra3-down: (_docker_compose_down _CASSANDRA_V3_FILE)

[private]
_CASSANDRA_V4_FILE := CI_DIR / "cassandra" / "v4.yml"

# Setup, perform integration tests for cassandra driver, cassandra server v4
[group('driver-cassandra')]
cassandra4-up: (_up "cassandra_v4" _CASSANDRA_V4_FILE)

# Cleanup integration test environment for cassandra driver, cassandra server v4
[group('driver-cassandra')]
cassandra4-down: (_docker_compose_down _CASSANDRA_V4_FILE)

[private]
_MARIA_DB_FILE := CI_DIR / "mysql" / "mariadb_v10.yml"

# Setup, perform integration tests for mysql driver, mariadb server
[group('driver-mysql')]
mariadb-up: (_up "mariadb" _MARIA_DB_FILE)

# Cleanup integration test environment for mysql driver, mariadb server
[group('driver-mysql')]
mariadb-down: (_docker_compose_down _MARIA_DB_FILE)

[private]
_POSTGRES_V14_FILE := CI_DIR / "postgres" / "v14.yml"

# Setup, perform integration tests for postgres driver, postgres v14 server
[group('driver-postgres')]
postgres14-up: (_up "postgres_v14" _POSTGRES_V14_FILE)

# Cleanup integration test environment for postgres driver, postgres v14 server
[group('driver-postgres')]
postgres14-down: (_docker_compose_down _POSTGRES_V14_FILE)

[private]
_POSTGRES_V15_FILE := CI_DIR / "postgres" / "v15.yml"

# Setup, perform integration tests for postgres driver, postgres v15 server
[group('driver-postgres')]
postgres15-up: (_up "postgres_v15" _POSTGRES_V15_FILE)

# Cleanup integration test environment for postgres driver, postgres v15 server
[group('driver-postgres')]
postgres15-down: (_docker_compose_down _POSTGRES_V15_FILE)

[private]
_SQLITE3_FILE := CI_DIR / "sqlite3" / "docker-compose.yml"

# Setup, perform integration tests for sqlite3 driver
[group('driver-sqlite3')]
sqlite3-up: (_up "sqlite3" _SQLITE3_FILE)

# Cleanup integration test environment for sqlite3 driver
[group('driver-sqlite3')]
sqlite3-down: (_docker_compose_down _SQLITE3_FILE)

[private]
_SQLSERVER_FILE := CI_DIR / "sqlserver" / "docker-compose.yml"

# Cetup, perform integration tests for sqlserver driver
[group('driver-sqlserver')]
sqlserver-up: (_up "sqlserver" _SQLSERVER_FILE)

# Cleanup integration test environment for sqlserver driver
[group('driver-sqlserver')]
sqlserver-down: (_docker_compose_down _SQLSERVER_FILE)

_up driver_basename compose_file: (make-builder-img driver_basename) (_docker_compose_up compose_file) (_cp_coverage_to_host compose_file)

_docker_compose_up compose_file:
    docker compose -f {{ compose_file }} up --build --exit-code-from client

_cp_coverage_to_host compose_file:
    {{ CI_DIR }}/cp_coverage_to_host.sh {{ compose_file }}

_docker_compose_down compose_file:
    docker compose -f {{ compose_file }} down --rmi all --volumes

# Build and tag builder image
make-builder-img driver_basename:
    #!/bin/sh
    set -eu
    build_dir={{ clean(BASE_BUILD_DIR / driver_basename) }}
    [ -d "${build_dir}" ] && rm -rf "${build_dir}"
    mkdir -pv "${build_dir}" && chmod -v 700 "${build_dir}"
    git clone --depth=1 'file://{{ justfile_directory() }}' "${build_dir}"
    docker image build -f {{ CI_DIR }}/client_base.Dockerfile -t {{ BASENAME }}/client_base "${build_dir}"

# Remove builder image
rm-builder-img:
    docker image rmi $(docker image ls -aq {{ BASENAME }}/client_base)
