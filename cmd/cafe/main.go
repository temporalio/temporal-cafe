package main

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cafe",
	Short: "Command line tool for Temporal Cafe application.",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}
