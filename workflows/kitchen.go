package workflows

import (
	"errors"
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type KitchenOrderWorfklow struct {
	Status *proto.KitchenOrderStatus
	err    error
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

func (s *KitchenOrderWorfklow) waitForItems(ctx workflow.Context) error {
	sel := workflow.NewSelector(ctx)

	var fulfilmentStarted = false
	var fulfilmentSignalled = false

	// Listen for signals from Kitchen staff
	ch := workflow.GetSignalChannel(ctx, proto.KitchenOrderItemStatusSignal)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal proto.KitchenOrderItemStatusUpdate
		c.Receive(ctx, &signal)

		if err := s.updateItem(ctx, signal.Line, signal.Status); err != nil {
			s.err = err
			return
		}

		switch signal.Status {
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_STARTED:
			fulfilmentStarted = true
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED:
			fulfilmentStarted = true
			if s.isOrderCompleted() {
				s.Status.Open = false
			}
		case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_FAILED:
			s.err = fmt.Errorf("item %d failed", signal.Line)
			s.Status.Open = false
		}
	})

	// Listen for Workflow cancellation
	sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {
		s.err = temporal.NewCanceledError()
	})

	for s.Status.Open {
		sel.Select(ctx)
		if s.err != nil {
			if errors.Is(s.err, workflow.ErrCanceled) {
				return nil
			}
			return s.err
		}
		if fulfilmentStarted && !fulfilmentSignalled {
			if err := s.signalFulfilmentStarted(ctx); err != nil {
				return err
			}
			fulfilmentSignalled = true
		}
	}

	return nil
}

func (s *KitchenOrderWorfklow) updateItem(ctx workflow.Context, line uint32, status proto.KitchenOrderItemStatus) error {
	if line < 1 || line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid line item: %d", line)
	}

	// Adjust item number because array is 0-indexed.
	s.Status.Items[line-1].Status = status

	return nil
}

func (s *KitchenOrderWorfklow) isOrderCompleted() bool {
	for _, v := range s.Status.Items {
		if v.Status != proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED {
			return false
		}
	}
	return true
}

func (s *KitchenOrderWorfklow) signalFulfilmentStarted(ctx workflow.Context) error {
	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	if we == nil {
		return nil
	}
	signal := workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, proto.OrderFulfilmentStartedSignal, nil)
	return signal.Get(ctx, nil)
}
