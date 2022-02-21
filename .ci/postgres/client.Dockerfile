FROM godfish_test/client_base:latest

WORKDIR /src
RUN apk update && \
  apk --no-cache add postgresql-client && \
  make build-postgres

COPY .ci/postgres/client.sh /
ENTRYPOINT /client.sh
