package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// DefaultIssueFormat used to print an issue
const DefaultIssueFormat = "{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"

// Severity of linter message
type Severity string

// Linter message severity levels.
const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type issuePath struct {
	currentDir string
	path       string
}

func (i issuePath) String() string {
	fallback := i.path
	root := resolvePath(i.currentDir)
	path := resolvePath(i.path)

	var err error
	path, err = filepath.Rel(root, path)
	if err != nil {
		warning("failed to make %s a relative path: %s", fallback, err)
		return fallback
	}
	return path
}

func (i issuePath) Abs() string {
	return resolvePath(i.path)
}

func resolvePath(path string) string {
	var err error
	fallback := path
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			warning("failed to make %s an absolute path: %s", fallback, err)
			return fallback
		}
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		warning("failed to resolve symlinks in %s: %s", fallback, err)
		return fallback
	}
	return path
}

type Issue struct {
	Linter     string    `json:"linter"`
	Severity   Severity  `json:"severity"`
	Path       issuePath `json:"path"`
	Line       int       `json:"line"`
	Col        int       `json:"col"`
	Message    string    `json:"message"`
	formatTmpl *template.Template
}

// NewIssue returns a new issue. Returns an error if formatTmpl is not a valid
// template for an Issue.
func NewIssue(linter string, formatTmpl *template.Template) (*Issue, error) {
	issue := &Issue{
		Line:       1,
		Severity:   Warning,
		Linter:     linter,
		formatTmpl: formatTmpl,
	}
	err := formatTmpl.Execute(ioutil.Discard, issue)
	return issue, err
}

func (i *Issue) String() string {
	if i.formatTmpl == nil {
		col := ""
		if i.Col != 0 {
			col = fmt.Sprintf("%d", i.Col)
		}
		return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", strings.TrimSpace(i.Path.String()), i.Line, col, i.Severity, strings.TrimSpace(i.Message), i.Linter)
	}
	buf := new(bytes.Buffer)
	_ = i.formatTmpl.Execute(buf, i)
	return buf.String()
}

type sortedIssues struct {
	issues []*Issue
	order  []string
}

func (s *sortedIssues) Len() int      { return len(s.issues) }
func (s *sortedIssues) Swap(i, j int) { s.issues[i], s.issues[j] = s.issues[j], s.issues[i] }

func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	return CompareIssue(*l, *r, s.order)
}

// CompareIssue two Issues and return true if left should sort before right
// nolint: gocyclo
func CompareIssue(l, r Issue, order []string) bool {
	for _, key := range order {
		switch {
		case key == "path" && l.Path != r.Path:
			return l.Path.String() < r.Path.String()
		case key == "line" && l.Line != r.Line:
			return l.Line < r.Line
		case key == "column" && l.Col != r.Col:
			return l.Col < r.Col
		case key == "severity" && l.Severity != r.Severity:
			return l.Severity < r.Severity
		case key == "message" && l.Message != r.Message:
			return l.Message < r.Message
		case key == "linter" && l.Linter != r.Linter:
			return l.Linter < r.Linter
		}
	}
	return true
}

// SortIssueChan reads issues from one channel, sorts them, and returns them to another
// channel
func SortIssueChan(issues chan *Issue, order []string) chan *Issue {
	out := make(chan *Issue, 1000000)
	sorted := &sortedIssues{
		issues: []*Issue{},
		order:  order,
	}
	go func() {
		for issue := range issues {
			sorted.issues = append(sorted.issues, issue)
		}
		sort.Sort(sorted)
		for _, issue := range sorted.issues {
			out <- issue
		}
		close(out)
	}()
	return out
}
