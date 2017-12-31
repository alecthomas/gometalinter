package engine

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/gometalinter/config"
)

func TestEngine(t *testing.T) {
	engine, err := New(&config.Config{}, nil)
	require.NoError(t, err)
}
