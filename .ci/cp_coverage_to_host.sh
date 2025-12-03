#!/bin/sh

# Copies the code coverage files created inside a test suite's container to the
# container host so GitHub Actions can access them.

set -eu

compose_filename="${1:?missing compose_filename}"
container_tool="${CONTAINER_TOOL:-docker}"

if [ ! -f "${compose_filename}" ]; then
	echo >&2 "no file at ${compose_filename}"
	exit 1
fi

echo >&2 "ensuring that client container is running..."
"${container_tool}" compose -f "${compose_filename}" start client

# Only needed the client container for this step, but starting one container will probably also
# start any containers that the target depended on. Should ensure that those dependee containers are
# also stopped.
trap '${container_tool} compose -f "${compose_filename}" stop -t 5' EXIT

container_id="$(
	"${container_tool}" compose -f "${compose_filename}" ps --format '{{ .ID }} {{ .Names }}' |
		grep 'client' |
		awk '{ print $1 }'
)"
if [ -z "${container_id}" ]; then
	echo >&2 "no container id found"
	exit 1
fi

# Using realpath is helpful for testing on MacOS.
# Otherwise, an error message about too many symlinks may occur.
tmp_dir="$(realpath /tmp)"
"${container_tool}" cp "${container_id}:/tmp/cover.out" "${tmp_dir}"
"${container_tool}" cp "${container_id}:/tmp/cover_driver.out" "${tmp_dir}"
