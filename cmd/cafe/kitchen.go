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

		v, err := c.QueryWorkflow(ctx, kitchenOrderID, "", proto.KitchenOrderStatusQuery)
		if err != nil {
			return err
		}

		var status proto.KitchenOrderStatus
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

		statusName := fmt.Sprintf("KITCHEN_ORDER_ITEM_STATUS_%s", strings.ToUpper(args[0]))
		status, ok := proto.KitchenOrderItemStatus_value[statusName]
		if !ok {
			return fmt.Errorf("unknown status: %s", args[0])
		}

		err = c.SignalWorkflow(
			ctx,
			kitchenOrderID,
			"",
			proto.KitchenOrderItemStatusSignal,
			proto.KitchenOrderItemStatusUpdate{
				Line:   uint32(kitchenOrderItemNumber),
				Status: proto.KitchenOrderItemStatus(status),
			},
		)
		if err != nil {
			return err
		}

		fmt.Printf("Kitchen Order: %s\n", kitchenOrderID)
		fmt.Printf("Sent update: %d:\t[%s]\n", kitchenOrderItemNumber, args[0])

		return nil
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
	kitchenStatusCmd.Flags().StringVarP(&kitchenOrderID, "id", "i", "", "Order ID (required)")
	kitchenStatusCmd.MarkFlagRequired("id")

	kitchenUpdateCmd.Flags().StringVarP(&kitchenOrderID, "id", "i", "", "Order ID (required)")
	kitchenUpdateCmd.MarkFlagRequired("id")
	kitchenUpdateCmd.Flags().IntVarP(&kitchenOrderItemNumber, "number", "n", 0, "Item Number (required)")
	kitchenUpdateCmd.MarkFlagRequired("number")

	kitchenCmd.AddCommand(kitchenStatusCmd)
	kitchenCmd.AddCommand(kitchenUpdateCmd)
	kitchenCmd.AddCommand(kitchenBoardCmd)
	rootCmd.AddCommand(kitchenCmd)
}
