FROM godfish_test/client_base:latest

WORKDIR /src
RUN apk update && \
  apk --no-cache add mysql-client && \
  make build-mysql

COPY .ci/mysql/client.sh /
ENTRYPOINT /client.sh
