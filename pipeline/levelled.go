package pipeline

import (
	"github.com/alecthomas/gometalinter/api"
)

// Levelled shows only issues at or above the given severity level.
func Levelled(issues chan *api.Issue, severity api.Severity) chan *api.Issue {
	return Filter(issues, func(issue *api.Issue) bool {
		return !issue.Severity.Less(severity)
	})
}
