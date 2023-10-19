package workflows_test

import (
	"context"
	"fmt"
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

	input := &api.OrderInput{
		Email:        "test@example.com",
		PaymentToken: "x",
		Items: []*api.OrderLineItem{
			{Type: api.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "coffee", Count: 1},
			{Type: api.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "latte", Count: 2},
			{Type: api.ProductType_PRODUCT_TYPE_FOOD, Name: "bagel", Count: 2},
		},
	}

	env.SetOnChildWorkflowStartedListener(func(workflowInfo *workflow.Info, ctx workflow.Context, args converter.EncodedValues) {
		wid := workflowInfo.WorkflowExecution.ID

		if workflowInfo.WorkflowType.Name == "BaristaOrder" {
			env.SignalWorkflowByID(
				wid,
				api.BaristaOrderItemStatusSignal,
				api.BaristaOrderItemStatusUpdate{
					Line:   1,
					Status: api.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
			env.SignalWorkflowByID(
				wid,
				api.BaristaOrderItemStatusSignal,
				api.BaristaOrderItemStatusUpdate{
					Line:   2,
					Status: api.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
			env.SignalWorkflowByID(
				wid,
				api.BaristaOrderItemStatusSignal,
				api.BaristaOrderItemStatusUpdate{
					Line:   3,
					Status: api.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
		}

		if workflowInfo.WorkflowType.Name == "KitchenOrder" {
			env.SignalWorkflowByID(
				wid,
				api.KitchenOrderItemStatusSignal,
				api.KitchenOrderItemStatusUpdate{
					Line:   1,
					Status: api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
			env.SignalWorkflowByID(
				wid,
				api.KitchenOrderItemStatusSignal,
				api.KitchenOrderItemStatusUpdate{
					Line:   2,
					Status: api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
		}
	})

	env.OnActivity(a.ProcessPayment, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *api.ProcessPaymentInput) (*api.ProcessPaymentResult, error) {
		return &api.ProcessPaymentResult{}, nil
	})

	env.OnActivity(a.ProcessPaymentRefund, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *api.ProcessPaymentRefundInput) (*api.ProcessPaymentRefundResult, error) {
		return &api.ProcessPaymentRefundResult{}, nil
	})

	env.OnActivity(a.AddLoyaltyPoints, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *api.AddLoyaltyPointsInput) (*api.AddLoyaltyPointsResult, error) {
		return &api.AddLoyaltyPointsResult{}, nil
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

	var result api.OrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)

	assert.Equal(t, expectedCalls, activityCalls)
}

func TestOrderWorkflowFulfilmentDeadline(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(a.ProcessPayment)
	env.RegisterActivity(a.ProcessPaymentRefund)
	env.RegisterWorkflow(workflows.KitchenOrder)
	env.RegisterWorkflow(workflows.BaristaOrder)

	input := &api.OrderInput{
		Email:        "test@example.com",
		PaymentToken: "x",
		Items: []*api.OrderLineItem{
			{Type: api.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "coffee", Count: 1},
			{Type: api.ProductType_PRODUCT_TYPE_FOOD, Name: "bagel", Count: 1},
		},
	}

	env.SetOnChildWorkflowStartedListener(func(workflowInfo *workflow.Info, ctx workflow.Context, args converter.EncodedValues) {
		wid := workflowInfo.WorkflowExecution.ID

		if workflowInfo.WorkflowType.Name == "BaristaOrder" {
			env.SignalWorkflowByID(
				wid,
				api.BaristaOrderItemStatusSignal,
				api.BaristaOrderItemStatusUpdate{
					Line:   1,
					Status: api.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
				},
			)
		}
	})

	env.OnActivity(a.ProcessPayment, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *api.ProcessPaymentInput) (*api.ProcessPaymentResult, error) {
		return &api.ProcessPaymentResult{}, nil
	})

	env.OnActivity(a.ProcessPaymentRefund, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *api.ProcessPaymentRefundInput) (*api.ProcessPaymentRefundResult, error) {
		return &api.ProcessPaymentRefundResult{}, nil
	})

	var activityCalls []string
	env.SetOnActivityStartedListener(func(activityInfo *activity.Info, ctx context.Context, args converter.EncodedValues) {
		activityCalls = append(activityCalls, activityInfo.ActivityType.Name)
	})

	expectedCalls := []string{
		"ProcessPayment",
		"ProcessPaymentRefund",
	}

	env.ExecuteWorkflow(workflows.Order, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result api.OrderResult

	err := env.GetWorkflowResult(&result)
	assert.Error(t, fmt.Errorf("order not fulfilled within window"), err)

	assert.Equal(t, expectedCalls, activityCalls)
}
