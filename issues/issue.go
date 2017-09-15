package issues

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"text/template"
)

// DefaultFormat used to print an issue
const DefaultFormat = "{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"

// Severity of linter message
type Severity string

// Linter message severity levels.
const ( // nolint: deadcode
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Issue struct {
	Linter     string   `json:"linter"`
	Severity   Severity `json:"severity"`
	Path       string   `json:"path"`
	Line       int      `json:"line"`
	Col        int      `json:"col"`
	Message    string   `json:"message"`
	formatTmpl *template.Template
}

// NewIssue returns a new issue. Returns an error if formatTmpl is not a valid
// template for an Issue.
func NewIssue(linter string, formatTmpl *template.Template) (*Issue, error) {
	if formatTmpl == nil {
		var err error
		formatTmpl, err = template.New("output").Parse(DefaultFormat)
		if err != nil {
			return nil, err
		}
	}
	issue := &Issue{
		Line:       1,
		Linter:     linter,
		formatTmpl: formatTmpl,
	}
	err := formatTmpl.Execute(ioutil.Discard, issue)
	return issue, err
}

func (i *Issue) String() string {
	col := ""
	if i.Col != 0 {
		col = fmt.Sprintf("%d", i.Col)
	}
	if i.formatTmpl == nil {
		return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", strings.TrimSpace(i.Path), i.Line, col, i.Severity, strings.TrimSpace(i.Message), i.Linter)
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

// nolint: gocyclo
func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	for _, key := range s.order {
		switch key {
		case "path":
			if l.Path > r.Path {
				return false
			}
		case "line":
			if l.Line > r.Line {
				return false
			}
		case "column":
			if l.Col > r.Col {
				return false
			}
		case "severity":
			if l.Severity > r.Severity {
				return false
			}
		case "message":
			if l.Message > r.Message {
				return false
			}
		case "linter":
			if l.Linter > r.Linter {
				return false
			}
		}
	}
	return true
}

// SortChan reads issues from one channel, sorts them, and returns them to another
// channel
func SortChan(issues chan *Issue, order []string) chan *Issue {
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

// Sort a slice of issues
func Sort(issues []*Issue, order []string) {
	sort.Sort(&sortedIssues{issues: issues, order: order})
}
