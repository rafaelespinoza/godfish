# godfish

[![Go Reference](https://pkg.go.dev/badge/github.com/rafaelespinoza/godfish.svg)](https://pkg.go.dev/github.com/rafaelespinoza/godfish)
[![codecov](https://codecov.io/gh/rafaelespinoza/godfish/branch/main/graph/badge.svg?token=EoLelW4qiy)](https://codecov.io/gh/rafaelespinoza/godfish)
[![Go Report Card](https://goreportcard.com/badge/github.com/rafaelespinoza/godfish)](https://goreportcard.com/report/github.com/rafaelespinoza/godfish)

[![cassandra](https://github.com/rafaelespinoza/godfish/actions/workflows/build-cassandra.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/build-cassandra.yml)
[![mysql](https://github.com/rafaelespinoza/godfish/actions/workflows/build-mysql.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/build-mysql.yml)
[![postgres](https://github.com/rafaelespinoza/godfish/actions/workflows/build-postgres.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/build-postgres.yml)
[![sqlite3](https://github.com/rafaelespinoza/godfish/actions/workflows/build-sqlite3.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/build-sqlite3.yml)
[![sqlserver](https://github.com/rafaelespinoza/godfish/actions/workflows/build-sqlserver.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/sqlserver.yml)

`godfish` is a database migration manager, similar to the very good
[`dogfish`](https://github.com/dwb/dogfish), but written in golang.

## goals

- use the native query language in the migration files, no other high-level DSLs
- interface with many DBs
- light on dependencies
- not terrible error messages

## build

NOTE: these require [just](https://just.systems).

Make a CLI binary for the DB you want to use. This tool comes with some driver
implementations. Build one like so:

```
just build-cassandra
just build-mysql
just build-postgres
just build-sqlite3
just build-sqlserver
```

From there you could move it to `$GOPATH/bin`, move it to your project or
whatever else you need to do.

## usage

```
godfish help
godfish -h
godfish <command> -h
```

Configuration options are read from command line flags first. If those are not
set, then it checks the configuration file.

#### connecting to the db

Database connection parameters are always read from environment variables. Set:
```
DB_DSN=
```

#### configure file paths

Manually set path to db migration files.

```sh
godfish -files db/migrations <command>
```

Make your life easier by creating a configuration file by invoking `godfish
init`. This creates a file at `.godfish.json`, where you can configure things.

Change the path to the configuration file.

```sh
mv .godfish.json foo.json
godfish -conf foo.json
```

#### everything else

```sh
cat .godfish.json
# { "path_to_files": "db/migrations" }

godfish create-migration -name alpha
# outputs:
# db/migrations/forward-20200128070010-alpha.sql
# db/migrations/reverse-20200128070010-alpha.sql

godfish create-migration -name bravo -reversible=false
# outputs:
# db/migrations/forward-20200128070106-bravo.sql

#
# ... write the sql in those files ...
#

# apply migrations
godfish migrate
# apply migrations to up a specific version
godfish migrate -version 20060102150405

# show status
godfish info

# apply a reverse migration
godfish rollback

# rollback and re-apply the last migration
godfish remigrate

# show build metadata
godfish version
godfish version -json
```

## other minutiae

Here are some notable differences between `dogfish` and `godfish`:

Filenames:

- dogfish: `migrate-${date}-${name}.sql`, or `rollback-${date}-${name}.sql`
- godfish: `forward-${date}-${name}.sql`, or `reverse-${date}-${name}.sql`

Note, dogfish uses the words, "migrate" and "rollback" to describe the
migration's direction whereas godfish uses "forward" and "reverse". They are
the same in that they are two complementaries. This change has one trivial
benefit, the pieces of metadata encoded into the filename naturally align:

```
cd /path/to/db/migrations && ls -1

forward-20191112050547-init_foos.sql
forward-20191127051242-add_bars.sql
forward-20191205031405-update_more_stuff.sql
reverse-20191112050547-init_foos.sql
reverse-20191127051242-add_bars.sql
reverse-20191205031405-update_more_stuff.sql
```

## contributing

These are welcome. To get you started, the code has some documentation, a godoc
page, at least one implementation of each interface and tests.

Comments line lengths should be limited to 80 characters wide. Try not to make
source code lines too long. More lines is fine with the exception of
declarations of exported identifiers; they should be on one line, otherwise the
generated godoc looks weird. There are also tests, those should pass.

The GitHub Actions run a security scanner on all of the source code using
[gosec](https://github.com/securego/gosec). There should be no rule violations
here. The Justfile provides a convenience target if you want to run `gosec` on
your development machine.

## tests

Docker and docker-compose are used to create environments and run the tests
against a live database. Each database has a separate configuration. All of this
lives in `ci.Justfile` and the `.ci/` directory.

Build environments and run tests
```sh
just -f ci.Justfile ci-cassandra3-up
just -f ci.Justfile ci-cassandra4-up

just -f ci.Justfile ci-sqlserver-up

just -f ci.Justfile ci-mariadb-up
just -f ci.Justfile ci-mysql57-up
just -f ci.Justfile ci-mysql8-up

just -f ci.Justfile ci-postgres14-up
just -f ci.Justfile ci-postgres15-up

just -f ci.Justfile ci-sqlite3-up
```

Teardown
```sh
just -f ci.Justfile ci-cassandra3-down
just -f ci.Justfile ci-cassandra4-down

just -f ci.Justfile ci-sqlserver-down

just -f ci.Justfile ci-mariadb-down
just -f ci.Justfile ci-mysql57-down
just -f ci.Justfile ci-mysql8-down

just -f ci.Justfile ci-postgres14-down
just -f ci.Justfile ci-postgres15-down

just -f ci.Justfile ci-sqlite3-down
```
