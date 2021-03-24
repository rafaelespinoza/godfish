GO ?= go
BIN=godfish
DB_USER ?= godfish
DB_HOST ?= localhost
TEST_DB_NAME=godfish_test
PKG_IMPORT_PATH=github.com/rafaelespinoza/godfish

# inject this metadata when building a binary.
define LDFLAGS
-X $(PKG_IMPORT_PATH)/internal/version.BranchName=$(shell git rev-parse --abbrev-ref HEAD) \
-X $(PKG_IMPORT_PATH)/internal/version.BuildTime=$(shell date --rfc-3339=seconds --utc | tr ' ' 'T') \
-X $(PKG_IMPORT_PATH)/internal/version.CommitHash=$(shell git rev-parse --short=7 HEAD) \
-X $(PKG_IMPORT_PATH)/internal/version.GoVersion=$(shell $(GO) version | awk '{ print $$3 }') \
-X $(PKG_IMPORT_PATH)/internal/version.Tag=$(shell git describe --tag)
endef

# Register database drivers to make. For every item in this array, there should
# be three separate targets elsewhere in the Makefile. Here's an example using a
# made-up DBMS:
#
#	foodb-test-setup:
#		command to create $(TEST_DB_NAME)
#	foodb-test-teardown:
#		command to drop $(TEST_DB_NAME)
#	foodb:
#		go build ...
#
# One should have a target suffix "-test-teardown", another should have the
# target suffix "-test-setup" and the last one is just named after the DBMS,
# which builds the CLI binary.
DRIVERS ?= postgres mysql

SETUPS=$(addsuffix -test-setup, $(DRIVERS))
TEARDOWNS=$(addsuffix -test-teardown, $(DRIVERS))

test: test-teardowns test-setups
	go test $(ARGS) . ./internal/stub $(addprefix ./drivers/, $(DRIVERS))
test-setups: $(SETUPS)
test-teardowns: $(TEARDOWNS)

.PHONY: $(DRIVERS) clean
clean:
	rm $(BIN)

postgres:
	go build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/version.Driver=postgres" \
		./drivers/postgres/godfish
postgres-test-teardown:
	dropdb --if-exists $(TEST_DB_NAME)
postgres-test-setup:
	createdb -E utf8 $(TEST_DB_NAME)

mysql:
	go build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/version.Driver=mysql" \
		./drivers/mysql/godfish
mysql-test-teardown:
	mysql -u $(DB_USER) -h $(DB_HOST) \
		-e "DROP DATABASE IF EXISTS ${TEST_DB_NAME}"
mysql-test-setup:
	mysql -u $(DB_USER) -h $(DB_HOST) \
		-e "CREATE DATABASE IF NOT EXISTS ${TEST_DB_NAME}"
