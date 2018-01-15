package external

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"

	"github.com/alecthomas/gometalinter/api"
	"github.com/alecthomas/gometalinter/config"
	. "github.com/alecthomas/gometalinter/util" // nolint
	"github.com/alecthomas/kingpin"
)

// ExecuteLinter runs linter with the given arguments.
//
// "args" is expected to be the concatenation of the result of parseCommand() with one of the partitioning functions.
func ExecuteLinter(ctx api.Context, linter *Linter, args []string) (*api.Issue, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing linter command")
	}
	start := time.Now()
	ctx.Debug("executing %s", strings.Join(args, " "))
	buf := bytes.NewBuffer(nil)
	command := args[0]
	cmd := exec.Command(command, args[1:]...) // nolint: gas
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to execute linter %s: %s", command, err)
	}

	done := make(chan bool)
	go func() {
		err = cmd.Wait()
		done <- true
	}()

	// Wait for process to complete or deadline to expire.
	select {
	case <-done:

	case <-ctx.Done():
		err = fmt.Errorf("deadline exceeded by linter %s (try increasing --deadline)",
			ctx.Name())
		kerr := cmd.Process.Kill()
		if kerr != nil {
			Warning("failed to kill %s: %s", ctx.Name(), kerr)
		}
		return nil, err
	}

	if err != nil {
		ctx.Debug("warning: %s returned %s: %s", command, err, buf.String())
	}

	processOutput(ctx, linter, buf.Bytes())
	elapsed := time.Since(start)
	ctx.Debug("took %s", elapsed)
	return nil
}

func parseCommand(command string) ([]string, error) {
	args, err := shlex.Split(command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("invalid command %q", command)
	}
	exe, err := exec.LookPath(args[0])
	if err != nil {
		return nil, err
	}
	return append([]string{exe}, args[1:]...), nil
}

// nolint: gocyclo
func processOutput(ctx api.Context, linter *Linter, out []byte) {
	re := linter.def.Pattern.Regexp
	all := re.FindAllSubmatchIndex(out, -1)
	ctx.Debug("%s hits %d: %s", ctx.Name(), len(all), linter.def.Pattern)

	cwd, err := os.Getwd()
	if err != nil {
		Warning("failed to get working directory %s", err)
	}

	// Create a local copy of vars so they can be modified by the linter output
	vars := state.vars.Copy()

	for _, indices := range all {
		group := [][]byte{}
		for i := 0; i < len(indices); i += 2 {
			var fragment []byte
			if indices[i] != -1 {
				fragment = out[indices[i]:indices[i+1]]
			}
			group = append(group, fragment)
		}

		issue := api.NewIssue(state.Linter.Name)

		for i, name := range re.SubexpNames() {
			if group[i] == nil {
				continue
			}
			part := string(group[i])
			if name != "" {
				vars[name] = part
			}
			switch name {
			case "path":
				issue.Path = relativePath(cwd, part)

			case "line":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "line matched invalid integer")
				issue.Line = int(n)

			case "col":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "col matched invalid integer")
				issue.Col = int(n)

			case "message":
				issue.Message = part

			case "":
			}
		}
		// TODO: set messageOveride and severity on the Linter instead of reading
		// them directly from the static config
		if m, ok := config.MessageOverride[state.Name]; ok {
			issue.Message = vars.Replace(m)
		}
		if sev, ok := config.Severity[state.Name]; ok {
			issue.Severity = api.Severity(sev)
		}
		if state.exclude != nil && state.exclude.MatchString(issue.String()) {
			continue
		}
		if state.include != nil && !state.include.MatchString(issue.String()) {
			continue
		}
		state.issues <- issue
	}
}
