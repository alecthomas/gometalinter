package pipeline

import "github.com/alecthomas/gometalinter/api"

func Filter(issues chan *api.Issue, filter func(issue *api.Issue) bool) chan *api.Issue {
	out := make(chan *api.Issue, 1)
	go func() {
		defer close(out)
		for issue := range issues {
			if filter(issue) {
				out <- issue
			}
		}
	}()
	return out
}
