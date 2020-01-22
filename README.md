# godfish

[![GoDoc](https://godoc.org/github.com/rafaelespinoza/godfish?status.svg)](https://godoc.org/github.com/rafaelespinoza/godfish)

`godfish` is a relational database migration manager. It's similar to the very
good [`dogfish`](https://github.com/dwb/dogfish), but written in golang.

It's been tested w/ golang v1.12 on linux systems.

## goals

- use SQL in the migration files, no other high-level DSLs
- interface with many RDBMSs
- as little dependencies outside of the standard library as possible
- not terrible error messages

## build

Make a CLI binary for the DB you want to use. This tool comes with a couple of
driver implementations (mysql, postgres at the moment). Build one like so:

```
make mysql
make postgres
```

This outputs a binary, named `godfish`, which would only import from libraries
of your targeted DBMS. From there you could move it to `$GOPATH/bin`, move it
to your project or whatever else you need to do.

One goal of this project is to minimize dependencies outside of the golang
standard library *yet* also support many DBMSs. As far as I know, it's more
common to have one DBMS in a project. The point is that you don't need to
install `fooDB` for your `barDB` project.  But if you do need `fooDB` and
`barDB` in the same project, build both and take care of the naming yourself.

## usage

```
godfish help
godfish -h
godfish <command> -h
```

Make your life easier by creating a configuration file.

```
godfish init
```

Configuration options are read from command line flags first. If those are not
set, then it checks the configuration file.


Database connection parameters are always read from environment variables. The
ones to set are:

```
DB_HOST=
DB_NAME=
DB_PASSWORD=
DB_PORT=
DB_USER=
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
page, at least one implementation of each interface and tests. `gofmt` or gtfo.

Comments line lengths should be limited to 80 characters wide. Try not to make
source code lines too long. More lines is fine with the exception of
declarations of exported identifiers; they should be on one line, otherwise the
generated godoc looks weird. There are also tests, those should pass.
