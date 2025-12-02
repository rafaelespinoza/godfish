FROM godfish_test/client_base:latest

WORKDIR /src
RUN just build-cassandra

# Alpine linux doesn't have a cassandra client. Build a golang binary to check
# if server is ready and setup the test DB. Use it in the entrypoint.
WORKDIR /src/drivers/cassandra/internal/ci
RUN go build -v -o /client_setup_keyspace .

WORKDIR /src/.ci/cassandra
COPY .ci/cassandra/client.sh /

WORKDIR /src
ENTRYPOINT /client.sh
