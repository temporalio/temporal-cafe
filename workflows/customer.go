package workflows

import (
	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/workflow"
)

// CustomerStartingBalance is the number of points to credit a loyalty account on signup
const CustomerStartingBalance = 100

// NewCustomerWorkflowState creates a workflow state
func NewCustomerWorkflowState(state *api.CustomerWorkflowState) *api.CustomerWorkflowState {
	if state != nil {
		return state
	}

	return &api.CustomerWorkflowState{Points: CustomerStartingBalance}
}

func handleEvents(ctx workflow.Context, state *api.CustomerWorkflowState) error {
	ch := workflow.GetSignalChannel(ctx, api.CustomerPointsAddSignalName)
	s := workflow.NewSelector(ctx)

	s.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal api.CustomerPointsAddSignal
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

func Customer(ctx workflow.Context, input *api.CustomerWorkflowInput, state *api.CustomerWorkflowState) error {
	wf := NewCustomerWorkflowState(state)

	workflow.SetQueryHandler(ctx, api.CustomerPointsBalanceQueryName, func() (*api.CustomerPointsBalanceQuery, error) {
		return &api.CustomerPointsBalanceQuery{Points: wf.Points}, nil
	})

	handleEvents(ctx, wf)

	return workflow.NewContinueAsNewError(ctx, Customer, input, wf)
}
