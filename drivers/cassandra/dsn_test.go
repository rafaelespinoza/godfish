package cassandra

import (
	"testing"
	"time"

	"github.com/gocql/gocql"
)

func TestNewClusterConfig(t *testing.T) {
	type testCase struct {
		input    string
		expected *gocql.ClusterConfig
		expErr   bool
	}
	const defaultExpectedTimeout = 600 * time.Millisecond

	runTest := func(t *testing.T, test testCase) {
		t.Helper()

		got, err := newClusterConfig(test.input)
		if !test.expErr && err != nil {
			t.Error(err)
		} else if test.expErr && err == nil {
			t.Error("expected error, got nil")
		} else if test.expErr && err != nil {
			return
		}

		exp := test.expected

		if got.Keyspace != exp.Keyspace {
			t.Errorf(
				"wrong Keyspace; got %q, expected %q",
				got.Keyspace, exp.Keyspace,
			)
		}

		if len(got.Hosts) != len(exp.Hosts) {
			t.Errorf(
				"wrong number of Hosts; got %d, expected %d",
				len(got.Hosts), len(exp.Hosts),
			)
		}
		for j, host := range got.Hosts {
			if host != exp.Hosts[j] {
				t.Errorf("wrong Hosts[%d]; got %q, expected %q", j, host, exp.Hosts[j])
			}
		}

		if got.ProtoVersion != exp.ProtoVersion {
			t.Errorf("wrong ProtoVersion; got %d, expected %d", got.ProtoVersion, exp.ProtoVersion)
		}

		if got.Timeout != exp.Timeout {
			t.Errorf("wrong Timeout; got %d, expected %d", got.Timeout, exp.Timeout)
		}

		if got.ConnectTimeout != exp.ConnectTimeout {
			t.Errorf("wrong ConnectTimeout; got %d, expected %d", got.ConnectTimeout, exp.ConnectTimeout)
		}

		if got.Authenticator == nil && exp.Authenticator != nil {
			t.Error("expected Authenticator, got nil")
		} else if got.Authenticator != nil && exp.Authenticator == nil {
			t.Error("unexpected Authenticator")
		} else if got.Authenticator == nil && exp.Authenticator == nil {
			return
		}
		gotAuth := got.Authenticator.(gocql.PasswordAuthenticator)
		expAuth := exp.Authenticator.(gocql.PasswordAuthenticator)
		if gotAuth.Username != expAuth.Username {
			t.Errorf("got %q, expected %q", gotAuth.Username, expAuth.Username)
		}
		if gotAuth.Password != expAuth.Password {
			t.Errorf("got %q, expected %q", gotAuth.Password, expAuth.Password)
		}
	}

	t.Run("ok", func(t *testing.T) {
		runTest(t, testCase{
			input: "cassandra://foo/bar",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})

		runTest(t, testCase{
			// Really, the schema section of the input DSN doesn't matter as
			// long as it's here.
			input: "dummy://foo/bar",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})
	})

	t.Run("multiple hosts", func(t *testing.T) {
		runTest(t, testCase{
			input: "cassandra://foo,bar/baz",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo", "bar"},
				Keyspace:       "baz",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})

		runTest(t, testCase{
			input: "cassandra://foo:123,bar:234/baz",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo:123", "bar:234"},
				Keyspace:       "baz",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})
	})

	t.Run("protocol version", func(t *testing.T) {
		runTest(t, testCase{
			input: "cassandra://foo/bar?protocol_version=3",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				ProtoVersion:   3,
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})
	})

	t.Run("timeouts", func(t *testing.T) {
		runTest(t, testCase{
			input: "cassandra://foo/bar?timeout_ms=2000",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				Timeout:        2 * time.Second,
				ConnectTimeout: defaultExpectedTimeout,
			},
		})

		runTest(t, testCase{
			input: "cassandra://foo/bar?connect_timeout_ms=3000",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: 3 * time.Second,
			},
		})
	})

	t.Run("authentication", func(t *testing.T) {
		runTest(t, testCase{
			input: "cassandra://username:password@foo/bar",
			expected: &gocql.ClusterConfig{
				Hosts:          []string{"foo"},
				Keyspace:       "bar",
				Timeout:        defaultExpectedTimeout,
				ConnectTimeout: defaultExpectedTimeout,
				Authenticator:  gocql.PasswordAuthenticator{Username: "username", Password: "password"},
			},
		})
	})

	// These are example inputs that are not expected to work at all.
	t.Run("err", func(t *testing.T) {
		runTest(t, testCase{
			input:  "foo/bar",
			expErr: true, // missing schema
		})

		runTest(t, testCase{
			input:  "cassandra://foo",
			expErr: true, // missing keyspace
		})
	})
}
