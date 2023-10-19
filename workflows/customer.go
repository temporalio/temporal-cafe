package workflows

import (
	"github.com/temporalio/temporal-cafe/api"
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
	ch := workflow.GetSignalChannel(ctx, api.CustomerLoyaltyPointsEarnedSignal)
	s := workflow.NewSelector(ctx)

	s.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal api.CustomerLoyaltyPointsEarned
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

func Customer(ctx workflow.Context, input *api.CustomerInput, state *CustomerWorkflowState) error {
	wf := NewCustomerWorkflowState(state)

	workflow.SetQueryHandler(ctx, api.CustomerLoyaltyPointsBalanceQuery, func() (*api.CustomerLoyaltyPointsBalance, error) {
		return &api.CustomerLoyaltyPointsBalance{Points: wf.Points}, nil
	})

	handleEvents(ctx, wf)

	return workflow.NewContinueAsNewError(ctx, Customer, input, wf)
}
