#!/usr/bin/env sh

# Copies the code coverage files created inside a test suite's container to the
# docker host so GitHub Actions can access them.

set -eu

docker_compose_filename="${1:?missing docker_compose_filename}"

docker_container_id="$(docker-compose -f "${docker_compose_filename}" ps -q client)"

docker cp "${docker_container_id}:/tmp/cover.out" /tmp
docker cp "${docker_container_id}:/tmp/cover_driver.out" /tmp
