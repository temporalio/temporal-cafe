package workflows

import (
	"fmt"

	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type KitchenOrderWorfklow struct {
	Status *api.KitchenOrderStatus
}

func (s *KitchenOrderWorfklow) signalFulfilmentStarted(ctx workflow.Context) error {
	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	signal := workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, api.OrderFulfilmentStartedSignal, nil)
	return signal.Get(ctx, nil)
}

func NewKitchenOrderWorkflow(items []*api.OrderLineItem) *KitchenOrderWorfklow {
	var kitchenItems []*api.KitchenOrderLineItem
	for _, li := range items {
		for i := uint32(0); i < li.Count; i++ {
			kitchenItems = append(kitchenItems, &api.KitchenOrderLineItem{Name: li.Name})
		}
	}

	return &KitchenOrderWorfklow{Status: &api.KitchenOrderStatus{Open: true, Items: kitchenItems}}
}

func (s *KitchenOrderWorfklow) updateItem(ctx workflow.Context, line uint32, status api.KitchenOrderItemStatus) error {
	if line < 1 || line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid line item: %d", line)
	}

	// Adjust item number because array is 0-indexed.
	s.Status.Items[line-1].Status = status

	return nil
}

func (s *KitchenOrderWorfklow) checkForOrderCompleted() {
	for _, v := range s.Status.Items {
		if v.Status != api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED {
			return
		}
	}
	s.Status.Open = false
}

func (s *KitchenOrderWorfklow) waitForItems(ctx workflow.Context) error {
	sel := workflow.NewSelector(ctx)

	var err error
	var fulfilmentStarted = false
	var fulfilmentSignalled = false

	// Listen for signals from Kitchen staff
	ch := workflow.GetSignalChannel(ctx, api.KitchenOrderItemStatusSignal)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal api.KitchenOrderItemStatusUpdate
		c.Receive(ctx, &signal)

		err = s.updateItem(ctx, signal.Line, signal.Status)
		if err != nil {
			return
		}

		switch signal.Status {
		case api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_STARTED:
			fulfilmentStarted = true
		case api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED:
			fulfilmentStarted = true
			s.checkForOrderCompleted()
		case api.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_FAILED:
			err = fmt.Errorf("item %d failed", signal.Line)
		}
	})

	// Listen for Workflow cancellation
	sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {
		err = temporal.NewCanceledError()
	})

	for s.Status.Open {
		sel.Select(ctx)
		if err != nil {
			s.Status.Open = false
			return err
		}
		if fulfilmentStarted && !fulfilmentSignalled {
			err = s.signalFulfilmentStarted(ctx)
			if err != nil {
				return err
			}
			fulfilmentSignalled = true
		}
	}

	return nil
}

func KitchenOrder(ctx workflow.Context, input *api.KitchenOrderInput) (*api.KitchenOrderResult, error) {
	wf := NewKitchenOrderWorkflow(input.Items)

	err := workflow.SetQueryHandler(ctx, api.KitchenOrderStatusQuery, func() (*api.KitchenOrderStatus, error) {
		return wf.Status, nil
	})
	if err != nil {
		return &api.KitchenOrderResult{}, err
	}

	err = wf.waitForItems(ctx)

	return &api.KitchenOrderResult{}, err
}
