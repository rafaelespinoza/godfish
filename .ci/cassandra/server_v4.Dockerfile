FROM cassandra:4.0.1
LABEL driver=cassandra role=server

# Tests run on a single node, only need to expose the CQL listener port.
EXPOSE 9042
