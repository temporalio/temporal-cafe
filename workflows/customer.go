package workflows

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

const CustomerPointsBalanceQueryName = "customer-points-balance"
const CustomerPointsAddSignalName = "customer-points-add"

type CustomerPointsBalanceQuery struct {
	Points uint
}

type CustomerPointsAddSignal struct {
	Points uint
}

type CustomerWorkflowInput struct {
	Email string
	State *CustomerWorkflowState
}

type CustomerWorkflowState struct {
	Points uint
}

// CustomerStartingBalance is the number of points to credit a loyalty account on signup
const CustomerStartingBalance = 100

// NewCustomerWorkflowState creates a workflow state
func NewCustomerWorkflowState(input *CustomerWorkflowInput) *CustomerWorkflowState {
	if input.State == nil {
		input.State = &CustomerWorkflowState{Points: CustomerStartingBalance}
	}

	return input.State
}

func (wf *CustomerWorkflowState) handleEvents(ctx workflow.Context) error {
	ch := workflow.GetSignalChannel(ctx, CustomerPointsAddSignalName)
	s := workflow.NewSelector(ctx)

	s.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal CustomerPointsAddSignal
		c.Receive(ctx, &signal)

		wf.Points += signal.Points
	})

	for {
		fmt.Printf("select: %v\n", workflow.GetInfo(ctx).GetContinueAsNewSuggested())
		s.Select(ctx)
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			break
		}
	}
	for s.HasPending() {
		fmt.Printf("draining\n")
		s.Select(ctx)
	}

	return nil
}

func Customer(ctx workflow.Context, input *CustomerWorkflowInput) error {
	wf := NewCustomerWorkflowState(input)

	workflow.SetQueryHandler(ctx, CustomerPointsBalanceQueryName, func() (*CustomerPointsBalanceQuery, error) {
		return &CustomerPointsBalanceQuery{Points: wf.Points}, nil
	})

	wf.handleEvents(ctx)

	input.State = wf

	return workflow.NewContinueAsNewError(ctx, Customer, input)
}
