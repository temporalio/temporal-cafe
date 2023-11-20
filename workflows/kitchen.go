package workflows

import (
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type KitchenOrderWorfklow struct {
	Status *proto.KitchenOrderStatus
}

func (s *KitchenOrderWorfklow) signalFulfilmentStarted(ctx workflow.Context) error {
	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	signal := workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, proto.OrderFulfilmentStartedSignal, nil)
	return signal.Get(ctx, nil)
}

func NewKitchenOrderWorkflow(name string, items []*proto.OrderLineItem) *KitchenOrderWorfklow {
	var kitchenItems []*proto.KitchenOrderLineItem
	for _, li := range items {
		for i := uint32(0); i < li.Count; i++ {
			kitchenItems = append(kitchenItems, &proto.KitchenOrderLineItem{Name: li.Name})
		}
	}

	return &KitchenOrderWorfklow{Status: &proto.KitchenOrderStatus{Name: name, Open: true, Items: kitchenItems}}
}

func (s *KitchenOrderWorfklow) updateItem(ctx workflow.Context, line uint32, status proto.KitchenOrderItemStatus) error {
	if line < 1 || line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid line item: %d", line)
	}

	// Adjust item number because array is 0-indexed.
	s.Status.Items[line-1].Status = status

	return nil
}

func (s *KitchenOrderWorfklow) checkForOrderCompleted() {
	for _, v := range s.Status.Items {
		if v.Status != proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED {
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
	ch := workflow.GetSignalChannel(ctx, proto.KitchenOrderItemStatusSignal)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal proto.KitchenOrderItemStatusUpdate
		c.Receive(ctx, &signal)

		err = s.updateItem(ctx, signal.Line, signal.Status)
		if err != nil {
			return
		}

		switch signal.Status {
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_STARTED:
			fulfilmentStarted = true
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED:
			fulfilmentStarted = true
			s.checkForOrderCompleted()
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_FAILED:
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

func KitchenOrder(ctx workflow.Context, input *proto.KitchenOrderInput) (*proto.KitchenOrderResult, error) {
	wf := NewKitchenOrderWorkflow(input.Name, input.Items)

	err := workflow.SetQueryHandler(ctx, proto.KitchenOrderStatusQuery, func() (*proto.KitchenOrderStatus, error) {
		return wf.Status, nil
	})
	if err != nil {
		return &proto.KitchenOrderResult{}, err
	}

	err = wf.waitForItems(ctx)

	return &proto.KitchenOrderResult{}, err
}
