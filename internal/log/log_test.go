package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"testing"
)

type LeveledLoggingTestcase struct {
	level     slog.Level
	inputErr  error
	logAction func()
	expLevel  slog.Level
	expMsg    string
	expData   map[string]any
}

func TestInfo(t *testing.T) {
	testLeveledLogging(t, LeveledLoggingTestcase{
		level:     slog.LevelInfo,
		logAction: func() { Info(context.Background(), t.Name(), slog.String("foo", "bar")) },
		expLevel:  slog.LevelInfo,
		expMsg:    "godfish: " + t.Name(),
		expData:   map[string]any{"foo": "bar"},
	})
}

func TestWarn(t *testing.T) {
	testLeveledLogging(t, LeveledLoggingTestcase{
		level:     slog.LevelWarn,
		logAction: func() { Warn(context.Background(), t.Name(), slog.String("foo", "bar")) },
		expLevel:  slog.LevelWarn,
		expMsg:    "godfish: " + t.Name(),
		expData:   map[string]any{"foo": "bar"},
	})
}

func TestError(t *testing.T) {
	err := errors.New("OOF")
	testLeveledLogging(t, LeveledLoggingTestcase{
		level:     slog.LevelError,
		logAction: func() { Error(context.Background(), err, t.Name(), slog.String("foo", "bar")) },
		inputErr:  err,
		expLevel:  slog.LevelError,
		expMsg:    "godfish: " + t.Name(),
		expData:   map[string]any{"foo": "bar", "error": err.Error()},
	})
}

func testLeveledLogging(t *testing.T, test LeveledLoggingTestcase) {
	// The function under test uses a logger in the package level scope. Replace
	// that temporarily so its data can be observed. The test replacement
	// handler is JSON to make it easy to parse individual fields.
	originalLogger := theLogger
	t.Cleanup(func() { theLogger = originalLogger })
	var buf bytes.Buffer

	SetLogger(&buf, test.level.String(), "JSON")

	test.logAction()

	var gotMessage JSONMessage
	rawMessage := buf.Bytes()
	if err := json.Unmarshal(rawMessage, &gotMessage); err != nil {
		t.Fatal(err)
	}

	t.Run("Level", func(t *testing.T) {
		if gotMessage.Level != test.expLevel.String() {
			t.Errorf("wrong Level; got %q, expected %q", gotMessage.Level, test.expLevel.String())
		}
	})

	t.Run("Msg", func(t *testing.T) {
		if gotMessage.Msg != test.expMsg {
			t.Errorf("wrong Msg; got %q, expected %q", gotMessage.Msg, test.expMsg)
		}
	})

	t.Run("Data", func(t *testing.T) {
		if maps.Equal(gotMessage.Data, test.expData) {
			return
		}

		// Show what's different
		if len(gotMessage.Data) != len(test.expData) {
			t.Errorf("wrong number of keys, got %d, test.expData %d", len(gotMessage.Data), len(test.expData))
		}

		for key := range test.expData {
			_, found := gotMessage.Data[key]
			if !found {
				t.Errorf("test.expData to find key %q", key)
			}
		}

		for key, gotVal := range gotMessage.Data {
			expVal, found := test.expData[key]
			if !found {
				t.Errorf("did not expect to find key %q", key)
			} else if gotVal != expVal {
				t.Errorf(
					"wrong value at key %q; got %#v (type %T), test.expData %#v (type %T)",
					key, gotVal, gotVal, expVal, expVal,
				)
			}
		}
	})
}

