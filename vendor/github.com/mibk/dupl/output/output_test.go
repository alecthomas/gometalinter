package output

import "testing"

func TestToWhitespace(t *testing.T) {
	testCases := []struct {
		in     string
		expect string
	}{
		{"\t   ", "\t   "},
		{"\tčřď", "\t   "},
		{"  \ta", "  \t "},
	}

	for _, tc := range testCases {
		actual := toWhitespace([]byte(tc.in))
		if tc.expect != string(actual) {
			t.Errorf("got '%s', want '%s'", actual, tc.expect)
		}
	}
}

func TestDeindent(t *testing.T) {
	testCases := []struct {
		in     string
		expect string
	}{
		{"\t$\n\t\t$\n\t$", "$\n\t$\n$"},
		{"\t$\r\n\t\t$\r\n\t$", "$\r\n\t$\r\n$"},
		{"\t$\n\t\t$\n", "$\n\t$\n"},
		{"\t$\n\n\t\t$", "$\n\n\t$"},
	}
	for _, tc := range testCases {
		actual := deindent([]byte(tc.in))
		if tc.expect != string(actual) {
			t.Errorf("got '%s', want '%s'", actual, tc.expect)
		}
	}
}
