package workflows

import (
	"fmt"
	"time"

	"github.com/temporalio/temporal-cafe/activities"
	"github.com/temporalio/temporal-cafe/api"
	workflowEnums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

const OrderStartToCompleteDeadline = 15 * time.Minute

type OrderWorkflowStatus struct {
	subOrders map[string]workflow.ChildWorkflowFuture
}

func NewOrderWorkflow() *OrderWorkflowStatus {
	return &OrderWorkflowStatus{
		subOrders: make(map[string]workflow.ChildWorkflowFuture),
	}
}

func (s OrderWorkflowStatus) processPayment(ctx workflow.Context, token string) (activities.ProcessPaymentResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	var result activities.ProcessPaymentResult
	err := workflow.ExecuteActivity(ctx, a.ProcessPayment, activities.ProcessPaymentInput{Token: token}).Get(ctx, &result)

	return result, err
}

func (s OrderWorkflowStatus) refundPayment(ctx workflow.Context, payment activities.Payment) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	err := workflow.ExecuteActivity(ctx, a.ProcessPaymentRefund, activities.ProcessPaymentRefundInput{Payment: payment}).Get(ctx, nil)

	return err
}

func (s OrderWorkflowStatus) sendSubOrders(ctx workflow.Context, items []api.OrderLineItem) workflow.CancelFunc {
	itemsByType := make(map[string][]api.OrderLineItem)

	for _, v := range items {
		itemsByType[v.Type] = append(itemsByType[v.Type], v)
	}

	childCtx, cancelChildren := workflow.WithCancel(ctx)

	childCtx = workflow.WithChildOptions(childCtx, workflow.ChildWorkflowOptions{
		ParentClosePolicy: workflowEnums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})

	for t, items := range itemsByType {
		var cw interface{}
		var input interface{}
		switch t {
		case api.OrderLineItemTypeFood:
			cw = KitchenOrder
			input = KitchenOrderWorkflowInput{Items: items}
		case api.OrderLineItemTypeBeverage:
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
	ch := workflow.GetSignalChannel(ctx, api.OrderStartedSignalName)
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

func (wf *OrderWorkflowStatus) itemCount(input *api.OrderWorkflowInput) uint {
	i := 0
	for _, item := range input.Items {
		i += item.Count
	}

	return uint(i)
}

func (wf *OrderWorkflowStatus) addLoyaltyPoints(ctx workflow.Context, input *api.OrderWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	items := wf.itemCount(input)
	err := workflow.ExecuteActivity(ctx, a.AddLoyaltyPoints, activities.AddLoyaltyPointsInput{Email: input.Email, Points: items}).Get(ctx, nil)

	return err
}

func Order(ctx workflow.Context, input *api.OrderWorkflowInput) (*api.OrderWorfklowResult, error) {
	wf := NewOrderWorkflow()

	p, err := wf.processPayment(ctx, input.PaymentToken)
	if err != nil {
		return &api.OrderWorfklowResult{}, err
	}
	defer func() {
		if err != nil {
			wf.refundPayment(ctx, p.Payment)
		}
	}()

	cancelSubOrders := wf.sendSubOrders(ctx, input.Items)
	err = wf.waitForSubOrders(ctx)
	if err != nil {
		cancelSubOrders()
		return &api.OrderWorfklowResult{}, err
	}

	if input.Email != "" {
		_ = wf.addLoyaltyPoints(ctx, input)
	}

	return &api.OrderWorfklowResult{}, nil
}
