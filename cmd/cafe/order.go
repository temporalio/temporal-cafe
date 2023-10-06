package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/client"
)

var foodItems []string
var beverageItems []string

// orderCmd represents the order command
var orderCmd = &cobra.Command{
	Use:   "order item ...",
	Short: "Place an order",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Dial(client.Options{})
		if err != nil {
			log.Fatalf("client error: %v", err)
		}
		defer c.Close()

		items := []workflows.OrderLineItem{}
		for _, v := range foodItems {
			items = append(items, workflows.OrderLineItem{Type: workflows.OrderLineItemTypeFood, Name: v, Count: 1})
		}
		for _, v := range beverageItems {
			items = append(items, workflows.OrderLineItem{Type: workflows.OrderLineItemTypeBeverage, Name: v, Count: 1})
		}

		order := workflows.OrderWorkflowInput{
			Items: items,
		}

		we, err := c.ExecuteWorkflow(
			context.Background(),
			client.StartWorkflowOptions{
				TaskQueue: "cafe",
			},
			"Order",
			order,
		)
		if err != nil {
			return err
		}

		fmt.Printf("Order %s created.\n", we.GetID())

		return nil
	},
}

func init() {
	orderCmd.Flags().StringArrayVarP(&foodItems, "food", "f", []string{}, "Food")
	orderCmd.Flags().StringArrayVarP(&beverageItems, "beverage", "b", []string{}, "Beverage")

	rootCmd.AddCommand(orderCmd)
}
