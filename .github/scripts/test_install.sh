#!/usr/bin/env bats

# Unit tests for the installation script.
# They use the bats library: https://bats-core.readthedocs.io/.

set -eu -o pipefail

: "${BASE_INSTALL_DIR:?env var BASE_INSTALL_DIR is required}"
bats_load_library bats-assert
bats_load_library bats-file
bats_load_library bats-support

function _test_one_driver() {
	local -r driver="${1:?missing driver}"
	local -r install_dir="${BASE_INSTALL_DIR}"

	mkdir -pv "${install_dir}"

	local -r binary="${install_dir}/godfish_${driver}"
	local -r version_file="${install_dir}/version_${driver}"

	run ./scripts/install.sh -d "${driver}" -o "$(realpath "${install_dir}")"
	assert_file_executable "${binary}"

	run bash -c "${binary} version > ${version_file}"
	assert_file_contains "${version_file}" "^Driver.*${driver}$"
}

@test 'outputs a Usage message with no args' {
	run ./scripts/install.sh
	assert_output --partial 'Usage:'
}

@test 'outputs a Usage message with arg, -h' {
	run ./scripts/install.sh -h
	assert_output --partial 'Usage:'
}

@test 'installs cassandra driver' {
	_test_one_driver cassandra
}

@test 'installs mysql driver' {
	_test_one_driver mysql
}

@test 'installs postgres driver' {
	_test_one_driver postgres
}

@test 'installs sqlite3 driver' {
	_test_one_driver sqlite3
}

@test 'installs sqlserver driver' {
	_test_one_driver sqlserver
}

@test 'installs all drivers' {
	local -r install_dir="${BASE_INSTALL_DIR}/all"
	mkdir -pv "${install_dir}"

	run ./scripts/install.sh -o "$(realpath "${install_dir}")"
	local binary version_file

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		binary="${install_dir}/godfish_${driver}"
		version_file="${install_dir}/version_${driver}"

		assert_file_executable "${binary}"

		run bash -c "${binary} version > ${version_file}"
		assert_file_contains "${version_file}" "^Driver.*${driver}$"
	done
}

@test 'installs a tagged version' {
	local -r install_dir="${BASE_INSTALL_DIR}/tagged"
	mkdir -pv "${install_dir}"
	local -r tag=v0.13.0

	run ./scripts/install.sh -o "$(realpath "${install_dir}")" -t "${tag}"
	local binary version_file

	for driver in cassandra mysql postgres sqlite3 sqlserver; do
		binary="${install_dir}/godfish_${driver}"
		version_file="${install_dir}/version_${driver}"

		assert_file_executable "${binary}"

		run bash -c "${binary} version > ${version_file}"
		assert_file_contains "${version_file}" "^Driver.*${driver}$"
		assert_file_contains "${version_file}" "^Tag.*${tag}$"
	done
}
