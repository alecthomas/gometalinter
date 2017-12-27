package output

import (
	"fmt"
	"io"
	"text/template"

	"github.com/alecthomas/gometalinter/api"
)

// Text writes a textual representation of all issues to w.
func Text(w io.Writer, template *template.Template, issues chan *api.Issue) error {
	for issue := range issues {
		if template == nil {
			fmt.Fprintln(w, issue.String())
		} else {
			err := template.Execute(w, issue)
			if err != nil {
				return err
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}
