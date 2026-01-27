FROM godfish_test/client_base:latest

WORKDIR /src
RUN apk update && \
  apk --no-cache add postgresql-client && \
  just build-postgres && \
  just build-postgres-test

COPY .ci/postgres/client.sh /
ENTRYPOINT /client.sh
