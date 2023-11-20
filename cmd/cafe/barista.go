package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/cmd/cafe/ui"
	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/client"
)

var baristaOrderID string
var baristaOrderItemNumber int

// baristaCmd represents the barista command
var baristaCmd = &cobra.Command{
	Use:   "barista",
	Short: "Barista commands",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// baristaStatusCmd represents the barista get command
var baristaStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of an order",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Dial(client.Options{})
		if err != nil {
			log.Fatalf("client error: %v", err)
		}
		defer c.Close()

		ctx := context.Background()

		v, err := c.QueryWorkflow(ctx, baristaOrderID, "", proto.BaristaOrderStatusQuery)
		if err != nil {
			return err
		}

		var status proto.BaristaOrderStatus
		err = v.Get(&status)
		if err != nil {
			return err
		}

		fmt.Printf("Barista Order: %s\n", baristaOrderID)
		for i, item := range status.Items {
			fmt.Printf("%d:\t[%s]\t%s\n", i+1, item.Status, item.Name)
		}

		return nil
	},
}

// baristaUpdatedCmd represents the barista update command
var baristaUpdateCmd = &cobra.Command{
	Use:   "update started|completed|failed",
	Short: "Update the status of an order",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Dial(client.Options{})
		if err != nil {
			log.Fatalf("client error: %v", err)
		}
		defer c.Close()

		ctx := context.Background()

		statusName := fmt.Sprintf("BARISTA_ORDER_ITEM_STATUS_%s", strings.ToUpper(args[0]))
		status, ok := proto.BaristaOrderItemStatus_value[statusName]
		if !ok {
			return fmt.Errorf("unknown status: %s", args[0])
		}

		err = c.SignalWorkflow(
			ctx,
			baristaOrderID,
			"",
			proto.BaristaOrderItemStatusSignal,
			proto.BaristaOrderItemStatusUpdate{
				Line:   uint32(baristaOrderItemNumber),
				Status: proto.BaristaOrderItemStatus(status),
			},
		)
		if err != nil {
			return err
		}

		fmt.Printf("Barista Order: %s\n", baristaOrderID)
		fmt.Printf("Sent update: %d:\t[%s]\n", baristaOrderItemNumber, args[0])

		return nil
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
	baristaStatusCmd.Flags().StringVarP(&baristaOrderID, "id", "i", "", "Order ID (required)")
	baristaStatusCmd.MarkFlagRequired("id")

	baristaUpdateCmd.Flags().StringVarP(&baristaOrderID, "id", "i", "", "Order ID (required)")
	baristaUpdateCmd.MarkFlagRequired("id")
	baristaUpdateCmd.Flags().IntVarP(&baristaOrderItemNumber, "number", "n", 0, "Item Number (required)")
	baristaUpdateCmd.MarkFlagRequired("number")

	baristaCmd.AddCommand(baristaStatusCmd)
	baristaCmd.AddCommand(baristaUpdateCmd)
	baristaCmd.AddCommand(baristaBoardCmd)
	rootCmd.AddCommand(baristaCmd)
}