func TestLog(t *testing.T) {
	t.Run("not enabled", func(t *testing.T) {
		var callCount int
		lgr := StubLogger{
			enabledResp: func(ctx context.Context, lvl slog.Level) bool { return false },
			logAttrsResp: func(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr) {
				callCount++
			},
		}

		log(context.Background(), &lgr, slog.LevelError, "")

		if callCount > 0 {
			t.Errorf("did not expect to call LogAttrs, but called it %d times", callCount)
		}
	})

	t.Run("error input non-empty", func(t *testing.T) {
		var callCount int
		var gotAttrs []slog.Attr

		lgr := StubLogger{
			enabledResp: func(ctx context.Context, lvl slog.Level) bool { return true },
			logAttrsResp: func(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr) {
				callCount++

				gotAttrs = make([]slog.Attr, len(attrs))
				copy(gotAttrs, attrs)
			},
		}

		testError := errors.New("TEST ERROR")
		log(context.Background(), &lgr, slog.LevelError, "", slog.Any("error", testError))

		if callCount != 1 {
			t.Errorf("expected to call LogAttrs %d time, got %d", 1, callCount)
		}

		const expGroupName = "data"
		foundDataGroups := findMatchingGroups(t, expGroupName, gotAttrs...)
		if len(foundDataGroups) != 1 {
			t.Fatalf("expected to find 1 slog attr with a key %q and value of %s", expGroupName, slog.KindGroup.String())
		}

		foundError := slices.ContainsFunc(foundDataGroups, func(attr slog.Attr) bool {
			if !isMatchingGroup(t, expGroupName, attr) {
				return false
			}

			subAttrs := attr.Value.Any().([]slog.Attr)
			return slices.ContainsFunc(subAttrs, func(subAttr slog.Attr) bool {
				if subAttr.Key != "error" {
					return false
				}
				val, ok := subAttr.Value.Any().(error)
				if !ok {
					return false
				}
				return errors.Is(val, testError)
			})
		})

		if !foundError {
			t.Errorf("expected to find error %v among log attributes %v", testError, foundDataGroups)
		}
	})

	t.Run("message", func(t *testing.T) {
		tests := []struct {
			name     string
			inputMsg string
		}{
			{name: "output gets a prefix", inputMsg: "foo bar"},
			{name: "when input already has a prefix", inputMsg: "godfish: foo bar"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var gotMsg string
				lgr := StubLogger{
					enabledResp: func(ctx context.Context, lvl slog.Level) bool { return true },
					logAttrsResp: func(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr) {
						gotMsg = msg
					},
				}

				log(context.Background(), &lgr, slog.LevelError, test.inputMsg)

				const expected = "godfish: foo bar"
				if gotMsg != expected {
					t.Errorf("wrong message; got %q, expected %q", gotMsg, expected)
				}
			})
		}
	})

	t.Run("fields", func(t *testing.T) {
		var callCount int
		var gotAttrs []slog.Attr

		lgr := StubLogger{
			enabledResp: func(ctx context.Context, lvl slog.Level) bool { return true },
			logAttrsResp: func(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr) {
				callCount++

				gotAttrs = make([]slog.Attr, len(attrs))
				copy(gotAttrs, attrs)
			},
		}

		const keyA, keyB, keyC = "alpha", "bravo", "charlie"
		inputFields := []slog.Attr{slog.String(keyA, "aaa"), slog.Bool(keyB, true), slog.Int(keyC, 42)}
		log(context.Background(), &lgr, slog.LevelError, "", inputFields...)

		if callCount != 1 {
			t.Errorf("expected to call LogAttrs %d time, got %d", 1, callCount)
		}

		const expGroupName = "data"
		foundDataGroups := findMatchingGroups(t, expGroupName, gotAttrs...)
		if len(foundDataGroups) != 1 {
			t.Fatalf("expected to find 1 slog attr with a key %q and value of %s", expGroupName, slog.KindGroup.String())
		}

		subAttrs := foundDataGroups[0].Value.Any().([]slog.Attr)

		expAttrs := []slog.Attr{
			{Key: keyA, Value: slog.StringValue("aaa")},
			{Key: keyB, Value: slog.BoolValue(true)},
			{Key: keyC, Value: slog.IntValue(42)},
		}
		if len(subAttrs) != len(expAttrs) {
			t.Fatalf("wrong length of data group; got %d, expected %d", len(subAttrs), len(expAttrs))
		}
		for i, got := range subAttrs {
			exp := expAttrs[i]
			if got.Key != exp.Key {
				t.Errorf("wrong Key at attribute[%d]; got %q, expected %q", i, got.Key, exp.Key)
			}
			if !got.Value.Equal(exp.Value) {
				t.Errorf("wrong Value at attribute[%d]; got %v, expected %v", i, got.Value, exp.Value)
			}
		}
	})
}

func findMatchingGroups(t *testing.T, key string, gotAttrs ...slog.Attr) (out []slog.Attr) {
	t.Helper()

	for _, attr := range gotAttrs {
		if isMatchingGroup(t, key, attr) {
			out = append(out, attr)
		}
	}

	return
}

func isMatchingGroup(t *testing.T, key string, attr slog.Attr) bool {
	t.Helper()

	return attr.Key == key && attr.Value.Kind() == slog.KindGroup
}

type StubLogger struct {
	enabledResp  func(ctx context.Context, lvl slog.Level) bool
	logAttrsResp func(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr)
}

func (l *StubLogger) Enabled(ctx context.Context, lvl slog.Level) bool {
	if l.enabledResp == nil {
		panic("enabledResp is not defined")
	}
	return l.enabledResp(ctx, lvl)
}

func (l *StubLogger) LogAttrs(ctx context.Context, lvl slog.Level, msg string, attrs ...slog.Attr) {
	if l.logAttrsResp == nil {
		panic("logAttrsResp is not defined")
	}
	l.logAttrsResp(ctx, lvl, msg, attrs...)
}

// JSONMessage is the shape of 1 log entry. It's most useful for inspecting
// discrete pieces of data emitted with a JSON handler.
type JSONMessage struct {
	Time  string
	Level string
	Msg   string
	Data  map[string]any
}
