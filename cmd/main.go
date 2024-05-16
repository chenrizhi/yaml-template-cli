package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var settings = New()

func init() {
	log.SetFlags(log.Lshortfile)
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}

func main() {
	cmd, err := newRootCmd(os.Stdout, os.Args[1:])
	if err != nil {
		warning("%+v", err)
		os.Exit(1)
	}

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		// TODO
	})

	if err := cmd.Execute(); err != nil {
		debug("%+v", err)
		os.Exit(1)
	}
}
