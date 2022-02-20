# cassandra

This `godfish.Driver` implementation has been tested against cassandra versions:

- 3.11.12
- 4.0.3

## Connecting

Like other `godfish.Driver` implementations, you must specify an environment
variable, `DB_DSN`, to connect to the DB. There does not seem to be a standard
connection URI schema for cassandra, but nonetheless this library expects a
`DB_DSN` value. The form is roughly:

```
scheme://[userinfo@]host[,more,hosts]/keyspace[?query]
```

It's parsed by `net/url.Parse` from the standard library and ends up making a
`*gocql.ClusterConfig`. See the tests for working and non-working examples.

### DSN Components

- `scheme`: Required, the value can be something like `cassandra`, or really
  anything, followed by a `://`. If this is empty or malformed, then the parsing
  function gets confused, and may mix up the host and the keyspace. So, just to
  make all of this easier, it's best to put something here.
- `userinfo`: Optionally specify username and password.
- `host`: Required, IP address of DB server.
  - may also include a port, in the form: `ip_address:port` 
  - optionally add `comma,delimited,hosts`
- `keyspace`: Required, should be the first "path" in the DSN string.
- `query`: Various options, represented as query string key value pairs. If any
  key is unspecified or the value is zero, then the corresponding field for
  `gocql.ClusterConfig` is set to its default.
  - `connect_timeout_ms`: Integer, milliseconds. Sets `gocql.ClusterConfig.ConnectTimeout`.
  - `protocol_version`: Integer. Sets `gocql.ClusterConfig.ProtoVersion`.
  - `timeout_ms`: Integer, milliseconds. Sets `gocql.ClusterConfig.Timeout`.
