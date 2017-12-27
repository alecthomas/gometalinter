package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareOrderWithMessage(t *testing.T) {
	order := []string{"path", "line", "column", "message"}
	issueM := Issue{Path: "file.go", Message: "message"}
	issueU := Issue{Path: "file.go", Message: "unknown"}

	assert.True(t, CompareIssue(issueM, issueU, order))
	assert.False(t, CompareIssue(issueU, issueM, order))
}
