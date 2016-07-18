package misspell

import (
	"bytes"
	"log"
	"strings"
	"text/scanner"
)

// Diff is datastructure showing what changed in a single line
type Diff struct {
	Filename  string
	Line      int
	Column    int
	Original  string
	Corrected string
}

func isWhite(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\t' || ch == '\r'
}

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

// commonPrefixWordLength finds the common prefix then backs up until
// it finds whitespace.  Given "foo bar" and "foo bat", this will
// return "foo " NOT "foo ba"
func commonPrefixWordLength(a, b string) int {
	// re-order so len(a) <= len(b) always
	if len(a) > len(b) {
		b, a = a, b
	}
	lastWhite := 0
	for i := 0; i < len(a); i++ {
		ch := a[i]
		if ch != b[i] {
			if lastWhite == 0 {
				return 0
			}
			return min(lastWhite+1, len(a))
		}
		if isWhite(ch) {
			lastWhite = i
		}

	}
	return len(a)
}

// commonSuffixWordLength
func commonSuffixWordLength(a, b string) int {
	alen, blen := len(a), len(b)
	n := min(alen, blen)
	lastWhite := 0
	for i := 0; i < n; i++ {
		ch := a[alen-i-1]
		if ch != b[blen-i-1] {
			if lastWhite == 0 && !isWhite(a[alen-1]) {
				return 0
			}
			return min(lastWhite+1, n)
		}
		if isWhite(ch) {
			lastWhite = i
		}
	}
	return n
}

// Return the misspelled word, the correction and the column position
//
func corrected(instr, outstr string) (string, string, int) {
	prefixLen := commonPrefixWordLength(instr, outstr)
	suffixLen := commonSuffixWordLength(instr, outstr)
	orig := instr[prefixLen : len(instr)-suffixLen]
	corr := outstr[prefixLen : len(outstr)-suffixLen]
	return orig, corr, prefixLen
}

// shouldUndo checks if a corrected string should be kept as original
func shouldUndo(s string) bool {
	// this is some blob of text that has no spaces for 20 characters!
	// Smells like programming or some base64 mess
	if len(s) > 20 {
		return true
	}

	// perhaps a URL
	if strings.Contains(s, "/") {
		return true
	}

	return false
}

// DiffLines produces a grep-like diff between two strings showing
// filename, linenum and change.  It is not meant to be a comprehensive diff.
func DiffLines(input, output string) (string, []Diff) {
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
		s1, s2, col := corrected(inlines[i], outlines[i])
		if shouldUndo(s1) {
			buf.WriteString(inlines[i])
			continue
		}

		diff := Diff{
			Line:      i + 1, // lines start at 1
			Column:    col,   // columns start at 0
			Original:  s1,
			Corrected: s2,
		}
		changes = append(changes, diff)
		buf.WriteString(outlines[i])
	}

	return buf.String(), changes
}

func inArray(haystack []string, needle string) bool {
	for _, word := range haystack {
		if needle == word {
			return true
		}
	}
	return false
}

// Replacer is the main struct for spelling correction
type Replacer struct {
	Replacements []string
	Debug        bool
	engine       *strings.Replacer
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
	r.engine = strings.NewReplacer(r.Replacements...)
}

// Replace make spelling corrects to the input string
func (r *Replacer) Replace(input string) string {
	if r.Debug {
		r.ReplaceDebug(input)
	}
	return r.engine.Replace(input)
}

// ReplaceDebug logs exactly what was matched and replaced for use
// in debugging
func (r *Replacer) ReplaceDebug(input string) {
	for linenum, line := range strings.Split(input, "\n") {
		for i := 0; i < len(r.Replacements); i += 2 {
			idx := strings.Index(line, r.Replacements[i])
			if idx != -1 {
				left := max(0, idx-10)
				right := min(idx+len(r.Replacements[i])+10, len(line))
				snippet := strings.TrimSpace(line[left:right])
				log.Printf("line %d: Found %q in %q  (%q)",
					linenum+1, r.Replacements[i], snippet, r.Replacements[i+1])
			}
		}
	}
}

// ReplaceGo is a specialized routine for correcting Golang source
// files.  Currently only checks comments, not identifiers for
// spelling.
//
// Other items:
//   - check strings, but need to ignore
//      * import "statements" blocks
//      * import ( "blocks" )
//   - skip first comment (line 0) if build comment
//
func (r *Replacer) ReplaceGo(input string) string {
	var s scanner.Scanner
	s.Init(strings.NewReader(input))
	s.Mode = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments
	lastPos := 0
	output := ""
	for {

		switch s.Scan() {
		case scanner.Comment:
			origComment := s.TokenText()
			newComment := r.Replace(origComment)

			if origComment != newComment {
				// s.Pos().Offset is the end of the current token
				//  subtract len(origComment) to get the start of token
				offset := s.Pos().Offset
				output = output + input[lastPos:offset-len(origComment)] + newComment
				lastPos = offset
			}
		case scanner.EOF:
			// no changes, no copies
			if lastPos == 0 {
				return input
			}
			if lastPos >= len(input) {
				return output
			}

			return output + input[lastPos:]
		}
	}
}
