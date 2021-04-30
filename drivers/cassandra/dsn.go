package cassandra

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

func newClusterConfig(connectionURI string) (cluster *gocql.ClusterConfig, err error) {
	dsn, err := parseDSN(connectionURI)
	if err != nil {
		return
	}

	cluster = gocql.NewCluster(dsn.hosts...)
	cluster.Keyspace = dsn.keyspace

	// If 0, then the default is to let the cluster config guess.
	cluster.ProtoVersion = dsn.protoVersion

	// The library's defaults are 600ms. Only override when the dsn specifies.
	if dsn.timeout > 0 {
		cluster.Timeout = dsn.timeout
	}
	if dsn.connectTimeout > 0 {
		cluster.ConnectTimeout = dsn.connectTimeout
	}

	if dsn.username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: dsn.username,
			Password: dsn.password,
		}
	}
	return
}

const sampleDSN = "scheme://[userinfo@]host[,more,hosts]/keyspace[?query]"

type dsn struct {
	hosts          []string
	keyspace       string
	username       string
	password       string
	protoVersion   int
	timeout        time.Duration
	connectTimeout time.Duration
}

func parseDSN(in string) (out dsn, err error) {
	uri, err := url.Parse(in)
	if err != nil {
		return
	}
	if uri.Scheme == "" {
		// The value could be nearly anything. Without something here, parsing is too complicated.
		err = fmt.Errorf(
			`input dsn should have a scheme prefix. ie: the "scheme://" part of: %q`,
			sampleDSN,
		)
		return
	}
	if uri.Path == "" {
		err = fmt.Errorf(
			`input dsn keyspace empty; should be the "keyspace" part of: %q`, sampleDSN,
		)
		return
	}

	var username, password string
	if uri.User != nil {
		username = uri.User.Username()
		password, _ = uri.User.Password()
	}

	queryVals := uri.Query()
	var protocol, timeoutMS, connectTimeoutMS int
	if protocol, err = parseInt(queryVals.Get("protocol_version")); err != nil {
		err = fmt.Errorf("%w; key %q", err, "protocol_version")
		return
	}
	if timeoutMS, err = parseInt(queryVals.Get("timeout_ms")); err != nil {
		err = fmt.Errorf("%w; key %q", err, "timeout_ms")
		return
	}
	if connectTimeoutMS, err = parseInt(queryVals.Get("connect_timeout_ms")); err != nil {
		err = fmt.Errorf("%w; key %q", err, "connect_timeout_ms")
		return
	}

	out = dsn{
		hosts:          strings.Split(uri.Host, ","),
		keyspace:       uri.Path[1:],
		username:       username,
		password:       password,
		protoVersion:   protocol,
		timeout:        time.Duration(timeoutMS * int(time.Millisecond)),
		connectTimeout: time.Duration(connectTimeoutMS * int(time.Millisecond)),
	}

	return
}

func parseInt(val string) (out int, err error) {
	if val == "" {
		return
	}
	out, err = strconv.Atoi(val)
	return
}
