GO ?= go
BIN_DIR=bin
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
	rm -rf $(BIN_DIR)

_mkdir:
	mkdir -pv $(BIN_DIR)

build-cassandra: BIN=$(BIN_DIR)/godfish_cassandra
build-cassandra: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=cassandra" \
		./drivers/cassandra/godfish
	@echo "built cassandra to $(BIN)"
test-cassandra:
	$(GO) test $(ARGS) ./drivers/cassandra/...

build-postgres: BIN=$(BIN_DIR)/godfish_postgres
build-postgres: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=postgres" \
		./drivers/postgres/godfish
	@echo "built postgres to $(BIN)"
test-postgres:
	$(GO) test $(ARGS) ./drivers/postgres/...

build-mysql: BIN=$(BIN_DIR)/godfish_mysql
build-mysql: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=mysql" \
		./drivers/mysql/godfish
	@echo "built mysql to $(BIN)"
test-mysql:
	$(GO) test $(ARGS) ./drivers/mysql/...

build-sqlite3: BIN=$(BIN_DIR)/godfish_sqlite3
build-sqlite3: _mkdir
	CGO_ENABLED=1 $(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=sqlite3" \
		./drivers/sqlite3/godfish
	@echo "built sqlite3 to $(BIN)"
test-sqlite3:
	$(GO) test $(ARGS) ./drivers/sqlite3/...
