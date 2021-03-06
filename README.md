# godfish

[![Go Reference](https://pkg.go.dev/badge/github.com/rafaelespinoza/godfish.svg)](https://pkg.go.dev/github.com/rafaelespinoza/godfish)

[![cassandra](https://github.com/rafaelespinoza/godfish/actions/workflows/cassandra.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/cassandra.yml)
[![mysql](https://github.com/rafaelespinoza/godfish/actions/workflows/mysql.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/mysql.yml)
[![postgres](https://github.com/rafaelespinoza/godfish/actions/workflows/postgres.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/postgres.yml)
[![sqlite3](https://github.com/rafaelespinoza/godfish/actions/workflows/sqlite3.yml/badge.svg)](https://github.com/rafaelespinoza/godfish/actions/workflows/sqlite3.yml)

`godfish` is a database migration manager, similar to the very good
[`dogfish`](https://github.com/dwb/dogfish), but written in golang.

## goals

- use the native query language in the migration files, no other high-level DSLs
- interface with many DBs
- light on dependencies
- not terrible error messages

## build

Make a CLI binary for the DB you want to use. This tool comes with some driver
implementations. Build one like so:

```
make cassandra
make postgres
make mysql
make sqlite3
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

## tests

Docker and docker-compose are used to create environments and run the tests
against a live database. Each database has a separate configuration. All of this
lives in `Makefile.docker` and the `.ci/` directory.

```sh
# Build environments and run tests
make -f Makefile.docker ci-cassandra-up
make -f Makefile.docker ci-mysql-up
make -f Makefile.docker ci-postgres-up
make -f Makefile.docker ci-sqlite3-up

# Teardown
make -f Makefile.docker ci-cassandra-down
make -f Makefile.docker ci-mysql-down
make -f Makefile.docker ci-postgres-down
make -f Makefile.docker ci-sqlite3-down
```
