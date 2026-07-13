#!/usr/bin/env bats

# These tests use a library, bats.
# See https://bats-core.readthedocs.io.

set -eu -o pipefail

: "${DB_DSN:?DB_DSN is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"
: "${DB_DRIVER:?DB_DRIVER is required}"

bats_load_library bats-assert
bats_load_library bats-file
bats_load_library bats-support

function setup_file() {
	mkdir -pv "${GOCOVERDIR}"

	# Most variables set here are meant to be in the global scope so they can be
	# read from tests and other functions. Those variables must be exported.

	root_src_dir=$(git rev-parse --show-toplevel)
	work_base_dir="$(mktemp -d -p /tmp godfish_XXXXXX)"
	local -r work_bin_dir="${work_base_dir}/bin"
	prev_version='v0.14.0'
	prev_bin="${work_bin_dir}/godfish_${DB_DRIVER}"
	# next_bin is the binary to test for upgrades. It can be created with the
	# Justfile. $ just build-${DB_DRIVER}-test
	next_bin="${root_src_dir}/bin/godfish-${DB_DRIVER}_test"
	local -r testdata_base_dir="${root_src_dir}/testdata"

	case "${DB_DRIVER}" in
	cassandra | sqlserver)
		testdata_dir="${testdata_base_dir}/${DB_DRIVER}"
		;;
	*)
		testdata_dir="${testdata_base_dir}/default"
		;;
	esac

	readonly root_src_dir work_base_dir prev_version prev_bin next_bin testdata_dir
	export root_src_dir work_base_dir prev_version prev_bin next_bin testdata_dir
}

function teardown_file() {
	[[ -d "${work_base_dir}" ]] && (rm -rf -- "${work_base_dir}" || true)

	# reset state for next tests.
	"${next_bin}" -files "${testdata_dir}" rollback -version 1234
	"${next_bin}" -files "${testdata_dir}" info
}

function _get_info_as_json() {
	local -r bin="${1:?missing bin}"
	local -r jq_expression="${2:?missing jq_expression}"

	assert_file_executable "${bin}"

	"${bin}" -files "${testdata_dir}" info --format json |
		jq -c "${jq_expression}"
}

@test 'install previous version' {
	assert_not_exists "${prev_bin}"

	local install_path
	install_path="$(dirname "${prev_bin}")"
	run "${root_src_dir}/scripts/install.sh" \
		-d "${DB_DRIVER}" \
		-o "$(realpath "${install_path}")" \
		-t "${prev_version}"
	assert_file_executable "${prev_bin}"

	run "${prev_bin}" version
	assert_output --regexp "Tag.*${prev_version}"
}

@test 'perform basic operations on previous version ...' {
	assert_file_executable "${prev_bin}"

	run "${prev_bin}" version
	assert_output --regexp "Tag.*${prev_version}"

	run "${prev_bin}" -files "${testdata_dir}" info
	assert_output --partial '1234'

	run "${prev_bin}" -files "${testdata_dir}" migrate -version 1234
	assert_output --regexp 'migrating.*1234.*ok'

	run _get_info_as_json "${prev_bin}" 'select(.version == "1234").state'
	assert_output --partial up
}

# Now test backwards compatibility.
@test 'new binary encountering data created by previous version' {
	assert_file_executable "${next_bin}"

	# Expect for an error that indicates that the schema migrations table is
	# missing columns and you should upgrade
	run "${next_bin}" -files "${testdata_dir}" info
	assert_output --partial upgrade
}

@test 'upgrade schema' {
	assert_file_executable "${next_bin}"

	run "${next_bin}" -files "${testdata_dir}" upgrade
	assert_success
	assert_output --partial 'upgrade complete'
}

@test 'upgrade again does nothing' {
	assert_file_executable "${next_bin}"

	# attempting to upgrade again results in a message mentioning that
	# there is no need to upgrade.
	run "${next_bin}" -files "${testdata_dir}" upgrade
	assert_success
	assert_output --partial 'no need to upgrade'
}

@test 'works on upgraded schema' {
	assert_file_executable "${next_bin}"

	run _get_info_as_json "${next_bin}" 'select(.version == "1234").applied'
	assert_output true

	run "${next_bin}" -files "${testdata_dir}" rollback
	assert_output --partial ok

	run _get_info_as_json "${next_bin}" 'select(.version == "1234").applied'
	assert_output false

	run "${next_bin}" -files "${testdata_dir}" migrate
	assert_output --partial ok
	refute_output --partial ERROR

	run _get_info_as_json "${next_bin}" 'select(.version == "3456").applied'
	assert_output true
}
