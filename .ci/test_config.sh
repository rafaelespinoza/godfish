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
	bats_require_minimum_version 1.5.0
	mkdir -pv "${GOCOVERDIR}"

	# Most variables set here are meant to be in the global scope so they can be
	# read from tests and other functions. Those variables must be exported.

	default_config_file=.godfish.json # use a relative path here.

	local root_src_dir
	root_src_dir=$(git rev-parse --show-toplevel)

	local -r testdata_base_dir="${root_src_dir}/testdata"
	case "${DB_DRIVER}" in
	cassandra | sqlserver)
		src_testdata_dir="${testdata_base_dir}/${DB_DRIVER}"
		;;
	*)
		src_testdata_dir="${testdata_base_dir}/default"
		;;
	esac

	# driver_bin can be created with the Justfile. $ just build-${DB_DRIVER}-test.
	driver_bin="${root_src_dir}/bin/godfish-${DB_DRIVER}_test"

	readonly default_config_file driver_bin src_testdata_dir
	export default_config_file driver_bin src_testdata_dir
}

function setup() {
	assert_file_executable "${driver_bin}"
	assert_dir_exists "${src_testdata_dir}"

	pushd "${BATS_TEST_TMPDIR}"
}

function teardown() {
	if [[ -e "${default_config_file}" ]]; then
		rm -v "${default_config_file}"
	fi

	popd
}

# _make_config_json constructs JSON config data and writes to stdout.
# The inputs are:
# 	1. migrations_table optional, default-''.
# 	2. path_to_files    optional, default-''.
function _make_config_json() {
	local migrations_table="${1:-}"
	local path_to_files="${2:-}"

	jq --null-input \
		--arg migrations_table "${migrations_table}" \
		--arg path_to_files "${path_to_files}" \
		'{migrations_table: $migrations_table, path_to_files: $path_to_files}'
}

# _get_info_as_json calls godfish info --format json and pipes the result to jq.
# The inputs are:
# 	1. migrations_table optional, default-'', if '' then flag is not passed.
# 	2. path_to_files    optional, default-'', if '' then flag is not passed.
# 	3. jq_expression    optional, default='.',
function _get_info_as_json() {
	local -r migrations_table="${1:-}"
	local -r path_to_files="${2:-}"
	local -r jq_expression="${3:-.}" # Defaults to '.' (identity filter) if empty

	local godfish_args=()
	if [[ -n "${migrations_table}" ]]; then
		godfish_args+=( "-migrations-table" "${migrations_table}" )
	fi
	if [[ -n "$path_to_files" ]]; then
		godfish_args+=( "-files" "${path_to_files}" )
	fi

	"${driver_bin}" "${godfish_args[@]}" info --format json |
		jq --compact-output --raw-output "${jq_expression}"
}

@test 'no config file, no flags specified' {
	assert_not_exists "${default_config_file}"

	run --separate-stderr "${driver_bin}" info
	assert_failure
}

@test 'no config file, flag -files=something' {
	assert_not_exists "${default_config_file}"

	run --separate-stderr _get_info_as_json '' "${src_testdata_dir}" '.version'
	assert_success
	assert_output --partial 1234
	assert_output --partial 2345
	assert_output --partial 3456
}

@test 'config.path_to_files is set, no -files flag' {
	_make_config_json "" "${src_testdata_dir}" >"${default_config_file}"
	assert_exists "${default_config_file}"

	run --separate-stderr _get_info_as_json '' '' '.version'
	assert_success
	assert_output --partial 1234
	assert_output --partial 2345
	assert_output --partial 3456
}

@test 'config.path_to_files is set, flag -files=something' {
	_make_config_json "" "${src_testdata_dir}" | tee "${default_config_file}"
	assert_exists "${default_config_file}"

	local -r test_testdata_dir="${BATS_TEST_TMPDIR}/testdata"
	mkdir -pv "${test_testdata_dir}"
	cp -v "${src_testdata_dir}/forward-2345-bravo.sql" "${test_testdata_dir}"
	assert_exists "${test_testdata_dir}/forward-2345-bravo.sql"

	# The command line flag has higher precedence than config file value.
	run --separate-stderr _get_info_as_json '' "${test_testdata_dir}" '.version'
	assert_success
	refute_output --partial 1235
	assert_output --partial 2345
	refute_output --partial 3456
}

