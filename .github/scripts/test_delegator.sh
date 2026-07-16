#!/usr/bin/env bats

# Unit tests for the delegator command.
# They use the bats library: https://bats-core.readthedocs.io/.

set -eu -o pipefail

bats_load_library bats-assert
bats_load_library bats-file
bats_load_library bats-support

function setup_file() {
	export BIN=bin/godfish
	readonly BIN
}

@test 'outputs a Usage message' {
	assert_file_executable "${BIN}"

	run "${BIN}"
	assert_success
	assert_output --partial 'USAGE:'

	run "${BIN}" -h
	assert_success
	assert_output --partial 'USAGE:'

	run "${BIN}" --help
	assert_success
	assert_output --partial 'USAGE:'

	run "${BIN}" help
	assert_success
	assert_output --partial 'USAGE:'
}

@test 'error for invalid driver name' {
	assert_file_executable "${BIN}"

	run "${BIN}" not_a_driver
	assert_failure
	assert_output --partial 'not found'
}

@test 'routes commands to correct driver' {
	assert_file_executable "${BIN}"

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		run "${BIN}" "${driver}" version
		assert_output --regexp "Driver.*${driver}"
	done
}
