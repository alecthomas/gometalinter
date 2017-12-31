package util

import (
	"fmt"
	"os"
)

// Debugging enables or disables debug logging.
func Debugging(ok bool) {
	if ok {
		Debug = enabledDebug
	} else {
		Debug = noopDebug
	}
}

type DebugFunction func(format string, args ...interface{})

// Debug writes a debug message to stderr if Debugging(true).
var Debug = noopDebug

func enabledDebug(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
}

func noopDebug(format string, args ...interface{}) {}

// NamespacedDebug writes
func NamespacedDebug(prefix string) DebugFunction {
	return func(format string, args ...interface{}) {
		Debug(prefix+format, args...)
	}
}

// Warning writes a warning message to stderr.
func Warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARNING: "+format+"\n", args...)
}
