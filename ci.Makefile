BASENAME=godfish_test
CI_DIR=./.ci

#
# Build CI environment, run test suite against a live DB.
# NOTE: The client entrypoints require the other Makefile.
#
ci-cassandra3-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/cassandra/v3.yml up --build --exit-code-from client
ci-cassandra3-down:
	docker-compose -f $(CI_DIR)/cassandra/v3.yml down --rmi all --volumes
ci-cassandra4-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/cassandra/v4.yml up --build --exit-code-from client
ci-cassandra4-down:
	docker-compose -f $(CI_DIR)/cassandra/v4.yml down --rmi all --volumes

ci-postgres12-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/postgres/v12.yml up --build --exit-code-from client
ci-postgres12-down:
	docker-compose -f $(CI_DIR)/postgres/v12.yml down --rmi all --volumes
ci-postgres13-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/postgres/v13.yml up --build --exit-code-from client
ci-postgres13-down:
	docker-compose -f $(CI_DIR)/postgres/v13.yml down --rmi all --volumes

ci-mariadb-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/mysql/mariadb_v10.yml up --build --exit-code-from client
ci-mariadb-down:
	docker-compose -f $(CI_DIR)/mysql/mariadb_v10.yml down --rmi all --volumes
ci-mysql57-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/mysql/mysql_v57.yml up --build --exit-code-from client
ci-mysql57-down:
	docker-compose -f $(CI_DIR)/mysql/mysql_v57.yml down --rmi all --volumes
ci-mysql8-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/mysql/mysql_v8.yml up --build --exit-code-from client
ci-mysql8-down:
	docker-compose -f $(CI_DIR)/mysql/mysql_v8.yml down --rmi all --volumes

ci-sqlite3-up: build-base
	BUILD_DIR=$(BUILD_DIR) docker-compose -f $(CI_DIR)/sqlite3/docker-compose.yml up --build --exit-code-from clientserver
ci-sqlite3-down:
	docker-compose -f $(CI_DIR)/sqlite3/docker-compose.yml down --rmi all --volumes

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
