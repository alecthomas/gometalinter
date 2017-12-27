package pipeline

import (
	"sort"
	"testing"

	"github.com/alecthomas/gometalinter/api"
	"github.com/stretchr/testify/require"
)

func TestSortedIssues(t *testing.T) {
	actual := []*api.Issue{
		{Path: "b.go", Line: 5, Col: 1},
		{Path: "a.go", Line: 3, Col: 2},
		{Path: "b.go", Line: 1, Col: 3},
		{Path: "a.go", Line: 1, Col: 4},
	}
	issues := &sortedIssues{
		issues: actual,
		order:  []string{"path", "line", "column"},
	}
	sort.Sort(issues)
	expected := []*api.Issue{
		{Path: "a.go", Line: 1, Col: 4},
		{Path: "a.go", Line: 3, Col: 2},
		{Path: "b.go", Line: 1, Col: 3},
		{Path: "b.go", Line: 5, Col: 1},
	}
	require.Equal(t, expected, actual)
}
