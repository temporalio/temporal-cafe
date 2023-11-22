package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/cmd/cafe/ui"
)

// kitchenCmd represents the kitchen command
var kitchenCmd = &cobra.Command{
	Use:   "kitchen",
	Short: "Kitchen commands",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

var kitchenBoardCmd = &cobra.Command{
	Use:   "board",
	Short: "Show kitchen order board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := ui.KitchenBoard{}

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
	kitchenCmd.AddCommand(kitchenBoardCmd)
	rootCmd.AddCommand(kitchenCmd)
}
