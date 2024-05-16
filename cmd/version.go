package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"os"
)

var (
	Version   = "unknown"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("version:", Version, "build time:", BuildTime)
		os.Exit(0)
	},
}
