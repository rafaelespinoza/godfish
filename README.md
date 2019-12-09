# godfish

`godfish` is a relational database migration manager. It's similar to the very
good [`dogfish`](https://github.com/dwb/dogfish), but written in golang.

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

## installation

```
go install bitbucket.org/rafaelespinoza/godfish
```

or for development

```
go get bitbucket.org/rafaelespinoza/godfish
cd "$GOPATH/bitbucket.org/rafaelespinoza/godfish"
make install
```

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
