package misspell

import (
	"bytes"
	"regexp"
	"strings"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func inArray(haystack []string, needle string) bool {
	for _, word := range haystack {
		if needle == word {
			return true
		}
	}
	return false
}

var wordRegexp = regexp.MustCompile(`[a-zA-Z']+`)

/*
line1 and line2 are different
extract words from each line1

replace word -> newword
if word == new-word
  continue
if new-word in list of replacements
  continue
new word not original, and not in list of replacements
  some substring got mixed up.  UNdo
*/
func recheckLine(s string, rep *strings.Replacer, corrected map[string]bool) (string, []Diff) {
	diffs := []Diff{}
	out := ""
	first := 0
	redacted := RemovePath(StripURL(s))

	idx := wordRegexp.FindAllStringIndex(redacted, -1)
	for _, ab := range idx {
		word := s[ab[0]:ab[1]]
		newword := rep.Replace(word)
		if newword == word {
			// no replacement done
			continue
		}
		if corrected[strings.ToLower(newword)] {
			// word got corrected into something we know
			out += s[first:ab[0]] + newword
			first = ab[1]
			diffs = append(diffs, Diff{
				Original:  word,
				Corrected: newword,
				Column:    ab[0],
			})
			continue
		}
		// Word got corrected into something unknown. Ignore it
	}
	out += s[first:]
	return out, diffs
}

// Diff is datastructure showing what changed in a single line
type Diff struct {
	Filename  string
	Line      int
	Column    int
	Original  string
	Corrected string
}

// DiffLines produces a grep-like diff between two strings showing
// filename, linenum and change.  It is not meant to be a comprehensive diff.
func DiffLines(input, output string, r *strings.Replacer, c map[string]bool) (string, []Diff) {
	var changes []Diff

	// fast case -- no changes!
	if output == input {
		return output, changes
	}

	buf := bytes.Buffer{}
	buf.Grow(max(len(output), len(input)))

	// line by line to make nice output
	// This is horribly slow.
	outlines := strings.SplitAfter(output, "\n")
	inlines := strings.SplitAfter(input, "\n")
	for i := 0; i < len(inlines); i++ {
		if inlines[i] == outlines[i] {
			buf.WriteString(outlines[i])
			continue
		}
		newline, linediffs := recheckLine(inlines[i], r, c)
		buf.WriteString(newline)
		for _, d := range linediffs {
			d.Line = i + 1
			changes = append(changes, d)
		}
	}

	return buf.String(), changes
}

// Replacer is the main struct for spelling correction
type Replacer struct {
	Replacements []string
	Debug        bool
	engine       *strings.Replacer
	corrected    map[string]bool
}

// New creates a new default Replacer using the main rule list
func New() *Replacer {
	r := Replacer{
		Replacements: DictMain,
	}
	r.Compile()
	return &r
}

// RemoveRule deletes existings rules.
// TODO: make inplace to save memory
func (r *Replacer) RemoveRule(ignore []string) {
	newwords := make([]string, 0, len(r.Replacements))
	for i := 0; i < len(r.Replacements); i += 2 {
		if inArray(ignore, r.Replacements[i]) {
			continue
		}
		newwords = append(newwords, r.Replacements[i:i+2]...)
	}
	r.engine = nil
	r.Replacements = newwords
}

// AddRuleList appends new rules.
// Input is in the same form as Strings.Replacer: [ old1, new1, old2, new2, ....]
// Note: does not check for duplictes
func (r *Replacer) AddRuleList(additions []string) {
	r.engine = nil
	r.Replacements = append(r.Replacements, additions...)
}

// Compile compiles the rules.  Required before using the Replace functions
func (r *Replacer) Compile() {
	r.corrected = make(map[string]bool)
	for i := 1; i < len(r.Replacements); i += 2 {
		r.corrected[strings.ToLower(r.Replacements[i])] = true
	}
	r.engine = strings.NewReplacer(r.Replacements...)
}

// Replace make spelling corrects to the input string
func (r *Replacer) Replace(input string) (string, []Diff) {
	news := r.engine.Replace(input)
	news, changes := DiffLines(input, news, r.engine, r.corrected)
	return news, changes
}
