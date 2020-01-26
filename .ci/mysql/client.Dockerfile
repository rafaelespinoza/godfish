FROM godfish_test/client_base:latest
LABEL driver=mysql role=client
WORKDIR /src
RUN apk update && apk --no-cache add mysql-client
RUN go build -v ./drivers/mysql/godfish && \
  go test -c . && go test -c ./drivers/mysql
COPY .ci/mysql/client.sh /
ENTRYPOINT /client.sh
