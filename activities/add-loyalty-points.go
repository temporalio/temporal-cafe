package activities

import (
	"context"
	"fmt"

	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/client"
)

func (a *Activities) AddLoyaltyPoints(ctx context.Context, input *api.AddLoyaltyPointsInput) (*api.AddLoyaltyPointsResult, error) {
	_, err := a.Client.SignalWithStartWorkflow(
		ctx,
		fmt.Sprintf("customer:%s", input.Email),
		api.CustomerLoyaltyPointsEarnedSignal,
		api.CustomerLoyaltyPointsEarned{
			Points: input.Points,
		},
		client.StartWorkflowOptions{
			TaskQueue: "cafe",
		},
		"Customer",
		api.CustomerInput{
			Email: input.Email,
		},
	)

	return &api.AddLoyaltyPointsResult{}, err
}
