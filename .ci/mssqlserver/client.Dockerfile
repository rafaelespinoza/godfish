FROM godfish_test/client_base:latest
LABEL driver=mssqlserver role=client

WORKDIR /src
RUN go build -v ./drivers/mssqlserver/godfish && \
  go test -c . && go test -c ./drivers/mssqlserver

# Alpine linux doesn't have a MS SQL Server client. Build a golang binary to
# check if server is ready. Use it in the entrypoint.
WORKDIR /src/.ci/mssqlserver
RUN go build -v -o /client_check_db .
COPY .ci/mssqlserver/client.sh /

WORKDIR /src
ENTRYPOINT /client.sh
