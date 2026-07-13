#!/usr/bin/env bats

# Unit tests for the delegator script.
# They use the bats library: https://bats-core.readthedocs.io/.

set -eu -o pipefail

: "${BASE_TEST_DIR:?BASE_TEST_DIR is required}"

bats_load_library bats-assert
bats_load_library bats-file
bats_load_library bats-support

function setup_file() {
	export DRIVER_BIN_BASENAME=godfish-
	readonly DRIVER_BIN_BASENAME

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		assert_file_executable "bin/${DRIVER_BIN_BASENAME}${driver}"
	done
}

function _test_delegator_script() {
	local -r delegator_script="${1:?missing delegator_script}"
	local -r driver="${2:?missing driver}"

	assert_file_executable "${delegator_script}"

	run "${delegator_script}" "${driver}" version
	assert_output --regexp "Driver.*${driver}"
}

@test 'outputs a Usage message' {
	local -r delegator_script=./scripts/godfish

	assert_file_executable "${delegator_script}"

	run "${delegator_script}"
	assert_success
	assert_output --partial 'Usage:'

	run "${delegator_script}" -h
	assert_success
	assert_output --partial 'Usage:'

	run "${delegator_script}" --help
	assert_success
	assert_output --partial 'Usage:'

	run "${delegator_script}" help
	assert_success
	assert_output --partial 'Usage:'
}

@test 'error for invalid driver name' {
	local -r delegator_script=./scripts/godfish

	assert_file_executable "${delegator_script}"

	run "${delegator_script}" not_a_driver
	assert_failure
	assert_output --partial 'Usage:'
}

@test 'driver binaries live in same directory as script' {
	local -r test_dir="${BASE_TEST_DIR}/${BATS_TEST_NAME}"

	mkdir -pv "${test_dir}/bin"

	cp -v scripts/godfish "${test_dir}/bin"

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		cp -v "bin/${DRIVER_BIN_BASENAME:?}${driver}" "${test_dir}/bin"
		_test_delegator_script "${test_dir}/bin/godfish" "${driver}"
	done

	rm -rf "${test_dir}"
}

@test 'driver binaries are placed in a libexec-like arrangement' {
	local -r test_dir="${BASE_TEST_DIR}/${BATS_TEST_NAME}"

	mkdir -pv "${test_dir}/bin" "${test_dir}/libexec"

	cp -v scripts/godfish "${test_dir}/bin"

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		cp -v "bin/${DRIVER_BIN_BASENAME:?}${driver}" "${test_dir}/libexec"
		_test_delegator_script "${test_dir}/bin/godfish" "${driver}"
	done

	rm -rf "${test_dir}"
}

@test 'driver binaries are placed in user PATH' {
	local -r test_dir="${BASE_TEST_DIR}/${BATS_TEST_NAME}"

	mkdir -pv "${test_dir}/bin" "${test_dir}/TEST_PATH"
	export PATH="${test_dir}/TEST_PATH:${PATH}"
	cp -v scripts/godfish "${test_dir}/bin"

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		cp -v "bin/${DRIVER_BIN_BASENAME:?}${driver}" "${test_dir}/TEST_PATH"
		_test_delegator_script "${test_dir}/bin/godfish" "${driver}"
	done

	rm -rf "${test_dir}"
}
