FROM godfish_test/client_base:latest

ENV CGO_ENABLED=1
ENV DB_DSN="file:/godfish_test.db"

WORKDIR /src
RUN apk update && \
  apk --no-cache add musl-dev sqlite && \
  make build-sqlite3

COPY .ci/sqlite3/client.sh /
ENTRYPOINT /client.sh
