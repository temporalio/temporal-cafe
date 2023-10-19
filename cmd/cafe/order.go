package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/client"
)

var email string
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

		items := []*api.OrderLineItem{}
		for _, v := range foodItems {
			items = append(items, &api.OrderLineItem{Type: api.ProductType_PRODUCT_TYPE_FOOD, Name: v, Count: 1})
		}
		for _, v := range beverageItems {
			items = append(items, &api.OrderLineItem{Type: api.ProductType_PRODUCT_TYPE_BEVERAGE, Name: v, Count: 1})
		}

		order := api.OrderInput{
			Email: email,
			Items: items,
		}

		we, err := c.ExecuteWorkflow(
			context.Background(),
			client.StartWorkflowOptions{
				TaskQueue: "cafe",
			},
			"Order",
			&order,
		)
		if err != nil {
			return err
		}

		fmt.Printf("Order %s created.\n", we.GetID())

		return nil
	},
}

func init() {
	orderCmd.Flags().StringVarP(&email, "email", "e", "", "Email")
	orderCmd.Flags().StringArrayVarP(&foodItems, "food", "f", []string{}, "Food")
	orderCmd.Flags().StringArrayVarP(&beverageItems, "beverage", "b", []string{}, "Beverage")

	rootCmd.AddCommand(orderCmd)
}
