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

test:
	$(GO) test $(ARGS) . ./internal/...

clean:
	rm $(BIN)

postgres:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/version.Driver=postgres" \
		./drivers/postgres/godfish
postgres-test:
	$(GO) test $(ARGS) ./drivers/postgres

mysql:
	$(GO) build -o $(BIN) -v \
		-ldflags "$(LDFLAGS) \
		-X $(PKG_IMPORT_PATH)/internal/version.Driver=mysql" \
		./drivers/mysql/godfish
mysql-test:
	$(GO) test $(ARGS) ./drivers/mysql
