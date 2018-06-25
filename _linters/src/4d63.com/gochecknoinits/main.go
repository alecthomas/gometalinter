package main // import "4d63.com/gochecknoinits"

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flagPrintHelp := flag.Bool("help", false, "")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gochecknoinits [path] [path] ...\n")
	}
	flag.Parse()

	if *flagPrintHelp {
		flag.Usage()
		return
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"./..."}
	}

	exitWithError := false

	for _, path := range paths {
		messages, err := checkNoInits(path)
		for _, message := range messages {
			fmt.Fprintf(os.Stdout, "%s\n", message)
			exitWithError = true
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			exitWithError = true
		}
	}

	if exitWithError {
		os.Exit(1)
	}
}
