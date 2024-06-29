#!/usr/bin/env -S just -f

BASENAME := "godfish_test"
CI_DIR := justfile_directory() / ".ci"
BASE_BUILD_DIR := "/tmp" / BASENAME

#
# Build CI environment, run test suite against a live DB.
# NOTE: The client entrypoints require the other Justfile.
#

# list available recipes
@default:
    {{ justfile() }} --list --unsorted

CASSANDRA_V3_FILE := CI_DIR / "cassandra" / "v3.yml"

# setup, perform integration tests for cassandra driver, server v3
ci-cassandra3-up: (_up "cassandra_v3" CASSANDRA_V3_FILE)

# cleanup integration test environment for cassandra driver, cassandra server v3
ci-cassandra3-down: (_docker_compose_down CASSANDRA_V3_FILE)

CASSANDRA_V4_FILE := CI_DIR / "cassandra" / "v4.yml"

# setup, perform integration tests for cassandra driver, cassandra server v4
ci-cassandra4-up: (_up "cassandra_v4" CASSANDRA_V4_FILE)

# cleanup integration test environment for cassandra driver, cassandra server v4
ci-cassandra4-down: (_docker_compose_down CASSANDRA_V4_FILE)

MARIA_DB_FILE := CI_DIR / "mysql" / "mariadb_v10.yml"

# setup, perform integration tests for mysql driver, mariadb server
ci-mariadb-up: (_up "mariadb" MARIA_DB_FILE)

# cleanup integration test environment for mysql driver, mariadb server
ci-mariadb-down: (_docker_compose_down MARIA_DB_FILE)

MYSQL_V57_FILE := CI_DIR / "mysql" / "mysql_v57.yml"

# setup, perform integration tests for mysql driver, mysql v5.7 server
ci-mysql57-up: (_up "mysql_v57" MYSQL_V57_FILE)

# cleanup integration test environment for mysql driver, mysql v5.7 server
ci-mysql57-down: (_docker_compose_down MYSQL_V57_FILE)

MYSQL_V8_FILE := CI_DIR / "mysql" / "mysql_v8.yml"

# setup, perform integration tests for mysql driver, mysql v8 server
ci-mysql8-up: (_up "mysql_v8" MYSQL_V8_FILE)

# cleanup integration test environment for mysql driver, mysql v8 server
ci-mysql8-down: (_docker_compose_down MYSQL_V8_FILE)

POSTGRES_V14_FILE := CI_DIR / "postgres" / "v14.yml"

# setup, perform integration tests for postgres driver, postgres v14 server
ci-postgres14-up: (_up "postgres_v14" POSTGRES_V14_FILE)

# cleanup integration test environment for postgres driver, postgres v14 server
ci-postgres14-down: (_docker_compose_down POSTGRES_V14_FILE)

POSTGRES_V15_FILE := CI_DIR / "postgres" / "v15.yml"

# setup, perform integration tests for postgres driver, postgres v15 server
ci-postgres15-up: (_up "postgres_v15" POSTGRES_V15_FILE)

# cleanup integration test environment for postgres driver, postgres v15 server
ci-postgres15-down: (_docker_compose_down POSTGRES_V15_FILE)

SQLITE3_FILE := CI_DIR / "sqlite3" / "docker-compose.yml"

# setup, perform integration tests for sqlite3 driver
ci-sqlite3-up: (_up "sqlite3" SQLITE3_FILE)

# cleanup integration test environment for sqlite3 driver
ci-sqlite3-down: (_docker_compose_down SQLITE3_FILE)

SQLSERVER_FILE := CI_DIR / "sqlserver" / "docker-compose.yml"

# setup, perform integration tests for sqlserver driver
ci-sqlserver-up: (_up "sqlserver" SQLSERVER_FILE)

# cleanup integration test environment for sqlserver driver
ci-sqlserver-down: (_docker_compose_down SQLSERVER_FILE)

_up DRIVER_BASENAME COMPOSE_FILE: (build-base DRIVER_BASENAME) (_docker_compose_up DRIVER_BASENAME COMPOSE_FILE) (_cp_coverage_to_host COMPOSE_FILE)

_docker_compose_up DRIVER_BASENAME COMPOSE_FILE:
    BUILD_DIR={{ clean(BASE_BUILD_DIR / DRIVER_BASENAME) }} docker-compose -f {{ COMPOSE_FILE }} up --build --exit-code-from client

_cp_coverage_to_host COMPOSE_FILE:
    {{ CI_DIR }}/cp_coverage_to_host.sh {{ COMPOSE_FILE }}

_docker_compose_down COMPOSE_FILE:
    docker-compose -f {{ COMPOSE_FILE }} down --rmi all --volumes

# Build and tag base image
build-base DRIVER_BASENAME:
    #!/bin/sh
    set -eu
    build_dir={{ clean(BASE_BUILD_DIR / DRIVER_BASENAME) }}
    [ -d "${build_dir}" ] && rm -rf "${build_dir}"
    mkdir -pv "${build_dir}" && chmod -v 700 "${build_dir}"
    git clone --depth=1 file://{{ justfile_directory() }} "${build_dir}"
    docker image build -f {{ CI_DIR }}/client_base.Dockerfile -t {{ BASENAME }}/client_base "${build_dir}"

# Remove base image
rmi-base:
    docker image rmi $(docker image ls -aq {{ BASENAME }}/client_base)
