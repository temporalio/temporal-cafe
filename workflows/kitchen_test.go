package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/testsuite"
)

type kitchenUpdateCallback struct {
	accept   func()
	reject   func(error)
	complete func(interface{}, error)
}

func (uc *kitchenUpdateCallback) Accept() {
	if uc.accept != nil {
		uc.accept()
	}
}

func (uc *kitchenUpdateCallback) Reject(err error) {
	if uc.reject != nil {
		uc.reject(err)
	}
}

func (uc *kitchenUpdateCallback) Complete(success interface{}, err error) {
	if uc.complete != nil {
		uc.complete(success, err)
	}
}

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
		uc := kitchenUpdateCallback{
			reject: func(err error) {
				t.Error(err)
			},
		}

		env.UpdateWorkflow(
			proto.KitchenOrderItemStatusSignal,
			"",
			&uc,
			&proto.KitchenOrderItemStatusUpdate{
				Line:   1,
				Status: proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED,
			},
		)
		env.UpdateWorkflow(
			proto.KitchenOrderItemStatusSignal,
			"",
			&uc,
			&proto.KitchenOrderItemStatusUpdate{
				Line:   2,
				Status: proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED,
			},
		)
	}, 1)

	env.ExecuteWorkflow(workflows.KitchenOrder, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result proto.KitchenOrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
}
