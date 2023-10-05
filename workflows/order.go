package workflows

import (
	"fmt"
	"time"

	workflowEnums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

const OrderStartedSignalName = "order-started"
const OrderStartToCompleteDeadline = 15 * time.Minute

const OrderLineItemTypeFood = "food"
const OrderLineItemTypeBevarage = "beverage"

type OrderLineItem struct {
	Name  string
	Type  string
	Count int
}

type OrderWorkflowInput struct {
	Items []OrderLineItem
}

type OrderWorkflowStatus struct {
	subOrders map[string]workflow.Future
}

func NewOrderWorkflowStatus() *OrderWorkflowStatus {
	return &OrderWorkflowStatus{
		subOrders: make(map[string]workflow.Future),
	}
}

func (s OrderWorkflowStatus) sendSubOrders(ctx workflow.Context, items []OrderLineItem) workflow.CancelFunc {
	itemsByType := make(map[string][]OrderLineItem)

	for _, v := range items {
		if itemsByType[v.Type] == nil {
			itemsByType[v.Type] = []OrderLineItem{}
		}
		for i := 0; i < v.Count; i++ {
			itemsByType[v.Type] = append(itemsByType[v.Type], v)
		}
	}

	childCtx, cancelChildren := workflow.WithCancel(ctx)

	childCtx = workflow.WithChildOptions(childCtx, workflow.ChildWorkflowOptions{
		ParentClosePolicy: workflowEnums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})

	for t, items := range itemsByType {
		var subOrder workflow.Future
		switch t {
		case OrderLineItemTypeFood:
			subOrder = workflow.ExecuteChildWorkflow(
				childCtx,
				KitchenOrder,
				KitchenOrderWorkflowInput{
					Items: items,
				},
			)
		case OrderLineItemTypeBevarage:
			subOrder = workflow.ExecuteChildWorkflow(
				childCtx,
				BaristaOrder,
				BaristaOrderWorkflowInput{
					Items: items,
				},
			)
		}
		s.subOrders[t] = subOrder
	}

	return cancelChildren
}

func (s OrderWorkflowStatus) waitForSubOrders(ctx workflow.Context, cancelSubOrders workflow.CancelFunc) error {
	var err error
	var orderTimer workflow.Future

	sel := workflow.NewSelector(ctx)

	// Handle SubOrder completion. We only care if there was an error here,
	// there is no meaningful result from SubOrders currently.
	for t, v := range s.subOrders {
		sel.AddFuture(v, func(f workflow.Future) {
			delete(s.subOrders, t)
			err = f.Get(ctx, nil)
		})
	}

	// Set a timer once a SubOrder is started to ensure that everything is completed
	// within a specific duration.
	ch := workflow.GetSignalChannel(ctx, OrderStartedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)

		if orderTimer == nil {
			orderTimer = workflow.NewTimer(ctx, OrderStartToCompleteDeadline)
			sel.AddFuture(orderTimer, func(f workflow.Future) {
				err = fmt.Errorf("order not completed within deadline: %s", OrderStartToCompleteDeadline)
			})
		}
	})

	// Workflow Cancelled
	sel.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, _ bool) {
		err = ctx.Err()
	})

	for len(s.subOrders) > 0 {
		sel.Select(ctx)
		if err != nil {
			cancelSubOrders()
			return err
		}
	}

	return nil
}

type OrderWorfklowResult struct {
}

func Order(ctx workflow.Context, input *OrderWorkflowInput) (*OrderWorfklowResult, error) {
	status := NewOrderWorkflowStatus()

	cancelSubOrdersFunc := status.sendSubOrders(ctx, input.Items)
	err := status.waitForSubOrders(ctx, cancelSubOrdersFunc)

	return &OrderWorfklowResult{}, err
}
