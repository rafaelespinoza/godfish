GO ?= go
BIN=godfish
DB_USER ?= godfish
DB_HOST ?= localhost
TEST_DB_NAME=godfish_test
PKG_IMPORT_PATH=github.com/rafaelespinoza/godfish

# inject this metadata when building a binary.
define LDFLAGS
-X $(PKG_IMPORT_PATH)/internal/cmd.versionBranchName=$(shell git rev-parse --abbrev-ref HEAD) \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionBuildTime=$(shell date --rfc-3339=seconds --utc | tr ' ' 'T') \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionCommitHash=$(shell git rev-parse --short=7 HEAD) \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionGoVersion=$(shell $(GO) version | awk '{ print $$3 }') \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionTag=$(shell git describe --tag)
endef

test:
	$(GO) test $(ARGS) . ./internal/...

clean:
	rm $(BIN)

cassandra:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=cassandra" \
		./drivers/cassandra/godfish
cassandra-test:
	$(GO) test $(ARGS) ./drivers/cassandra

postgres:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=postgres" \
		./drivers/postgres/godfish
postgres-test:
	$(GO) test $(ARGS) ./drivers/postgres

mysql:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=mysql" \
		./drivers/mysql/godfish
mysql-test:
	$(GO) test $(ARGS) ./drivers/mysql

sqlite3:
	CGO_ENABLED=1 $(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=sqlite3" \
		./drivers/sqlite3/godfish
sqlite3-test:
	$(GO) test $(ARGS) ./drivers/sqlite3
