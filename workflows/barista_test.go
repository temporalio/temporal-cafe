package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/proto"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/testsuite"
)

type baristaUpdateCallback struct {
	accept   func()
	reject   func(error)
	complete func(interface{}, error)
}

func (uc *baristaUpdateCallback) Accept() {
	if uc.accept != nil {
		uc.accept()
	}
}

func (uc *baristaUpdateCallback) Reject(err error) {
	if uc.reject != nil {
		uc.reject(err)
	}
}

func (uc *baristaUpdateCallback) Complete(success interface{}, err error) {
	if uc.complete != nil {
		uc.complete(success, err)
	}
}

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
		uc := baristaUpdateCallback{
			reject: func(err error) {
				t.Error(err)
			},
		}

		env.UpdateWorkflow(
			proto.BaristaOrderItemStatusSignal,
			"",
			&uc,
			&proto.BaristaOrderItemStatusUpdate{
				Line:   1,
				Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
			},
		)
		env.UpdateWorkflow(
			proto.BaristaOrderItemStatusSignal,
			"",
			&uc,
			&proto.BaristaOrderItemStatusUpdate{
				Line:   2,
				Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
			},
		)
		env.UpdateWorkflow(
			proto.BaristaOrderItemStatusSignal,
			"",
			&uc,
			&proto.BaristaOrderItemStatusUpdate{
				Line:   3,
				Status: proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED,
			},
		)
	}, 1)

	env.ExecuteWorkflow(workflows.BaristaOrder, input)
	assert.True(t, env.IsWorkflowCompleted())

	var result proto.BaristaOrderResult

	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
}
