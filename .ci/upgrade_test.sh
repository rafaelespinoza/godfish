#!/bin/sh

set -eu

# Input args

: "${DB_DSN:?DB_DSN is required}"
: "${GOCOVERDIR:?GOCOVERDIR is required}"
driver_name="${1:?missing name of driver}"

# other variables

print_info() {
	printf 'INFO %s\n' "${*}" >&2
}

print_error() {
	printf 'ERROR %s\n' "${*}" >&2
}

root_src_dir=$(git rev-parse --show-toplevel)
work_base_dir="$(mktemp -d -p /tmp godfish_XXXXXX)"
work_bin_dir="${work_base_dir}/bin"
prev_version='v0.14.0'
# next_bin is the binary to test for upgrades. It can be created with the
# Justfile. $ just build-${driver_name}-test
next_bin="${root_src_dir}/bin/godfish_${driver_name}_test"
testdata_base_dir="${root_src_dir}/testdata"
case "${driver_name}" in
	cassandra | sqlserver)
		testdata_dir="${testdata_base_dir}/${driver_name}"
		;;
	*)
		testdata_dir="${testdata_base_dir}/default"
		;;
esac

trap 'rm -rf -- "${work_base_dir}"' EXIT
trap 'exit' INT TERM

#
# test begins here
#

[ -x "${next_bin}" ] || (
	print_error "it appears that ${next_bin} is not executable"
	exit 1
)

# Set up data with the older schema.
print_info "installing previous version (${prev_version}) ..."
"${root_src_dir}/scripts/install.sh" \
	-d "${driver_name}" \
	-o "$(realpath "${work_bin_dir}")" \
	-t "${prev_version}"
prev_bin=$(
	find "${work_bin_dir}" -maxdepth 1 -type f -name "godfish_${driver_name}"
)
"${prev_bin}" version

print_info 'doing basic operations on previous version ...'
"${prev_bin}" -files "${testdata_dir}" info
"${prev_bin}" -files "${testdata_dir}" migrate -version 1234
"${prev_bin}" -files "${testdata_dir}" info

# Now test backwards compatibility.
print_info 'testing a newer version with data created by previous version'
mkdir -pv "${GOCOVERDIR}"
# Expect for an error that indicates that the schema migrations table is missing
# columns and you should upgrade
info_err=$("${next_bin}" -files "${testdata_dir}" info 2>&1 > /dev/null) || true
print_info "got this message: '${info_err}'"
if ! echo "${info_err}" | grep 'upgrade' > /dev/null; then
	print_error "expected for error message to mention 'upgrade'"
	exit 1
fi
"${next_bin}" -files "${testdata_dir}" upgrade
"${next_bin}" -files "${testdata_dir}" info

"${next_bin}" -files "${testdata_dir}" rollback
"${next_bin}" -files "${testdata_dir}" info
"${next_bin}" -files "${testdata_dir}" migrate
"${next_bin}" -files "${testdata_dir}" info

# "reset"
"${next_bin}" -files "${testdata_dir}" rollback -version 1234
"${next_bin}" -files "${testdata_dir}" info
