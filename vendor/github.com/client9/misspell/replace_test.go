package misspell

import (
	"strings"
	"testing"
)

func TestReplaceIgnore(t *testing.T) {
	cases := []struct {
		ignore string
		text   string
	}{
		{"knwo,gae", "https://github.com/Unknwon, github.com/hnakamur/gaesessions"},
	}
	for line, tt := range cases {
		r := New()
		r.RemoveRule(strings.Split(tt.ignore, ","))
		r.Compile()
		got := r.Replace(tt.text)
		if got != tt.text {
			t.Errorf("%d: Replace files want %q got %q", line, tt.text, got)
		}
	}
}

func TestReplaceLocale(t *testing.T) {
	cases := []struct {
		orig string
		want string
	}{
		{"The colours are pretty", "The colors are pretty"},
	}

	r := New()
	r.AddRuleList(DictAmerican)
	r.Compile()
	for line, tt := range cases {
		got := r.Replace(tt.orig)
		if got != tt.want {
			t.Errorf("%d: ReplaceLocale want %q got %q", line, tt.orig, got)
		}
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		orig string
		want string
	}{
		{"I live in Amercia", "I live in America"},
		{"grill brocoli now", "grill brocolli now"},
		{"There is a zeebra", "There is a zebra"},
		{"foo other bar", "foo other bar"},
		{"ten fiels", "ten fields"},
		{"Closeing Time", "Closing Time"},
		{"closeing Time", "closing Time"},
		{" TOOD: foobar", " TODO: foobar"},
		{" preceed ", " precede "},
		{" preceeding ", " preceding "},
	}
	r := New()
	for line, tt := range cases {
		got := r.Replace(tt.orig)
		if got != tt.want {
			t.Errorf("%d: Replace files want %q got %q", line, tt.orig, got)
		}
	}
}

func TestReplaceGo(t *testing.T) {
	cases := []struct {
		orig string
		want string
	}{
		{
			orig: `
// I am a zeebra
var foo 10
`,
			want: `
// I am a zebra
var foo 10
`,
		},
		{
			orig: `
var foo 10
// I am a zeebra`,
			want: `
var foo 10
// I am a zebra`,
		},
		{
			orig: `
// I am a zeebra
var foo int
/* multiline
 * zeebra
 */
`,
			want: `
// I am a zebra
var foo int
/* multiline
 * zebra
 */
`,
		},
	}

	r := New()
	for casenum, tt := range cases {
		got := r.ReplaceGo(tt.orig)
		if got != tt.want {
			t.Errorf("%d: %q got converted to %q", casenum, tt, got)
		}
	}
}

func TestCommonPrefixWordLength(t *testing.T) {
	cases := []struct {
		a   string
		b   string
		col int
	}{
		{"", "", 0},
		{"1", "1", 1},
		{"11", "11", 2},
		{"11", "22", 0},
		{"1", "22", 0},
		{"22", "1", 0},
		{"1", "11", 1},
		{"11", "1", 1},

		{"start ", "start ", 6},
		{"start 11", "start 22", 6},
		{"start 11", "start 123", 6},
		{"start 123", "start 11", 6},

		{"start word 123", "start word 11", 11},
		{"start word 123", "", 0},
		{"", "start word 123", 0},
	}

	for casenum, tt := range cases {
		col := commonPrefixWordLength(tt.a, tt.b)
		if col != tt.col {
			t.Errorf("%d: with %q, %q want prefix length of %d, got %d", casenum, tt.a, tt.b, tt.col, col)
		}
	}
}

