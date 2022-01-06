FROM cassandra:3.11.11
LABEL driver=cassandra role=server

# Tests run on a a single node, only need to expose the CQL listener port.
EXPOSE 9042
