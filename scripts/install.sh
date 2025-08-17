#!/bin/sh

set -eu

DB_DRIVER='all'
DEFAULT_OUTPUT_DIR="${HOME}/bin"
OUTPUT_DIR="${DEFAULT_OUTPUT_DIR}"
LOG_LEVEL="${LOG_LEVEL:-1}" # could be 1 2 or 3

github_repo_owner=rafaelespinoza
github_repo_name=godfish
checksums_file=checksums.txt
db_drivers='cassandra mysql postgres sqlite3 sqlserver'

usage() {
	usage_message="Usage: ${0} [flags]

Description:
  Installation script for ${github_repo_name}. The upstream repo is
  https://github.com/${github_repo_owner}/${github_repo_name}.

  This script does these things:
  * downloads pre-compiled release assets from GitHub
  * verifies that the downloaded completed successfully
  * unarchives the downloaded archive file
  * install the selected executable(s) to an output directory

  For every release of this tool, there are several archive release files
  (tar.gz) to choose from. Automatically select the one that corresponds to the
  host machine. Each archive file has pre-compiled executables for each
  supported DB: ${db_drivers}.
  Install all of them, or choose one.

Flags:
  -d  <str>  DB driver to target.
      If unset, then it installs all of them.
      Possible values: all ${db_drivers}.
  -h  <bool> Show help.
  -o  <str>  Output directory to install binaries.
      This should be an absolute path. Default ${DEFAULT_OUTPUT_DIR}.

Examples:
  # show help
  $ ${0} -h

  # install mysql driver to default location
  $ ${0} -d sqlite3

  # install all of the drivers to /usr/local/bin
  $ ${0} -o /usr/local/bin"

	printf '%s\n' "${usage_message}" | fold -w 80 -s
}

print_debug() {
	[ "${LOG_LEVEL}" -ge 3 ] || return 0
	printf 'DEBUG %s\n' "${*}" >&2
}

print_info() {
	[ "${LOG_LEVEL}" -ge 2 ] || return 0
	printf 'INFO %s\n' "${*}" >&2
}

print_error() {
	[ "${LOG_LEVEL}" -ge 1 ] || return 0
	printf 'ERROR %s\n' "${*}" >&2
}

has_command() {
	command -v "${1}" >/dev/null 2>&1
}

needs_commands() {
	for cmd in "$@"; do
		if ! has_command "${cmd}"; then
			print_error "command '${cmd}' is needed for this script"
			exit 1
		fi
	done
}

get_host_system() {
	needs_commands uname tr

	host_system=$(uname -s | tr '[:upper:]' '[:lower:]')
	print_debug "host_system=${host_system}"

	case "${host_system}" in
		cygwin_nt* | mingw* | msys_nt*)
			host_system=windows
			;;
		darwin | linux) ;;
		*)
			print_info "host system is '${host_system}', assuming windows (😬) b/c that's the only other one that is supported right now"
			host_system=windows
			;;
	esac

	printf '%s' "${host_system}"
}

get_host_arch() {
	needs_commands arch

	host_arch="$(arch)"
	print_debug "host_arch=${host_arch}"

	case "${host_arch}" in
		arm64 | aarch64)
			host_arch=arm64
			;;
		x86_64)
			host_arch=amd64
			;;
		*)
			err_msg="Sorry, a host architecture of '${host_arch}' is unsupported at this time. Please file an issue at the repo if you are interested in support for your system at https://github.com/${github_repo_owner}/${github_repo_name}"
			print_error "$(printf '%s' "${err_msg}" | fold -w 80 -s)"
			exit 1
			;;
	esac

	printf '%s' "${host_arch}"
}

