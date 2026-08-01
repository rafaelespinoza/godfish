package driver

import "errors"

// ErrSchemaMigrationsDoesNotExist means there is no database table to
// record migration status.
var ErrSchemaMigrationsDoesNotExist = errors.New("schema migrations table does not exist")

// ErrSchemaMigrationsMissingColumns means the schema migrations table exists,
// but is missing some extra metadata columns.
var ErrSchemaMigrationsMissingColumns = errors.New("schema migrations table is missing columns")
