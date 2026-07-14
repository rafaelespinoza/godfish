FROM golang:alpine

# Containers write test coverage data here
ENV TEST_COVERAGE_BASE_DIR="/tmp/test_coverage"
# GOCOVERDIR is required for capturing coverage from integration tests.
# See https://go.dev/doc/build-cover.
# Set TERM env var and install ncurses so that the --pretty output
# formatter may be used.
ENV GOCOVERDIR="${TEST_COVERAGE_BASE_DIR}/integration" TERM=xterm
VOLUME "${TEST_COVERAGE_BASE_DIR}"

WORKDIR /src
RUN apk update && apk --no-cache add bats gcc g++ git just jq ncurses
COPY go.mod /src
RUN go mod download && go mod verify && mkdir -pv "${GOCOVERDIR}"
COPY . /src