# migrations-table tests. These involve running migrations with different
# configurations for the migrations-table, and then checking the results.

@test 'config.schema_migrations unset, no -migrations-table flag' {
	_make_config_json "" "${src_testdata_dir}" | tee "${default_config_file}"
	assert_exists "${default_config_file}"
	run --separate-stderr _get_info_as_json '' '' 'select(.applied)'
	refute_output

	run --separate-stderr "${driver_bin}" migrate -version 1234
	assert_success

	run --separate-stderr _get_info_as_json '' '' 'select(.applied).version'
	assert_success
	assert_output --partial 1234
	refute_output --partial 2345
	refute_output --partial 3456

	# reset state for next tests
	run --separate-stderr "${driver_bin}" rollback -version 1234
	assert_success
}

@test 'config.schema_migrations unset, flag -migrations-table=something' {
	local -r migrations_table=migrations_flag
	_make_config_json "" "${src_testdata_dir}" | tee "${default_config_file}"
	assert_exists "${default_config_file}"
	run --separate-stderr _get_info_as_json "${migrations_table}" '' 'select(.applied)'
	refute_output

	run --separate-stderr "${driver_bin}" -migrations-table "${migrations_table}" migrate -version 1234
	assert_success

	run --separate-stderr _get_info_as_json "${migrations_table}" '' 'select(.applied).version'
	assert_success
	assert_output --partial 1234
	refute_output --partial 2345
	refute_output --partial 3456

	# reset state for next tests
	run --separate-stderr "${driver_bin}" -migrations-table ${migrations_table} rollback -version 1234
	assert_success
}

@test 'config.schema_migrations is set, no -migrations-table flag' {
	local -r migrations_table=other_migrations
	_make_config_json "${migrations_table}" "${src_testdata_dir}" | tee "${default_config_file}"
	assert_exists "${default_config_file}"
	run --separate-stderr _get_info_as_json '' '' 'select(.applied).version'
	refute_output

	run --separate-stderr "${driver_bin}" migrate -version 1234
	assert_success

	run --separate-stderr _get_info_as_json '' '' 'select(.applied).version'
	assert_success
	assert_output --partial 1234
	refute_output --partial 2345
	refute_output --partial 3456
	# Now check the default table, which should not output any applied migrations.
	run --separate-stderr _get_info_as_json 'schema_migrations' '' 'select(.applied).version'
	assert_success
	refute_output

	# reset state for next tests
	run --separate-stderr "${driver_bin}" rollback -version 1234
	assert_success
}

@test 'config.schema_migrations is set, flag -migrations-table=something_else' {
	local -r migrations_table_config=other_migrations
	local -r migrations_table_flag=even_more_migrations
	_make_config_json "${migrations_table_config}" "${src_testdata_dir}" | tee "${default_config_file}"
	assert_exists "${default_config_file}"
	run --separate-stderr _get_info_as_json '' '' 'select(.applied)'
	refute_output
	run --separate-stderr _get_info_as_json "${migrations_table_flag}" '' 'select(.applied)'
	refute_output

	# The command line value has higher priority than a config file value.
	run "${driver_bin}" -migrations-table "${migrations_table_flag}" migrate -version 1234
	assert_success

	run --separate-stderr _get_info_as_json "${migrations_table_flag}" '' 'select(.applied).version'
	assert_success
	assert_output --partial 1234
	refute_output --partial 2345
	refute_output --partial 3456
	# Now check the table from the config file, which should not output any applied migrations.
	run --separate-stderr _get_info_as_json "${migrations_table_config}" '' 'select(.applied).version'
	assert_success
	refute_output

	# reset state for next tests
	run --separate-stderr "${driver_bin}" -migrations-table "${migrations_table_flag}" rollback -version 1234
	assert_success
}
