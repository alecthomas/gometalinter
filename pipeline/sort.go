package pipeline

import (
	"sort"

	"github.com/alecthomas/gometalinter/api"
)

// Sort reads issues from one channel, sorts them, and returns them to another
// channel
func Sort(issues chan *api.Issue, order []string) chan *api.Issue {
	out := make(chan *api.Issue, 1)
	sorted := &sortedIssues{
		issues: []*api.Issue{},
		order:  order,
	}
	go func() {
		defer close(out)
		for issue := range issues {
			sorted.issues = append(sorted.issues, issue)
		}
		sort.Sort(sorted)
		for _, issue := range sorted.issues {
			out <- issue
		}
	}()
	return out
}

type sortedIssues struct {
	issues []*api.Issue
	order  []string
}

func (s *sortedIssues) Len() int      { return len(s.issues) }
func (s *sortedIssues) Swap(i, j int) { s.issues[i], s.issues[j] = s.issues[j], s.issues[i] }

func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	return api.CompareIssue(*l, *r, s.order)
}
