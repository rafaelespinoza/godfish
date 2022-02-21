FROM cassandra:4.0.3

# Tests run on a single node, only need to expose the CQL listener port.
EXPOSE 9042
