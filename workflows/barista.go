package workflows

import (
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type BaristaOrderWorfklow struct {
	Status *proto.BaristaOrderStatus
}

func (s *BaristaOrderWorfklow) signalFulfilmentStarted(ctx workflow.Context) error {
	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	signal := workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, proto.OrderFulfilmentStartedSignal, nil)
	return signal.Get(ctx, nil)
}

func NewBaristaOrderWorkflow(name string, items []*proto.OrderLineItem) *BaristaOrderWorfklow {
	var baristaItems []*proto.BaristaOrderLineItem
	for _, li := range items {
		for i := uint32(0); i < li.Count; i++ {
			baristaItems = append(baristaItems, &proto.BaristaOrderLineItem{Name: li.Name})
		}
	}

	return &BaristaOrderWorfklow{Status: &proto.BaristaOrderStatus{Name: name, Open: true, Items: baristaItems}}
}

func (s *BaristaOrderWorfklow) updateItem(ctx workflow.Context, line uint32, status proto.BaristaOrderItemStatus) error {
	if line < 1 || line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid line item: %d", line)
	}

	// Adjust item number because array is 0-indexed.
	s.Status.Items[line-1].Status = status

	return nil
}

func (s *BaristaOrderWorfklow) checkForOrderCompleted() {
	for _, v := range s.Status.Items {
		if v.Status != proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED {
			return
		}
	}
	s.Status.Open = false
}

func (s *BaristaOrderWorfklow) waitForItems(ctx workflow.Context) error {
	sel := workflow.NewSelector(ctx)

	var err error
	var fulfilmentStarted = false
	var fulfilmentSignalled = false

	// Listen for signals from Barista staff
	ch := workflow.GetSignalChannel(ctx, proto.BaristaOrderItemStatusSignal)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var signal proto.BaristaOrderItemStatusUpdate
		c.Receive(ctx, &signal)

		err = s.updateItem(ctx, signal.Line, signal.Status)
		if err != nil {
			return
		}

		switch signal.Status {
		case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_STARTED:
			fulfilmentStarted = true
		case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED:
			fulfilmentStarted = true
			s.checkForOrderCompleted()
		case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_FAILED:
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

func BaristaOrder(ctx workflow.Context, input *proto.BaristaOrderInput) (*proto.BaristaOrderResult, error) {
	wf := NewBaristaOrderWorkflow(input.Name, input.Items)

	err := workflow.SetQueryHandler(ctx, proto.BaristaOrderStatusQuery, func() (*proto.BaristaOrderStatus, error) {
		return wf.Status, nil
	})
	if err != nil {
		return &proto.BaristaOrderResult{}, err
	}

	err = wf.waitForItems(ctx)

	return &proto.BaristaOrderResult{}, err
}
