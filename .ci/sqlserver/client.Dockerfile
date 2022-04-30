FROM godfish_test/client_base:latest

WORKDIR /src
RUN make build-sqlserver

# Alpine linux doesn't have a SQL Server client. Build a golang binary to
# check if server is ready. Use it in the entrypoint.
WORKDIR /src/.ci/sqlserver
RUN go build -v -o /client_check_db .
COPY .ci/sqlserver/client.sh /

WORKDIR /src
ENTRYPOINT /client.sh
