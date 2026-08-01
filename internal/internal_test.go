package internal_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rafaelespinoza/godfish/internal"
)

func TestIsInvalidDataError(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		exp  bool
	}{
		{name: "nil", err: nil, exp: false},
		{
			name: "ErrDataInvalid",
			err:  internal.ErrDataInvalid,
			exp:  true,
		},
		{
			name: "wraps ErrDataInvalid",
			err:  fmt.Errorf("%w: test", internal.ErrDataInvalid),
			exp:  true,
		},
		{
			name: "implements interface but set to false",
			err:  invalidDataErr{error: errors.New("oof"), invalid: false},
			exp:  false,
		},
		{
			name: "implements interface and set to true",
			err:  invalidDataErr{error: errors.New("oof"), invalid: true},
			exp:  true,
		},
		{
			name: "wraps implemented interface but set to false",
			err:  fmt.Errorf("%w, bar", invalidDataErr{error: errors.New("oof"), invalid: false}),
			exp:  false,
		},
		{
			name: "wraps implemented interface and set to true",
			err:  fmt.Errorf("%w, bar", invalidDataErr{error: errors.New("oof"), invalid: true}),
			exp:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := internal.IsInvalidDataError(test.err)
			if got != test.exp {
				t.Errorf("got %t, expected %t", got, test.exp)
			}
		})
	}
}

// invalidDataErr simulates an error that implements the behavior targeted by
// [internal.IsInvalidDataError].
type invalidDataErr struct {
	error
	invalid bool
}

func (i invalidDataErr) Invalid() bool { return i.invalid }
