package testdata

import "embed"

// Migrations is embedded migrations data for tests.
//
//go:embed cassandra default sqlserver
var Migrations embed.FS
