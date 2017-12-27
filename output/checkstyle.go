package output

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/alecthomas/gometalinter/api"
)

type checkstyleOutput struct {
	XMLName xml.Name          `xml:"checkstyle"`
	Version string            `xml:"version,attr"`
	Files   []*checkstyleFile `xml:"file"`
}

type checkstyleFile struct {
	Name   string             `xml:"name,attr"`
	Errors []*checkstyleError `xml:"error"`
}

type checkstyleError struct {
	Column   int    `xml:"column,attr"`
	Line     int    `xml:"line,attr"`
	Message  string `xml:"message,attr"`
	Severity string `xml:"severity,attr"`
	Source   string `xml:"source,attr"`
}

// Checkstyle writes issues in checkstyle XML format.
func Checkstyle(w io.Writer, issues chan *api.Issue) error {
	var lastFile *checkstyleFile
	out := checkstyleOutput{
		Version: "5.0",
	}
	for issue := range issues {
		if lastFile != nil && lastFile.Name != issue.Path {
			out.Files = append(out.Files, lastFile)
			lastFile = nil
		}
		if lastFile == nil {
			lastFile = &checkstyleFile{
				Name: issue.Path,
			}
		}

		lastFile.Errors = append(lastFile.Errors, &checkstyleError{
			Column:   issue.Col,
			Line:     issue.Line,
			Message:  issue.Message,
			Severity: string(issue.Severity),
			Source:   issue.Linter,
		})
	}
	if lastFile != nil {
		out.Files = append(out.Files, lastFile)
	}
	fmt.Fprint(w, xml.Header)
	err := xml.NewEncoder(w).Encode(&out)
	if err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}
