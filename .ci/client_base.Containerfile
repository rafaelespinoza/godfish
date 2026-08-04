FROM golang:alpine3.24

# Prefer building without C.
ENV CGO_ENABLED=0

# Containers write test coverage data here.
ENV TEST_COVERAGE_BASE_DIR="/tmp/test_coverage"

# GOCOVERDIR is required for capturing coverage from integration tests.
# See https://go.dev/doc/build-cover.
# Set TERM env var and install ncurses so that the --pretty output
# formatter may be used.
ENV GOCOVERDIR="${TEST_COVERAGE_BASE_DIR}/integration" TERM=xterm

# A tool named task (https://taskfile.dev) is used for running tests.
# As of 2026-07, this is the most up-to-date release.
# See updates at https://github.com/go-task/task/releases.
# Avoid "latest" to help build behavior be predictable.
ARG TASK_VERSION=v3.52.0

# Mount a directory here to share code coverage data with the host.
VOLUME "${TEST_COVERAGE_BASE_DIR}"

RUN mkdir -pv "${GOCOVERDIR}" && \
  apk update && \
  apk --no-cache add bats git just jq ncurses && \
  wget -O /tmp/install_task.sh https://taskfile.dev/install.sh && \
  sh /tmp/install_task.sh -b /usr/local/bin "${TASK_VERSION}" && \
  task --version

WORKDIR /src

COPY go.mod go.sum .
RUN go mod download && go mod verify
COPY . .

ENTRYPOINT ["/usr/local/bin/task", "-d", ".ci"]
