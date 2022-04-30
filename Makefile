GO ?= go
BIN_DIR=bin
PKG_IMPORT_PATH=github.com/rafaelespinoza/godfish
GOSEC ?= gosec

CORE_SRC_PKG_PATHS=$(PKG_IMPORT_PATH) $(PKG_IMPORT_PATH)/internal/...
CASSANDRA_PATH=$(PKG_IMPORT_PATH)/drivers/cassandra
SQLSERVER_PATH=$(PKG_IMPORT_PATH)/drivers/sqlserver
MYSQL_PATH=$(PKG_IMPORT_PATH)/drivers/mysql
POSTGRES_PATH=$(PKG_IMPORT_PATH)/drivers/postgres
SQLITE3_PATH=$(PKG_IMPORT_PATH)/drivers/sqlite3

# inject this metadata when building a binary.
define LDFLAGS
-X $(PKG_IMPORT_PATH)/internal/cmd.versionBranchName=$(shell git rev-parse --abbrev-ref HEAD) \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionBuildTime=$(shell date --utc +%FT%T%z) \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionCommitHash=$(shell git rev-parse --short=7 HEAD) \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionGoVersion=$(shell $(GO) version | awk '{ print $$3 }') \
-X $(PKG_IMPORT_PATH)/internal/cmd.versionTag=$(shell git describe --tag)
endef

test:
	$(GO) test $(ARGS) $(CORE_SRC_PKG_PATHS)

vet:
	$(GO) vet $(ARGS) $(CORE_SRC_PKG_PATHS) $(PKG_IMPORT_PATH)/drivers/...

clean:
	rm -rf $(BIN_DIR)

_mkdir:
	mkdir -pv $(BIN_DIR)

# Run a security scanner over the source code. This Makefile won't install the
# scanner binary for you, so check out the gosec README for instructions:
# https://github.com/securego/gosec
#
# If necessary, specify the path to the built binary with the GOSEC env var.
#
# Also note, the package paths (last positional input to gosec command) should
# be a "relative" package path. That is, starting with a dot.
gosec:
	$(GOSEC) $(ARGS) . ./internal/... ./drivers/...

#
# Cassandra
#
build-cassandra: BIN=$(BIN_DIR)/godfish_cassandra
build-cassandra: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=cassandra" \
		$(CASSANDRA_PATH)/godfish
	@echo "built cassandra to $(BIN)"
test-cassandra:
	$(GO) test $(ARGS) $(CASSANDRA_PATH)/...

#
# Postgres
#
build-postgres: BIN=$(BIN_DIR)/godfish_postgres
build-postgres: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=postgres" \
		$(POSTGRES_PATH)/godfish
	@echo "built postgres to $(BIN)"
test-postgres:
	$(GO) test $(ARGS) $(POSTGRES_PATH)/...

#
# Microsoft SQL Server
#
build-sqlserver: BIN=$(BIN_DIR)/godfish_sqlserver
build-sqlserver:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=sqlserver" \
		$(SQLSERVER_PATH)/godfish
	@echo "built sqlserver to $(BIN)"
test-sqlserver:
	$(GO) test $(ARGS) $(SQLSERVER_PATH)/...

#
# MySQL
#
build-mysql: BIN=$(BIN_DIR)/godfish_mysql
build-mysql: _mkdir
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=mysql" \
		$(MYSQL_PATH)/godfish
	@echo "built mysql to $(BIN)"
test-mysql:
	$(GO) test $(ARGS) $(MYSQL_PATH)/...

#
# SQLite3
#
build-sqlite3: BIN=$(BIN_DIR)/godfish_sqlite3
build-sqlite3: _mkdir
	CGO_ENABLED=1 $(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/cmd.versionDriver=sqlite3" \
		$(SQLITE3_PATH)/godfish
	@echo "built sqlite3 to $(BIN)"
test-sqlite3:
	$(GO) test $(ARGS) $(SQLITE3_PATH)/...
