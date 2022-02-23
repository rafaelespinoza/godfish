BASENAME=godfish_test
CI_DIR=./.ci

#
# Build CI environment, run test suite against a live DB.
# NOTE: The client entrypoints require the other Makefile.
#
CASSANDRA_V3_FILE=$(CI_DIR)/cassandra/v3.yml
ci-cassandra3-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CASSANDRA_V3_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(CASSANDRA_V3_FILE)
ci-cassandra3-down:
	docker-compose -f $(CASSANDRA_V3_FILE) down --rmi all --volumes

CASSANDRA_V4_FILE=$(CI_DIR)/cassandra/v4.yml
ci-cassandra4-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CASSANDRA_V4_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(CASSANDRA_V4_FILE)
ci-cassandra4-down:
	docker-compose -f $(CASSANDRA_V4_FILE) down --rmi all --volumes

POSTGRES_V12_FILE=$(CI_DIR)/postgres/v12.yml
ci-postgres12-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(POSTGRES_V12_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(POSTGRES_V12_FILE)
ci-postgres12-down:
	docker-compose -f $(POSTGRES_V12_FILE) down --rmi all --volumes

POSTGRES_V13_FILE=$(CI_DIR)/postgres/v13.yml
ci-postgres13-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(POSTGRES_V13_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(POSTGRES_V13_FILE)
ci-postgres13-down:
	docker-compose -f $(POSTGRES_V13_FILE) down --rmi all --volumes

MARIA_DB_FILE=$(CI_DIR)/mysql/mariadb_v10.yml
ci-mariadb-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(MARIA_DB_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(MARIA_DB_FILE)
ci-mariadb-down:
	docker-compose -f $(MARIA_DB_FILE) down --rmi all --volumes

MYSQL_V57_FILE=$(CI_DIR)/mysql/mysql_v57.yml
ci-mysql57-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(MYSQL_V57_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(MYSQL_V57_FILE)
ci-mysql57-down:
	docker-compose -f $(MYSQL_V57_FILE) down --rmi all --volumes

MYSQL_V8_FILE=$(CI_DIR)/mysql/mysql_v8.yml
ci-mysql8-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(MYSQL_V8_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(MYSQL_V8_FILE)
ci-mysql8-down:
	docker-compose -f $(MYSQL_V8_FILE) down --rmi all --volumes

SQLITE3_FILE=$(CI_DIR)/sqlite3/docker-compose.yml
ci-sqlite3-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(SQLITE3_FILE) up --build --exit-code-from client && \
		.ci/cp_coverage_to_host.sh $(SQLITE3_FILE)
ci-sqlite3-down:
	docker-compose -f $(SQLITE3_FILE) down --rmi all --volumes

#
# Build and tag base image.
#
# Initializing the BUILD_DIR variable is written this way to address the same
# issue described at https://stackoverflow.com/q/1909188. That is, create a temp
# dir and capture the name, but only when this rule is invoked rather than every
# single time the file is parsed. This avoids creating a bunch of empty temp
# dirs, which is annoying.
#
build-base:
	$(eval BUILD_DIR=$(shell mktemp -d -p /tmp $(BASENAME)_XXXXXX))
	git clone --depth=1 file://$(PWD) $(BUILD_DIR)
	docker image build -f $(CI_DIR)/client_base.Dockerfile -t $(BASENAME)/client_base $(BUILD_DIR)

#
# More cleanup stuff.
#
rmi-base:
	docker image rmi $(shell docker image ls -aq $(BASENAME)/client_base)
