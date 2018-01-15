// Package config reads the gometalinter TOML configuration file.
//
// Here's an example configuration file:
//
//     output = "json"
//     deadline = "30s"
//
//     [linter.misspell]
//     locale = "UK"
//     ignore = ["color", "aluminum"]
//
//     [define.test]
//     name = "test"
//     command = "go test"
//     pattern = "^--- FAIL: .*$\s+(?P<path>.*?\.go):(?P<line>\d+): (?P<message>.*)$"
//     partition = "packages"
//     severity = "error"
//
// Top-level keys are for configuring gometalinter itself, [linter.<linter>] sections configure individual linters and
// [define.<linter>] defines extra external linters.
package config
