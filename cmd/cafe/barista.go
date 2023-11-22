package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/cmd/cafe/ui"
)

// baristaCmd represents the barista command
var baristaCmd = &cobra.Command{
	Use:   "barista",
	Short: "Barista commands",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

var baristaBoardCmd = &cobra.Command{
	Use:   "board",
	Short: "Show barista order board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := ui.BaristaBoard{}

		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return err
		}
		defer f.Close()

		p := tea.NewProgram(b, tea.WithAltScreen())
		_, err = p.Run()

		return err
	},
}

func init() {
	baristaCmd.AddCommand(baristaBoardCmd)
	rootCmd.AddCommand(baristaCmd)
}
