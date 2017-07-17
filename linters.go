package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

type Linter struct {
	Name             string
	Command          string
	Pattern          string
	InstallFrom      string
	SeverityOverride Severity
	MessageOverride  string

	regex             *regexp.Regexp
	partitionStrategy partitionStrategy
}

func (l *Linter) String() string {
	return l.Name
}

var predefinedPatterns = map[string]string{
	"PATH:LINE:COL:MESSAGE": `^(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
	"PATH:LINE:MESSAGE":     `^(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
}

func LinterFromName(name string) *Linter {
	s := linterDefinitions[name]
	parts := strings.SplitN(s, ":", 2)
	if len(parts) < 2 {
		kingpin.Fatalf("invalid linter: %q", name)
	}

	pattern := parts[1]
	if p, ok := predefinedPatterns[pattern]; ok {
		pattern = p
	}
	re, err := regexp.Compile("(?m:" + pattern + ")")
	kingpin.FatalIfError(err, "invalid regex for %q", name)
	return &Linter{
		Name:              name,
		Command:           s[0:strings.Index(s, ":")],
		Pattern:           pattern,
		InstallFrom:       installMap[name],
		SeverityOverride:  Severity(config.Severity[name]),
		MessageOverride:   config.MessageOverride[name],
		regex:             re,
		partitionStrategy: getPartitionStrategy(name),
	}
}

func makeInstallCommand(linters ...string) []string {
	cmd := []string{"get"}
	if config.VendoredLinters {
		cmd = []string{"install"}
	} else {
		if config.Update {
			cmd = append(cmd, "-u")
		}
		if config.Force {
			cmd = append(cmd, "-f")
		}
		if config.DownloadOnly {
			cmd = append(cmd, "-d")
		}
	}
	if config.Debug {
		cmd = append(cmd, "-v")
	}
	cmd = append(cmd, linters...)
	return cmd
}

func installLintersWithOneCommand(targets []string) error {
	cmd := makeInstallCommand(targets...)
	debug("go %s", strings.Join(cmd, " "))
	c := exec.Command("go", cmd...) // nolint: gas
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func installLintersIndividually(targets []string) {
	failed := []string{}
	for _, target := range targets {
		cmd := makeInstallCommand(target)
		debug("go %s", strings.Join(cmd, " "))
		c := exec.Command("go", cmd...) // nolint: gas
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			warning("failed to install %s: %s", target, err)
			failed = append(failed, target)
		}
	}
	if len(failed) > 0 {
		kingpin.Fatalf("failed to install the following linters: %s", strings.Join(failed, ", "))
	}
}

func installLinters() {
	names := make([]string, 0, len(installMap))
	targets := make([]string, 0, len(installMap))
	for name, target := range installMap {
		names = append(names, name)
		targets = append(targets, target)
	}
	namesStr := strings.Join(names, "\n  ")
	if config.DownloadOnly {
		fmt.Printf("Downloading:\n  %s\n", namesStr)
	} else {
		fmt.Printf("Installing:\n  %s\n", namesStr)
	}
	err := installLintersWithOneCommand(targets)
	if err == nil {
		return
	}
	warning("failed to install one or more linters: %s (installing individually)", err)
	installLintersIndividually(targets)
}
