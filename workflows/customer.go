package workflows

import (
	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/workflow"
)

// CustomerStartingBalance is the number of points to credit a loyalty account on signup
const CustomerStartingBalance = 100

type CustomerWorkflowState struct {
	Points uint32
}

// NewCustomerWorkflowState creates a workflow state
func NewCustomerWorkflowState(state *CustomerWorkflowState) *CustomerWorkflowState {
	if state != nil {
		return state
	}

	return &CustomerWorkflowState{Points: CustomerStartingBalance}
}

func handleEvents(ctx workflow.Context, state *CustomerWorkflowState) error {
	ch := workflow.GetSignalChannel(ctx, proto.CustomerLoyaltyPointsEarnedSignal)
	s := workflow.NewSelector(ctx)

	s.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal proto.CustomerLoyaltyPointsEarned
		c.Receive(ctx, &signal)

		state.Points += signal.Points
	})

	for {
		s.Select(ctx)
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			break
		}
	}
	for s.HasPending() {
		s.Select(ctx)
	}

	return nil
}

func Customer(ctx workflow.Context, input *proto.CustomerInput, state *CustomerWorkflowState) error {
	wf := NewCustomerWorkflowState(state)

	workflow.SetQueryHandler(ctx, proto.CustomerLoyaltyPointsBalanceQuery, func() (*proto.CustomerLoyaltyPointsBalance, error) {
		return &proto.CustomerLoyaltyPointsBalance{Points: wf.Points}, nil
	})

	handleEvents(ctx, wf)

	return workflow.NewContinueAsNewError(ctx, Customer, input, wf)
}
