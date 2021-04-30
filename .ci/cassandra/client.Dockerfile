FROM godfish_test/client_base:latest
LABEL driver=cassandra role=client

WORKDIR /src
RUN go build -v ./drivers/cassandra/godfish && \
  go test -c . && go test -c ./drivers/cassandra

# Alpine linux doesn't have a cassandra client. Build a golang binary to check
# if server is ready and setup the test DB. Use it in the entrypoint.
WORKDIR /src/.ci/cassandra
RUN go build -v -o /client_setup_keyspace .
COPY .ci/cassandra/client.sh /

WORKDIR /src
ENTRYPOINT /client.sh
