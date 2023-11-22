package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/testsuite"
)

func TestBaristaWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.BaristaOrder)

	input := &proto.BaristaOrderInput{
		Items: []*proto.OrderLineItem{
			{Name: "coffee", Count: 1},
			{Name: "latte", Count: 2},
		},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(
			proto.BaristaOrderItemStatusSignal,
			proto.BaristaOrderItemStatusUpdate{Line: 1, Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED},
		)

		env.SignalWorkflow(
			proto.BaristaOrderItemStatusSignal,
			proto.BaristaOrderItemStatusUpdate{Line: 2, Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED},
		)

		env.SignalWorkflow(
			proto.BaristaOrderItemStatusSignal,
			proto.BaristaOrderItemStatusUpdate{Line: 3, Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED},
		)
	}, 1)

	env.ExecuteWorkflow(workflows.BaristaOrder, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result proto.BaristaOrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
}