download() {
	uri="${1:?missing uri}"
	output_dest="${2:?missing output_dest}"

	# For one-offs, there shouldn't be a need to include this. But for CI, it's
	# helpful to send authenticated requests so that rate limiting is less of a
	# factor in getting the tests to work.
	# A token with permissions of `content: read` should be sufficient for now.
	auth_bearer_header=''
	if [ -n "${GITHUB_TOKEN:-}" ]; then
		auth_bearer_header="Authorization: Bearer ${GITHUB_TOKEN}"
	fi
	resp_code=0

	if has_command curl; then
		if [ -n "${auth_bearer_header:-}" ]; then
			resp_code=$(curl --write-out '%{http_code}' --fail --silent --show-error --location --output "${output_dest}" --header "${auth_bearer_header}" "${uri}")
		else
			resp_code=$(curl --write-out '%{http_code}' --fail --silent --show-error --location --output "${output_dest}" "${uri}")
		fi

		if [ "${resp_code}" != '200' ]; then
			print_error "downloading from ${uri}, status code ${resp_code}"
			return 1
		fi
	elif has_command wget; then
		if [ -n "${auth_bearer_header:-}" ]; then
			if ! wget --quiet --output-document "${output_dest}" --header "${auth_bearer_header}" "${uri}"; then
				print_error "error downloading from ${uri}"
				return 1
			fi
		else
			if ! wget --quiet --output-document "${output_dest}" "${uri}"; then
				print_error "error downloading from ${uri}"
				return 1
			fi
		fi
	else
		print_error "unable to find command to download release assets"
		return 1
	fi
}

download_release_assets() {
	pattern="${1:?missing pattern}"

	print_info 'getting assets from github...'

	# Prefer to use gh if available, it's much simpler for this task.
	if has_command gh; then
		# Use the pattern variable as a glob for this usage b/c that's what this
		# command expects
		gh -R "${github_repo_owner}/${github_repo_name}" release download -p ${checksums_file} -p "*${pattern}*"
		return 0
	fi

	# No gh? Consider installing it (https://cli.github.com/). Try this for now.

	# Get metadata about the releases.
	release_info='latest_release.json'
	remote_release_url="https://api.github.com/repos/${github_repo_owner}/${github_repo_name}/releases/latest"
	download "${remote_release_url}" "${release_info}"

	# Figure out what the remote filenames are.
	if has_command jq; then
		remote_checksums="$(jq --arg pattern "${checksums_file}" --raw-output '.assets[].browser_download_url | select(contains($pattern))' <"${release_info}")"
	else
		# NOTE: should github minify the JSON response body, this would break.
		# It relies on each attribute to be on its own line.
		remote_checksums="$(grep "browser_download_url.*${checksums_file}" <"${release_info}" | cut -d ':' -f 2,3 | tr -d '"' | tr -d ' ')"
	fi
	print_debug "remote_checksums='${remote_checksums}'"

	if [ -z "${remote_checksums}" ]; then
		print_error "failed to find remote checksums file with name matching ${checksums_file}"
		return 1
	fi

	if has_command jq; then
		remote_asset_filename="$(jq --arg pattern "${pattern}" --raw-output '.assets[].browser_download_url | select(contains($pattern))' <"${release_info}")"
	else
		remote_asset_filename="$(grep "browser_download_url.*${pattern}" <"${release_info}" | cut -d ':' -f 2,3 | tr -d '"' | tr -d ' ')"
	fi
	print_debug "remote_asset_filename='${remote_asset_filename}'"

	if [ -z "${remote_asset_filename}" ]; then
		print_error "failed to find remote asset with name matching ${pattern}"
		return 1
	fi

	# Now get the remote files.
	download "${remote_checksums}" "${checksums_file}"
	download "${remote_asset_filename}" "$(basename "${remote_asset_filename}")"
}

get_sha256() {
	archive_file="${1:?missing archive_file}"

	# Based off this great installation script.
	# https://github.com/twpayne/chezmoi/blob/master/assets/scripts/install.sh

	if has_command sha256sum; then
		h="$(sha256sum "${archive_file}")" || return 1
		printf '%s' "${h}" | cut -d ' ' -f 1
	elif has_command shasum; then
		h="$(shasum -a 256 "${archive_file}" 2>/dev/null)" || return 1
		printf '%s' "${h}" | cut -d ' ' -f 1
	elif has_command sha256; then
		h="$(sha256 -q "${archive_file}" 2>/dev/null)" || return 1
		printf '%s' "${h}" | cut -d ' ' -f 1
	elif has_command openssl; then
		h="$(openssl dgst -sha256 "${archive_file}" 2>/dev/null)" || return 1
		# the output format of this command would look like this:
		# SHA2-256(name_of_the_file.tar.gz)= 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
		printf '%s' "${h}" | cut -d ' ' -f 2
	else
		print_error "unable to find command to compute sha256"
		return 1
	fi
}

