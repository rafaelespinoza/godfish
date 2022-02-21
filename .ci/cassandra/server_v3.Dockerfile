FROM cassandra:3.11.12

# Tests run on a single node, only need to expose the CQL listener port.
EXPOSE 9042
