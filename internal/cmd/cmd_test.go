package cmd

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
	"github.com/rafaelespinoza/godfish/internal/stub"
)

func TestRoot(t *testing.T) {
	ctx := context.Background()
	t.Setenv(internal.DSNKey, t.Name())
	testdir := t.TempDir()

	args := [][]string{
		{"help"},
		{"create-migration"},
		{"create-migration", "-h"},
		{"create-migration", "-fwdlabel", "up"},
		{"create-migration", "-revlabel", "down"},
		{"info"},
		{"info", "-h"},
		{"info", "-format", "json"},
		{"info", "-direction", "reverse"},
		{"init", "-conf", filepath.Join(testdir, "test.json")},
		{"init", "-h"},
		{"migrate"},
		{"migrate", "-h"},
		{"remigrate"},
		{"remigrate", "-h"},
		{"rollback"},
		{"rollback", "-h"},
		{"upgrade"},
		{"upgrade", "-h"},
		{"version"},
		{"version", "-json"},
		{"version", "-h"},
	}
	for _, cmdAndArgs := range args {
		t.Run(strings.Join(cmdAndArgs, " "), func(t *testing.T) {
			godfishFlags := []string{
				"dummy_bin_value",
				"-conf", filepath.Join(testdir, ".godfish.json"),
				"-files", testdir,
			}
			combinedArgs := append(godfishFlags, cmdAndArgs...)

			err := New(stub.NewDriver(), "test").Run(ctx, combinedArgs)
			t.Log(err)
		})
	}
}

func TestDBDSN(t *testing.T) {
	testdir := t.TempDir()

	tests := []struct {
		name    string
		envVal  string
		flagVal string
		exp     string
		expErr  bool
	}{
		{
			name:   "env var already set, no flag value",
			envVal: "env_val",
			exp:    "env_val",
		},
		{
			name:    "env var already set, flag value overrides env val",
			envVal:  "env_val",
			flagVal: "flag_val",
			exp:     "flag_val",
		},
		{
			name:    "env var not set, flag value set",
			flagVal: "flag_val",
			exp:     "flag_val",
		},
		{
			name:   "env var not set, flag value not set",
			expErr: true,
		},
		{
			name:    "bad flag value set",
			flagVal: "bad\x00",
			expErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(internal.DSNKey, test.envVal)

			godfishFlags := []string{"dummy_bin_value", "-files", testdir}
			if test.flagVal != "" {
				godfishFlags = append(godfishFlags, "-dsn", test.flagVal)
			}
			combinedArgs := append(godfishFlags, "info")

			err := New(stub.NewDriver(), "test").Run(t.Context(), combinedArgs)
			if !test.expErr && err != nil {
				t.Fatal(err)
			} else if test.expErr && err == nil {
				t.Fatal("expected an error but got nil")
			} else if test.expErr && err != nil {
				if !strings.Contains(err.Error(), internal.DSNKey) {
					t.Fatalf("expected for error message (%s) to mention %s", err.Error(), internal.DSNKey)
				}
				return // OK
			}

			got, defined := os.LookupEnv(internal.DSNKey)
			if !defined {
				t.Fatalf("expected for env var %s to be defined", internal.DSNKey)
			}
			if got != test.exp {
				t.Errorf("wrong value; got %q, expected %q", got, test.exp)
			}
		})
	}
}

func TestWithConnection(t *testing.T) {
	t.Run("dsn unset", func(t *testing.T) {
		t.Setenv(internal.DSNKey, "")

		conn := connector{}
		err := withConnection(t.Context(), "", &conn, func(ictx context.Context) error {
			return nil
		})

		if err == nil {
			t.Fatal("expected an error")
		}
		if m := err.Error(); !strings.Contains(m, internal.DSNKey) {
			t.Errorf("expected for error message (%v) to contain %q", m, internal.DSNKey)
		}
	})

	for _, test := range []struct {
		name      string
		inputDSN  string
		dsnEnvVar string
	}{
		{name: "input non-empty, env var empty", inputDSN: "input", dsnEnvVar: ""},
		{name: "input empty, env var non-empty", inputDSN: "", dsnEnvVar: "envvar"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("err Connect", func(t *testing.T) {
				conn := connector{
					ConnectFn: func(d string) error { return errors.New("connect") },
				}

				t.Setenv(internal.DSNKey, test.dsnEnvVar)
				err := withConnection(t.Context(), test.inputDSN, &conn, func(ictx context.Context) error {
					return nil
				})

				if err == nil {
					t.Fatal("expected an error")
				}
				if m := err.Error(); !strings.Contains(m, "connect") {
					t.Errorf("expected for error message (%v) to contain %q", m, "connect")
				}
			})

			t.Run("err Close", func(t *testing.T) {
				// capture log message
				var buf bytes.Buffer
				l := slog.Default()
				t.Cleanup(func() { slog.SetDefault(l) })
				slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

				conn := connector{
					ConnectFn: func(d string) error { return nil },
					CloseFn:   func() error { return errors.New("close") },
				}

				t.Setenv(internal.DSNKey, test.dsnEnvVar)
				err := withConnection(t.Context(), test.inputDSN, &conn, func(ictx context.Context) error {
					return nil
				})

				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}

				if got := buf.String(); !strings.Contains(got, "close") {
					t.Errorf("expected for log message (%v) to contain %q", got, "close")
				}
			})

			t.Run("err callback func f", func(t *testing.T) {
				var calledConnect, calledClose bool
				conn := connector{
					ConnectFn: func(d string) error {
						calledConnect = true
						return nil
					},
					CloseFn: func() error {
						calledClose = true
						return nil
					},
				}

				t.Setenv(internal.DSNKey, test.dsnEnvVar)
				err := withConnection(t.Context(), test.inputDSN, &conn, func(ictx context.Context) error {
					return errors.New("press f to pay respects")
				})

				if err == nil {
					t.Fatal("expected an error")
				}
				if m := err.Error(); !strings.Contains(m, "press f to pay respects") {
					t.Errorf("expected for error message (%v) to contain %q", m, "press f to pay respects")
				}
				if !calledConnect {
					t.Error("expected to call Connect")
				}
				if !calledClose {
					t.Error("expected to call Close")
				}
			})

			t.Run("ok", func(t *testing.T) {
				var calledConnect, calledClose bool
				conn := connector{
					ConnectFn: func(d string) error {
						calledConnect = true
						return nil
					},
					CloseFn: func() error {
						calledClose = true
						return nil
					},
				}
				var calledF bool

				t.Setenv(internal.DSNKey, test.dsnEnvVar)
				err := withConnection(t.Context(), test.inputDSN, &conn, func(ictx context.Context) error {
					calledF = true
					return nil
				})

				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}
				if !calledConnect {
					t.Error("expected to call Connect")
				}
				if !calledClose {
					t.Error("expected to call Close")
				}
				if !calledF {
					t.Error("expected to call function f")
				}
			})
		})
	}
}

type connector struct {
	ConnectFn func(d string) error
	CloseFn   func() error
}

func (c *connector) Connect(dsn string) error {
	if c.ConnectFn == nil {
		panic("define ConnectFn")
	}
	return c.ConnectFn(dsn)
}

func (c *connector) Close() error {
	if c.CloseFn == nil {
		panic("define CloseFn")
	}
	return c.CloseFn()
}