verify_checksum() {
	needs_commands grep find

	pattern="${1:?missing pattern}"

	print_info "verifying checksum..."

	expected_checksum=$(grep "${pattern}" "${checksums_file}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
	if [ -z "${expected_checksum}" ]; then
		print_error "could not find a checksum for ${pattern} in file ${checksums_file}"
		return 1
	fi

	archive_file=$(find . -type f -iname '*.tar.gz')
	if [ -z "${archive_file}" ]; then
		print_error "could not find archive file for verifying checksum"
		return 1
	fi

	got_checksum="$(get_sha256 "${archive_file}")"
	if [ "${got_checksum}" != "${expected_checksum}" ]; then
		print_error "it appears that the checksum for ${archive_file} does not match"
		print_error "${got_checksum} != ${expected_checksum}"
		return 1
	fi
}

unarchive_and_install() {
	needs_commands tar find install

	print_info "unarchiving..."

	mkdir -p ./bin

	if ! tar --directory ./bin --extract --file ./*.tar.gz*; then
		print_error 'extracting tar failed'
		return 1
	fi
	find . -type f -iname '*.tar.gz' -exec rm {} +

	if [ ! -d "${OUTPUT_DIR}" ]; then
		install -d "${OUTPUT_DIR}"
	fi

	find_expr='godfish*'
	if [ "${DB_DRIVER}" != 'all' ]; then
		find_expr="godfish_${DB_DRIVER}"
	fi

	# This invocation of find should work on systems that have BSD-style tooling
	# or have GNU-style tooling.
	find ./bin -type f -name "${find_expr}" >matching_files
	if [ ! -s matching_files ]; then
		print_error "no matching files in unarchived assets. Pass in a DB driver name to target (one of: ${db_drivers}) or 'all' for this script"
		return 1 # an infinite loop may occur unless it stops early here.
	fi

	while IFS="" read -r filename || [ -n "${filename}" ]; do
		if [ ! -x "${filename}" ]; then
			print_debug "skipping non-executable file ${filename}"
			continue
		fi

		[ "${LOG_LEVEL}" -ge 3 ] && "${filename}" version
		install -m 0755 -- "${filename}" "${OUTPUT_DIR}/"
		# always display this regardless of the LOG_LEVEL
		printf "installed %s to %s\n" "$(basename "${filename}")" "${OUTPUT_DIR}"
	done <matching_files
}

main() {
	if [ "${#}" = 0 ]; then
		usage
		return 0
	elif [ "${#}" = 1 ]; then
		case "${1}" in
			help | -help | --help | -h)
				usage
				return 0
				;;
			*) ;;
		esac
	fi

	while getopts "d:ho:" opt; do
		case "${opt}" in
			d) DB_DRIVER="${OPTARG}" ;;
			h | \?)
				usage
				return 0
				;;
			o)
				OUTPUT_DIR="${OPTARG}"
				;;
			*)
				usage
				print_error "unknown flag ${OPTARG}"
				return 1
				;;
		esac
	done
	print_debug "DB_DRIVER=${DB_DRIVER} LOG_LEVEL=${LOG_LEVEL} OUTPUT_DIR=${OUTPUT_DIR}"

	# Ensure that the installation step may work properly.
	if [ "${OUTPUT_DIR}" = "${OUTPUT_DIR#/}" ]; then
		# The directory doesn't need to exist yet, it can be created upon demand.
		usage
		print_error "output directory (${OUTPUT_DIR}) must be an absolute path to a directory"
		return 1
	fi

	# Another input validation
	db_driver_ok=0
	for driver in $db_drivers; do
		if [ "${DB_DRIVER}" = "${driver}" ] || [ "${DB_DRIVER}" = 'all' ]; then
			db_driver_ok=1
			break
		fi
	done
	if [ "${db_driver_ok}" -ne 1 ]; then
		usage
		print_error "selected db_driver (${DB_DRIVER}) invalid. Should be 'all' or one of: ${db_drivers}"
		return 1
	fi

	# Figure out which release asset to download based on host environment.
	host_system=$(get_host_system)
	host_arch=$(get_host_arch)

	build_dir="$(mktemp -d -p /tmp godfish_XXXXXX)"
	trap 'rm -rf -- "${build_dir}"' EXIT
	trap 'exit' INT TERM

	cd "${build_dir}" && print_debug "build_dir=$(pwd)"

	pattern="${host_system}_${host_arch}"
	print_debug "pattern=${pattern}"

	download_release_assets "${pattern}"
	verify_checksum "${pattern}"
	unarchive_and_install
}

main "$@"
