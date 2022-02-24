package internal_test

import (
	"testing"
	"time"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestParseVersion(t *testing.T) {
	type testCase struct {
		input  string
		expVal int64
		expErr bool
	}

	runTest := func(t *testing.T, test testCase) {
		got, err := internal.ParseVersion(test.input)
		if !test.expErr && err != nil {
			t.Fatal(err)
		} else if test.expErr && err == nil {
			t.Fatal("expected error but did not get one")
		} else if test.expErr && err != nil {
			return // ok, nothing more to test.
		}

		val := got.Value()
		if val != test.expVal {
			t.Errorf("wrong Value; got %d, expected %d", val, test.expVal)
		}
	}

	t.Run("regular timestamp layout", func(t *testing.T) {
		runTest(t, testCase{
			input:  "20060102030405",
			expVal: time.Date(2006, time.January, 2, 3, 4, 5, 0, time.UTC).Unix(),
		})
	})

	t.Run("timestamp is short", func(t *testing.T) {
		runTest(t, testCase{
			input:  "1234",
			expVal: time.Date(1234, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		})

	})

	t.Run("timestamp is too short", func(t *testing.T) {
		runTest(t, testCase{input: "123", expErr: true})
	})

	t.Run("timestamp is long", func(t *testing.T) {
		runTest(t, testCase{
			input:  "2006010203040567890",
			expVal: time.Date(2006, time.January, 2, 3, 4, 5, 0, time.UTC).Unix(),
		})
	})

	t.Run("unix timestamp", func(t *testing.T) {
		runTest(t, testCase{
			input:  "1574079194",
			expVal: time.Date(2019, time.November, 18, 12, 13, 14, 0, time.UTC).Unix(),
		})
	})
}
