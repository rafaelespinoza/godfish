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
It is a CLI and a library.

## goals

- use the native query language in the migration files, no other high-level DSLs
- interface with many DBs
- light on dependencies
- not terrible error messages

## releases

The Releases page of the GitHub repository has pre-built artifacts for supported platforms.
Each archive file contains an executable binary per driver. Each executable binary will only work
for the targeted DB. Pick the one(s) you need.

There is also an installation script at [scripts/install.sh](./scripts/install.sh). Check it out.

## build

An alternative to using a pre-built release to is to build your own.
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

## concepts

Like any db migration tool, this helps you and your team manage the shape of
your application's database throughout time. Migrations are written in the
native language of the database, and exist as files in a directory. All
migrations to consider should live in the same directory. A migration is one of
these files, which may have metadata components as part of the filename:

* "direction": Migrations that introduce new changes to the DB shape are
  considered to have a "forward" direction. A migration intended as the
  inverse of a corresponding forward migration is considered to have a
  "reverse" direction.
* "version": Describe where a migration exists relative to the other migrations
  with the same direction. By default, it's a timestamp in the layout
  `YYYYMMDDHHmmss`.
* "name": A label you give to describe the migration's contents.

The delimiter of each part is a `-`. Each migration filename has this format:
```
${direction}-${version}-${name}.sql
```

There is an implicit ordering to migrations, denoted by the "version", where
subsequent migrations may build upon the shape created by preceding migrations.
When a migration is successfully applied, a new row is inserted into the schema
migrations table. By default its name is `schema_migrations`. A migration
that has not yet been applied will be a file in the directory, but without a
corresponding entry in the DB table.

The shape of the schema migrations table is roughly:

| column         | type    | description                                          |
|----------------|---------|------------------------------------------------------|
| `migration_id` | varchar | primary key timestamp, derived from filename version |
| `label`        | varchar | describes the migration, also derived from filename  |
| `executed_at`  | integer | unix epoch of when migration was applied             |

## usage

Not only is this tool a CLI, it's also a database migration library. Most of the
time, you'll probably want to use it as a CLI.

### command line usage

This section describes basic usage of a CLI binary. For details on getting a CLI
binary, see the [releases](#releases) section. Golang is not required here.

```
godfish help
godfish -h
godfish <command> -h
```

Configuration options are read from command line flags first. If those are not
set, then it checks the configuration file.

#### connecting to the db

Database connection parameters may be read from an environment variable. Set:
```
DB_DSN=
```
When using one of the pre-built binaries, the database may also be specified
with a command line flag, `-dsn`. If specified in both places, then the command
line value will have higher precedence than the environment variable.

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

### library usage

Though most of the time you'll probably want to use one of the pre-built
binaries, you could also use this as a golang library.

```
go get github.com/rafaelespinoza/godfish
```

#### embed migrations

An issue that may arise with deployments is that the migration files must be
deployed alongside the godfish binary. Migrations data and the behavior provided
by the `godfish` library can be combined into a single self-contained binary by
using the [`embed`](https://pkg.go.dev/embed) package.
See the [go doc](https://pkg.go.dev/github.com/rafaelespinoza/godfish?tab=doc)
page for an example.

### upgrading schema migrations

If you have data created with `v0.14.0` or lower and then later on use a newer
version, then you may run into an error message like:
```
schema migrations table is missing columns; run the upgrade command to fix this
```

A schema migrations table created with versions <= `v0.14.0` will lack the
`label` and `executed_at` columns. The `upgrade` command adds them.

## other minutiae

Here are some notable differences between `dogfish` and `godfish`:

Filenames:

- dogfish: `migrate-${date}-${name}.sql`, or `rollback-${date}-${name}.sql`
- godfish: `forward-${date}-${name}.sql`, or `reverse-${date}-${name}.sql`

Note, dogfish uses the words, "migrate" and "rollback" to describe the
migration's direction whereas godfish uses "forward" and "reverse". They are
the same in that they are two complementaries.

```sh
ls -1 /path/to/db/migrations

forward-20191112050547-init_foos.sql
forward-20221067051242-add_bars.sql
forward-20250805031405-update_more_stuff.sql
reverse-20191112050547-init_foos.sql
reverse-20221067051242-add_bars.sql
reverse-20250805031405-update_more_stuff.sql
```

## project organization

One of the goals of this project is to minimize the amount of dependencies.
Driver-specific code is placed in `drivers/` so that building a binary for one
database does not require building code for another database.

The `godfish` package defines library functions, interfaces needed to build a
driver implementation.

Test infrastructure mostly lives in the `.ci/` and `.github/` directories. Many
integration tests may be run in isolation on your local machine without GitHub
actions.

## tests

Docker (or equivalent) is used to create environments and run the tests against
a live database. Each database has a separate configuration. All of this lives
in `ci.Justfile` and the `.ci/` directory.

Using an OCI-compatible tool other than `docker` (ie: `podman`)?
```sh
just --set CONTAINER_TOOL podman -f ci.Justfile
```

Build environments and run tests
```sh
just -f ci.Justfile cassandra3-up
just -f ci.Justfile cassandra4-up

just -f ci.Justfile sqlserver-up

just -f ci.Justfile mariadb-up

just -f ci.Justfile postgres14-up
just -f ci.Justfile postgres15-up

just -f ci.Justfile sqlite3-up
```

Teardown
```sh
just -f ci.Justfile cassandra3-down
just -f ci.Justfile cassandra4-down

just -f ci.Justfile sqlserver-down

just -f ci.Justfile mariadb-down

just -f ci.Justfile postgres14-down
just -f ci.Justfile postgres15-down

just -f ci.Justfile sqlite3-down
```
