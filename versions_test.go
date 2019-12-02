package godfish

import "testing"

func TestListVersionsToApply(t *testing.T) {
	tests := []struct {
		direction   Direction
		applied     []string
		available   []string
		expectedOut []string
		expectError bool
	}{
		{
			direction:   DirForward,
			applied:     []string{"1234", "5678"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{},
		},
		{
			direction:   DirForward,
			applied:     []string{"1234"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"5678"},
		},
		{
			direction:   DirForward,
			applied:     []string{},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234", "5678"},
		},
		{
			direction:   DirReverse,
			applied:     []string{"1234", "5678"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234", "5678"},
		},
		{
			direction:   DirReverse,
			applied:     []string{"1234"},
			available:   []string{"1234", "5678"},
			expectedOut: []string{"1234"},
		},
		{
			direction:   DirReverse,
			applied:     []string{},
			available:   []string{"1234", "5678"},
			expectedOut: []string{},
		},
		{
			direction:   DirForward,
			applied:     []string{},
			available:   []string{},
			expectedOut: []string{},
		},
		{
			direction:   DirReverse,
			applied:     []string{},
			available:   []string{},
			expectedOut: []string{},
		},
		{
			applied:     []string{},
			available:   []string{},
			expectError: true,
		},
	}

	for i, test := range tests {
		actual, err := listVersionsToApply(
			test.direction,
			test.applied,
			test.available,
		)
		gotError := err != nil
		if gotError && !test.expectError {
			t.Errorf("test %d; got error %v but did not expect one", i, err)
			continue
		} else if !gotError && test.expectError {
			t.Errorf("test %d; did not get error but did expect one", i)
			continue
		}
		if len(actual) != len(test.expectedOut) {
			t.Errorf(
				"test %d; got wrong output length %d, expected length to be %d",
				i, len(actual), len(test.expectedOut),
			)
			continue
		}
		for j, version := range actual {
			if version != test.expectedOut[j] {
				t.Errorf(
					"test [%d][%d]; got version %q but expected %q",
					i, j, version, test.expectedOut[j],
				)
			}
		}
	}
}
