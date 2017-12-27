package pipeline

import (
	"sort"
	"strings"

	"github.com/alecthomas/gometalinter/api"
)

type issueKey struct {
	path      string
	line, col int
	message   string
}

type multiIssue struct {
	*api.Issue
	linterNames []string
}

// Aggregate reads issues from a channel, aggregates issues which have
// the same file, line, vol, and message, and returns aggregated issues on
// a new channel.
func Aggregate(issues chan *api.Issue) chan *api.Issue {
	out := make(chan *api.Issue, 1)
	issueMap := make(map[issueKey]*multiIssue)
	go func() {
		defer close(out)
		for issue := range issues {
			key := issueKey{
				path:    issue.Path,
				line:    issue.Line,
				col:     issue.Col,
				message: issue.Message,
			}
			if existing, ok := issueMap[key]; ok {
				existing.linterNames = append(existing.linterNames, issue.Linter)
			} else {
				issueMap[key] = &multiIssue{
					Issue:       issue,
					linterNames: []string{issue.Linter},
				}
			}
		}
		for _, multi := range issueMap {
			issue := multi.Issue
			sort.Strings(multi.linterNames)
			issue.Linter = strings.Join(multi.linterNames, ", ")
			out <- issue
		}
	}()
	return out
}
