#!/usr/bin/env sh

# Copies the code coverage files created inside a test suite's container to the
# docker host so GitHub Actions can access them.

set -eu

docker_compose_filename="${1:?missing docker_compose_filename}"

if [ ! -f "${docker_compose_filename}" ]; then
	echo >&2 "no file at ${docker_compose_filename}"
	exit 1
fi

echo >&2 "ensuring that client container is running..."
docker compose -f "${docker_compose_filename}" start client

# Only needed the client container for this step, but starting one container will probably also
# start any containers that the target depended on. Should ensure that those dependee containers are
# also stopped.
trap 'docker compose -f "${docker_compose_filename}" stop -t 5' EXIT

docker_container_id="$(docker compose -f "${docker_compose_filename}" ps -q client)"
if [ -z "${docker_container_id}" ]; then
	echo >&2 "no container id found"
	exit 1
fi

docker cp "${docker_container_id}:/tmp/cover.out" /tmp
docker cp "${docker_container_id}:/tmp/cover_driver.out" /tmp
