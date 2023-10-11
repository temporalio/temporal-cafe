package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/activities"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestOrderWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(activities.ProcessPayment)
	env.RegisterActivity(activities.ProcessPaymentRefund)
	env.RegisterWorkflow(workflows.KitchenOrder)
	env.RegisterWorkflow(workflows.BaristaOrder)

	input := &workflows.OrderWorkflowInput{
		PaymentToken: "x",
		Items: []workflows.OrderLineItem{
			{Type: workflows.OrderLineItemTypeBeverage, Name: "coffee", Count: 1},
			{Type: workflows.OrderLineItemTypeBeverage, Name: "latte", Count: 2},
			{Type: workflows.OrderLineItemTypeFood, Name: "bagel", Count: 2},
		},
	}

	env.SetOnChildWorkflowStartedListener(func(workflowInfo *workflow.Info, ctx workflow.Context, args converter.EncodedValues) {
		wid := workflowInfo.WorkflowExecution.ID

		if workflowInfo.WorkflowType.Name == "BaristaOrder" {
			env.SignalWorkflowByID(
				wid,
				workflows.BaristaOrderItemCompletedSignalName,
				workflows.BaristaOrderItemCompletedSignal{Line: 1},
			)
			env.SignalWorkflowByID(
				wid,
				workflows.BaristaOrderItemCompletedSignalName,
				workflows.BaristaOrderItemCompletedSignal{Line: 2},
			)
			env.SignalWorkflowByID(
				wid,
				workflows.BaristaOrderItemCompletedSignalName,
				workflows.BaristaOrderItemCompletedSignal{Line: 3},
			)
		}

		if workflowInfo.WorkflowType.Name == "KitchenOrder" {
			env.SignalWorkflowByID(
				wid,
				workflows.KitchenOrderItemCompletedSignalName,
				workflows.KitchenOrderItemCompletedSignal{Line: 1},
			)

			env.SignalWorkflowByID(
				wid,
				workflows.KitchenOrderItemCompletedSignalName,
				workflows.KitchenOrderItemCompletedSignal{Line: 2},
			)
		}
	})

	env.ExecuteWorkflow(workflows.Order, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result workflows.OrderWorfklowResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
}
