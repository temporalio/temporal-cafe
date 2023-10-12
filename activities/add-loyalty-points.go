package activities

import (
	"context"

	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/client"
)

type AddLoyaltyPointsInput struct {
	Email  string
	Points uint
}

type AddLoyaltyPointsResult struct {
}

func (a *Activities) AddLoyaltyPoints(ctx context.Context, input *AddLoyaltyPointsInput) (*AddLoyaltyPointsResult, error) {
	_, err := a.Client.SignalWithStartWorkflow(
		ctx,
		api.CustomerWorkflowID(input.Email),
		api.CustomerPointsAddSignalName,
		api.CustomerPointsAddSignal{
			Points: input.Points,
		},
		client.StartWorkflowOptions{
			TaskQueue: "cafe",
		},
		"Customer",
		api.CustomerWorkflowInput{
			Email: input.Email,
		},
	)

	return &AddLoyaltyPointsResult{}, err
}
