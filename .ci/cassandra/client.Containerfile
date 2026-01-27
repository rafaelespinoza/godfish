FROM godfish_test/client_base:latest

WORKDIR /src
RUN just build-cassandra && \
  just build-cassandra-test

# Alpine linux doesn't have a cassandra client. Build a golang binary to check
# if server is ready and setup the test DB. Use it in the entrypoint.
WORKDIR /src/.ci/cassandra
RUN go build -v -o /client_setup_keyspace .
COPY .ci/cassandra/client.sh /

WORKDIR /src
ENTRYPOINT /client.sh
