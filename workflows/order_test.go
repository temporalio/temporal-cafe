package workflows_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/temporalio/temporal-cafe/activities"
	"github.com/temporalio/temporal-cafe/api"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

var a *activities.Activities

func TestOrderWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(a.ProcessPayment)
	env.RegisterActivity(a.ProcessPaymentRefund)
	env.RegisterWorkflow(workflows.KitchenOrder)
	env.RegisterWorkflow(workflows.BaristaOrder)
	env.RegisterActivity(a.AddLoyaltyPoints)

	input := &api.OrderWorkflowInput{
		Email:        "test@example.com",
		PaymentToken: "x",
		Items: []api.OrderLineItem{
			{Type: api.OrderLineItemTypeBeverage, Name: "coffee", Count: 1},
			{Type: api.OrderLineItemTypeBeverage, Name: "latte", Count: 2},
			{Type: api.OrderLineItemTypeFood, Name: "bagel", Count: 2},
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

	env.OnActivity(a.ProcessPayment, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *activities.ProcessPaymentInput) (*activities.ProcessPaymentResult, error) {
		return &activities.ProcessPaymentResult{}, nil
	})

	env.OnActivity(a.ProcessPaymentRefund, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *activities.ProcessPaymentRefundInput) (*activities.ProcessPaymentRefundResult, error) {
		return &activities.ProcessPaymentRefundResult{}, nil
	})

	env.OnActivity(a.AddLoyaltyPoints, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *activities.AddLoyaltyPointsInput) (*activities.AddLoyaltyPointsResult, error) {
		return &activities.AddLoyaltyPointsResult{}, nil
	})

	var activityCalls []string
	env.SetOnActivityStartedListener(func(activityInfo *activity.Info, ctx context.Context, args converter.EncodedValues) {
		activityCalls = append(activityCalls, activityInfo.ActivityType.Name)
	})

	expectedCalls := []string{
		"ProcessPayment",
		"AddLoyaltyPoints",
	}

	env.ExecuteWorkflow(workflows.Order, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result api.OrderWorfklowResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)

	assert.Equal(t, expectedCalls, activityCalls)
}
