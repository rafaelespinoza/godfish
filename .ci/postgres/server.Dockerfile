FROM postgres:12.6-alpine
LABEL driver=postgres role=server
WORKDIR /var/lib/postgresql
COPY .ci/postgres/server.sh scripts/
RUN scripts/server.sh
EXPOSE 5432
