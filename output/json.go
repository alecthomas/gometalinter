package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alecthomas/gometalinter/api"
)

// JSON streams a JSON array of issues to w.
func JSON(w io.Writer, issues chan *api.Issue) error {
	fmt.Fprintln(w, "[")
	count := 0
	for issue := range issues {
		if count != 0 {
			fmt.Fprintf(w, ",\n")
		}
		count++
		d, err := json.Marshal(issue)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "  %s", d)
	}
	fmt.Fprintf(w, "\n]\n")
	return nil
}
