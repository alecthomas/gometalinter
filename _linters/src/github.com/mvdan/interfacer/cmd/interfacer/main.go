// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mvdan/interfacer"
)

var (
	verbose = flag.Bool("v", false, "print the names of packages as they are checked")
)

func main() {
	flag.Parse()
	if err := interfacer.CheckArgsOutput(flag.Args(), os.Stdout, *verbose); err != nil {
		errExit(err)
	}
}

func errExit(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}
