package main

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestSortedIssues(t *testing.T) {
	actual := []*Issue{
		{Path: newIssuePath("", "b.go"), Line: 5, Col: 1},
		{Path: newIssuePath("", "a.go"), Line: 3, Col: 2},
		{Path: newIssuePath("", "b.go"), Line: 1, Col: 3},
		{Path: newIssuePath("", "a.go"), Line: 1, Col: 4},
	}
	issues := &sortedIssues{
		issues: actual,
		order:  []string{"path", "line", "column"},
	}
	sort.Sort(issues)
	expected := []*Issue{
		{Path: newIssuePath("", "a.go"), Line: 1, Col: 4},
		{Path: newIssuePath("", "a.go"), Line: 3, Col: 2},
		{Path: newIssuePath("", "b.go"), Line: 1, Col: 3},
		{Path: newIssuePath("", "b.go"), Line: 5, Col: 1},
	}
	assert.Assert(t, is.Compare(expected, actual, cmpIssue))
}

func TestCompareOrderWithMessage(t *testing.T) {
	order := []string{"path", "line", "column", "message"}
	issueM := Issue{Path: newIssuePath("", "file.go"), Message: "message"}
	issueU := Issue{Path: newIssuePath("", "file.go"), Message: "unknown"}

	assert.Check(t, CompareIssue(issueM, issueU, order))
	assert.Check(t, !CompareIssue(issueU, issueM, order))
}

var cmpIssue = cmp.AllowUnexported(Issue{}, IssuePath{})