func TestCommonSuffixWordLength(t *testing.T) {
	cases := []struct {
		a   string
		b   string
		col int
	}{
		{"", "", 0},
		{"1", "1", 1},
		{"11", "11", 2},
		{"11", "22", 0},
		{"1", "22", 0},
		{"22", "1", 0},
		{"1", "11", 1},
		{"11", "1", 1},

		{"start end", "start end", len("start end")},
		{"abc end", "start end", len(" end")},
		{"start end", "abc end", len(" end")},
		{"start middle end", "foo middle end", len(" middle end")},
		{"start middle end ", "foo middle end ", len(" middle end ")},
	}

	for casenum, tt := range cases {
		col := commonSuffixWordLength(tt.a, tt.b)
		if col != tt.col {
			t.Errorf("%d: with %q, %q want suffix length of %d, got %d", casenum, tt.a, tt.b, tt.col, col)
		}
	}
}

func TestLineChange(t *testing.T) {
	cases := []struct {
		line1 string
		line2 string
		word1 string
		word2 string
		col   int
	}{
		{"zeebra end", "zebra end", "zeebra", "zebra", 0},
		{"start zeebra", "start zebra", "zeebra", "zebra", len("start ")},
		{"start zeebra end", "start zebra end", "zeebra", "zebra", len("start ")},
		{
			"rows withing the",
			"rows within the",
			"withing",
			"within",
			5,
		},
		{
			"foo yz the",
			"foo xyz the",
			"yz",
			"xyz",
			4,
		},
	}

	for casenum, tt := range cases {
		word1, word2, col := corrected(tt.line1, tt.line2)
		if word1 != tt.word1 || word2 != tt.word2 || col != tt.col {
			t.Errorf("%d (%q,%q,%d) != (%q,%q,%d)", casenum, tt.word1, tt.word2, tt.col, word1, word2, col)
		}
	}
}

func TestDiff(t *testing.T) {
	var out string
	var want string
	var diffs []Diff

	// not so nice doing a table driven test here.
	out, diffs = DiffLines("", "")
	if out != "" || len(diffs) != 0 {
		t.Errorf("DiffLines couldn't handle empty inputs: %q %d", out, len(diffs))
	}

	want = "nothing"
	out, diffs = DiffLines("nothing", want)
	if out != want || len(diffs) != 0 {
		t.Errorf("DiffLines couldn't handle same inputs: %q %d", out, len(diffs))
	}

	want = "nothing\n"
	out, diffs = DiffLines("nothing\n", want)
	if out != want || len(diffs) != 0 {
		t.Errorf("DiffLines couldn't handle same inputs with newlines")
	}

	// Normal correction case
	want = "nothing\nzebra\nnothing"
	out, diffs = DiffLines("nothing\nzeebra\nnothing", want)
	if out != want {
		t.Errorf("Want %q got %q", want, out)
	}
	if len(diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Line != 2 || diffs[0].Column != 0 {
		t.Errorf("Expected correction to be on line 2, column 0 - got %d, %d", diffs[0].Line, diffs[0].Column)
	}
	if diffs[0].Original != "zeebra" || diffs[0].Corrected != "zebra" {
		t.Errorf("Expected (%q,%q) got (%q,%q)", "zeebra", "zebra", diffs[0].Original, diffs[0].Corrected)
	}

	// Undo case correction case.. zeebra is part of a big chunk of text
	//  don't make correction
	orig := "nothing\nxxxxxxxxxxxxxxxxxxzeebraxxxxxxxxxxxxxxxxxxxxxxx\nnothing"
	corr := "nothing\nxxxxxxxxxxxxxxxxxxzebraxxxxxxxxxxxxxxxxxxxxxxx\nnothing"
	out, diffs = DiffLines(orig, corr)
	if out != orig {
		t.Errorf("Want %q got %q", orig, out)
	}
	if len(diffs) != 0 {
		t.Errorf("Expected 0 diff, got %d", len(diffs))
	}

	// URL case
	orig = "http://github.com"
	corr = "http://changed.com"
	out, diffs = DiffLines(orig, corr)
	if out != orig {
		t.Errorf("Want %q got %q", orig, out)
	}
	if len(diffs) != 0 {
		t.Errorf("Expected 0 diff, got %d", len(diffs))
	}

}
