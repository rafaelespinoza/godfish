#!/usr/bin/env sh

set -eu

datadir='/var/lib/postgresql/data'
dbname='godfish_test'
dbuser='godfish'

if ! su - postgres -c "pg_ctl init -D ${datadir}"; then
	echo "seems like cluster is already initialized at ${datadir}"
fi

if ! su - postgres -c "pg_isready"; then
	su - postgres -c "pg_ctl start -D ${datadir} -l logfile"
fi

su - postgres -c "psql -v ON_ERROR_STOP=1" <<-SQL
	CREATE USER ${dbuser};
	CREATE DATABASE ${dbname} WITH ENCODING utf8 OWNER ${dbuser};
	GRANT ALL PRIVILEGES ON DATABASE ${dbname} TO ${dbuser};
SQL
