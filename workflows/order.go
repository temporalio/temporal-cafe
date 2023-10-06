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
const OrderLineItemTypeBeverage = "beverage"

type OrderLineItem struct {
	Name  string
	Type  string
	Count int
}

type OrderWorkflowInput struct {
	Items []OrderLineItem
}

type OrderWorfklowResult struct {
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
		var cw interface{}
		var input interface{}
		switch t {
		case OrderLineItemTypeFood:
			cw = KitchenOrder
			input = KitchenOrderWorkflowInput{Items: items}
		case OrderLineItemTypeBeverage:
			cw = BaristaOrder
			input = BaristaOrderWorkflowInput{Items: items}
		}
		s.subOrders[t] = workflow.ExecuteChildWorkflow(childCtx, cw, input)
	}

	return cancelChildren
}

func (s OrderWorkflowStatus) waitForSubOrders(ctx workflow.Context) error {
	var err error
	var orderTimer workflow.Future

	sel := workflow.NewSelector(ctx)

	// Handle SubOrder completion. We only care if there was an error here,
	// there is no meaningful result from SubOrders currently.
	for t, v := range s.subOrders {
		tt := t
		sel.AddFuture(v, func(f workflow.Future) {
			delete(s.subOrders, tt)
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
			return err
		}
	}

	return nil
}

func Order(ctx workflow.Context, input *OrderWorkflowInput) (*OrderWorfklowResult, error) {
	status := NewOrderWorkflowStatus()

	cancelSubOrders := status.sendSubOrders(ctx, input.Items)
	err := status.waitForSubOrders(ctx)
	if err != nil {
		cancelSubOrders()
	}

	return &OrderWorfklowResult{}, err
}
