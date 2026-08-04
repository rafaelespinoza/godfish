FROM godfish_test/client_base:latest

WORKDIR /src
RUN apk update && \
  apk --no-cache add mysql-client && \
  just build-mysql && \
  just build-mysql-test

CMD ["mysql:all"]
