FROM godfish_test/client_base:latest
LABEL driver=postgres role=client
WORKDIR /src
RUN apk --no-cache add postgresql-client
RUN go build -v ./drivers/postgres/godfish && \
  go test -c . && go test -c ./drivers/postgres
COPY .ci/postgres/client.sh /
ENTRYPOINT /client.sh
