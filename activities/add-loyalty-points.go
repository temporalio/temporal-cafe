package activities

import (
	"context"
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/client"
)

func (a *Activities) AddLoyaltyPoints(ctx context.Context, input *proto.AddLoyaltyPointsInput) (*proto.AddLoyaltyPointsResult, error) {
	_, err := a.Client.SignalWithStartWorkflow(
		ctx,
		fmt.Sprintf("customer:%s", input.Email),
		proto.CustomerLoyaltyPointsEarnedSignal,
		proto.CustomerLoyaltyPointsEarned{
			Points: input.Points,
		},
		client.StartWorkflowOptions{
			TaskQueue: "cafe",
		},
		"Customer",
		proto.CustomerInput{
			Email: input.Email,
		},
	)

	return &proto.AddLoyaltyPointsResult{}, err
}
