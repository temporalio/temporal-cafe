package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/testsuite"
)

func TestKitchenWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.KitchenOrder)

	input := &proto.KitchenOrderInput{
		Items: []*proto.OrderLineItem{
			{Name: "bagel", Count: 1},
			{Name: "muffin", Count: 1},
		},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(
			proto.KitchenOrderItemStatusSignal,
			proto.KitchenOrderItemStatusUpdate{Line: 1, Status: proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED},
		)

		env.SignalWorkflow(
			proto.KitchenOrderItemStatusSignal,
			proto.KitchenOrderItemStatusUpdate{Line: 2, Status: proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED},
		)
	}, 1)

	env.ExecuteWorkflow(workflows.KitchenOrder, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result proto.KitchenOrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
}
