package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/workflows"
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

		v, err := c.QueryWorkflow(ctx, baristaOrderID, "", workflows.BaristaOrderStatusQueryName)
		if err != nil {
			return err
		}

		var status workflows.BaristaOrderWorfklowStatus
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

		var s string
		var si interface{}

		switch args[0] {
		case workflows.BaristaOrderItemStatusStarted:
			s = workflows.BaristaOrderItemStartedSignalName
			si = workflows.BaristaOrderItemStartedSignal{Line: baristaOrderItemNumber}
		case workflows.BaristaOrderItemStatusCompleted:
			s = workflows.BaristaOrderItemCompletedSignalName
			si = workflows.BaristaOrderItemCompletedSignal{Line: baristaOrderItemNumber}
		case workflows.BaristaOrderItemStatusFailed:
			s = workflows.BaristaOrderItemFailedSignalName
			si = workflows.BaristaOrderItemFailedSignal{Line: baristaOrderItemNumber}
		default:
			return fmt.Errorf("unknown status: %s", args[0])
		}

		err = c.SignalWorkflow(ctx, baristaOrderID, "", s, si)
		if err != nil {
			return err
		}

		fmt.Printf("Barista Order: %s\n", baristaOrderID)
		fmt.Printf("Sent update: %d:\t[%s]\n", baristaOrderItemNumber, args[0])

		return nil
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
	rootCmd.AddCommand(baristaCmd)
}
