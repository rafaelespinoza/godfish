FROM godfish_test/client_base:latest
LABEL driver=sqlite3
WORKDIR /src

ENV CGO_ENABLED=1
ENV DB_DSN="file:/godfish_test.db"

RUN apk update && \
  apk --no-cache add musl-dev sqlite && \
  go build -v ./drivers/sqlite3/godfish && \
  go test -c . && \
  go test -c ./drivers/sqlite3

COPY .ci/sqlite3/client.sh /
ENTRYPOINT /client.sh
