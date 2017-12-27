package pipeline

import (
	"github.com/alecthomas/gometalinter/api"
)

func Status(issues chan *api.Issue) (chan int, chan *api.Issue) {
	status := make(chan int, 1)
	out := make(chan *api.Issue, 1)
	go func() {
		defer close(out)
		defer close(status)
		code := 0
		for issue := range issues {
			if issue.Severity == api.Warning {
				code |= 1
			} else if issue.Severity == api.Error {
				code |= 2
			}
			out <- issue
		}
		status <- code
	}()
	return status, out
}
