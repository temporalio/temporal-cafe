package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/cmd/cafe/ui"
)

var orderCmd = &cobra.Command{
	Use:   "order item ...",
	Short: "Place an order",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := ui.NewPOS()

		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return err
		}
		defer f.Close()

		p := tea.NewProgram(b, tea.WithAltScreen(), tea.WithMouseCellMotion())
		_, err = p.Run()

		return err
	},
}

func init() {
	rootCmd.AddCommand(orderCmd)
}
