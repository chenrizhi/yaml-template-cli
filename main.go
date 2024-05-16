package main

import (
	"github.com/spf13/cobra"
	"log"
	"os"
	"yaml-template-cli/cmd"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	rootCmd, err := cmd.NewRootCmd(os.Stdout, os.Args[1:])
	if err != nil {
		log.Printf("%+v\n", err)
		os.Exit(1)
	}

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		// TODO
	})

	if err := rootCmd.Execute(); err != nil {
		log.Printf("%+v\n", err)
		os.Exit(1)
	}
}
