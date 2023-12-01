package workflows_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	act "github.com/temporalio/temporal-cafe/activities"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

var activities *act.Activities

func TestOrderWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(activities.ProcessPayment)
	env.RegisterActivity(activities.ProcessPaymentRefund)
	env.RegisterWorkflow(workflows.KitchenOrder)
	env.RegisterWorkflow(workflows.BaristaOrder)
	env.RegisterActivity(activities.AddLoyaltyPoints)

	input := &proto.OrderInput{
		Email:        "test@example.com",
		PaymentToken: "x",
		Items: []*proto.OrderLineItem{
			{Type: proto.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "coffee", Count: 1},
			{Type: proto.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "latte", Count: 2},
			{Type: proto.ProductType_PRODUCT_TYPE_FOOD, Name: "bagel", Count: 2},
		},
	}

	env.OnWorkflow(workflows.BaristaOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.BaristaOrderInput) (*proto.BaristaOrderResult, error) {
		return &proto.BaristaOrderResult{}, nil
	})

	env.OnWorkflow(workflows.KitchenOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.KitchenOrderInput) (*proto.KitchenOrderResult, error) {
		return &proto.KitchenOrderResult{}, nil
	})

	env.OnActivity(activities.ProcessPayment, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *proto.ProcessPaymentInput) (*proto.ProcessPaymentResult, error) {
		return &proto.ProcessPaymentResult{}, nil
	})

	env.OnActivity(activities.ProcessPaymentRefund, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *proto.ProcessPaymentRefundInput) (*proto.ProcessPaymentRefundResult, error) {
		return &proto.ProcessPaymentRefundResult{}, nil
	})

	env.OnActivity(activities.AddLoyaltyPoints, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *proto.AddLoyaltyPointsInput) (*proto.AddLoyaltyPointsResult, error) {
		return &proto.AddLoyaltyPointsResult{}, nil
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

	var result proto.OrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)

	assert.Equal(t, expectedCalls, activityCalls)
}

func TestOrderWorkflowFulfilmentDeadline(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(activities.ProcessPayment)
	env.RegisterActivity(activities.ProcessPaymentRefund)
	env.RegisterWorkflow(workflows.KitchenOrder)
	env.RegisterWorkflow(workflows.BaristaOrder)

	input := &proto.OrderInput{
		Email:        "test@example.com",
		PaymentToken: "x",
		Items: []*proto.OrderLineItem{
			{Type: proto.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "coffee", Count: 1},
			{Type: proto.ProductType_PRODUCT_TYPE_FOOD, Name: "bagel", Count: 1},
		},
	}

	env.OnWorkflow(workflows.BaristaOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.BaristaOrderInput) (*proto.BaristaOrderResult, error) {
		return &proto.BaristaOrderResult{}, fmt.Errorf("failed")
	})

	env.OnWorkflow(workflows.KitchenOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.KitchenOrderInput) (*proto.KitchenOrderResult, error) {
		return &proto.KitchenOrderResult{}, nil
	})

	env.OnActivity(activities.ProcessPayment, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *proto.ProcessPaymentInput) (*proto.ProcessPaymentResult, error) {
		return &proto.ProcessPaymentResult{}, nil
	})

	env.OnActivity(activities.ProcessPaymentRefund, mock.Anything, mock.Anything).Return(func(ctx context.Context, input *proto.ProcessPaymentRefundInput) (*proto.ProcessPaymentRefundResult, error) {
		return &proto.ProcessPaymentRefundResult{}, nil
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

	var result proto.OrderResult

	err := env.GetWorkflowResult(&result)
	assert.Error(t, fmt.Errorf("order not fulfilled within window"), err)

	assert.Equal(t, expectedCalls, activityCalls)
}

func TestOrderWorkflowRefund(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Order)
	env.RegisterActivity(activities.ProcessPayment)
	env.RegisterActivity(activities.ProcessPaymentRefund)

	input := &proto.OrderInput{
		PaymentToken: "x",
		Items: []*proto.OrderLineItem{
			{Type: proto.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "coffee", Count: 1},
			{Type: proto.ProductType_PRODUCT_TYPE_BEVERAGE, Name: "latte", Count: 2},
			{Type: proto.ProductType_PRODUCT_TYPE_FOOD, Name: "bagel", Count: 2},
		},
	}

	env.OnWorkflow(workflows.BaristaOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.BaristaOrderInput) (*proto.BaristaOrderResult, error) {
		return &proto.BaristaOrderResult{}, fmt.Errorf("failed")
	})
	env.OnWorkflow(workflows.KitchenOrder, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, input *proto.KitchenOrderInput) (*proto.KitchenOrderResult, error) {
		return &proto.KitchenOrderResult{}, nil
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

	var result proto.OrderResult

	err := env.GetWorkflowResult(&result)
	assert.Error(t, err)

	assert.Equal(t, expectedCalls, activityCalls)
}
