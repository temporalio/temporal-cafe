package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/client"
)

var kitchenOrderID string
var kitchenOrderItemNumber int

// kitchenCmd represents the kitchen command
var kitchenCmd = &cobra.Command{
	Use:   "kitchen",
	Short: "Kitchen commands",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// kitchenStatusCmd represents the kitchen get command
var kitchenStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of an order",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Dial(client.Options{})
		if err != nil {
			log.Fatalf("client error: %v", err)
		}
		defer c.Close()

		ctx := context.Background()

		v, err := c.QueryWorkflow(ctx, kitchenOrderID, "", workflows.KitchenOrderStatusQueryName)
		if err != nil {
			return err
		}

		var status workflows.KitchenOrderWorfklowStatus
		err = v.Get(&status)
		if err != nil {
			return err
		}

		fmt.Printf("Kitchen Order: %s\n", kitchenOrderID)
		for i, item := range status.Items {
			fmt.Printf("%d:\t[%s]\t%s\n", i+1, item.Status, item.Name)
		}

		return nil
	},
}

// kitchenUpdatedCmd represents the kitchen update command
var kitchenUpdateCmd = &cobra.Command{
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
		case workflows.KitchenOrderItemStatusStarted:
			s = workflows.KitchenOrderItemStartedSignalName
			si = workflows.KitchenOrderItemStartedSignal{Line: kitchenOrderItemNumber}
		case workflows.KitchenOrderItemStatusCompleted:
			s = workflows.KitchenOrderItemCompletedSignalName
			si = workflows.KitchenOrderItemCompletedSignal{Line: kitchenOrderItemNumber}
		case workflows.KitchenOrderItemStatusFailed:
			s = workflows.KitchenOrderItemFailedSignalName
			si = workflows.KitchenOrderItemFailedSignal{Line: kitchenOrderItemNumber}
		default:
			return fmt.Errorf("unknown status: %s", args[0])
		}

		err = c.SignalWorkflow(ctx, kitchenOrderID, "", s, si)
		if err != nil {
			return err
		}

		fmt.Printf("Kitchen Order: %s\n", kitchenOrderID)
		fmt.Printf("Sent update: %d:\t[%s]\n", kitchenOrderItemNumber, args[0])

		return nil
	},
}

func init() {
	kitchenStatusCmd.Flags().StringVarP(&kitchenOrderID, "id", "i", "", "Order ID (required)")
	kitchenStatusCmd.MarkFlagRequired("id")

	kitchenUpdateCmd.Flags().StringVarP(&kitchenOrderID, "id", "i", "", "Order ID (required)")
	kitchenUpdateCmd.MarkFlagRequired("id")
	kitchenUpdateCmd.Flags().IntVarP(&kitchenOrderItemNumber, "number", "n", 0, "Item Number (required)")
	kitchenUpdateCmd.MarkFlagRequired("number")

	kitchenCmd.AddCommand(kitchenStatusCmd)
	kitchenCmd.AddCommand(kitchenUpdateCmd)
	rootCmd.AddCommand(kitchenCmd)
}
