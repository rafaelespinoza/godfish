FROM postgres:14-alpine
WORKDIR /var/lib/postgresql
COPY .ci/postgres/server.sh scripts/
RUN scripts/server.sh
EXPOSE 5432
