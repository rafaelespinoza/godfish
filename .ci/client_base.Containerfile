FROM golang:alpine

# Containers write test coverage data here
ENV TEST_COVERAGE_BASE_DIR="/tmp/test_coverage"
# GOCOVERDIR is required for capturing coverage from integration tests.
# See https://go.dev/doc/build-cover.
ENV GOCOVERDIR="${TEST_COVERAGE_BASE_DIR}/integration"
VOLUME "${TEST_COVERAGE_BASE_DIR}"

WORKDIR /src
RUN apk update && apk --no-cache add gcc g++ git just
COPY go.mod /src
RUN go mod download && go mod verify && mkdir -pv "${GOCOVERDIR}"
COPY . /src
