package workflows

import (
	"errors"
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/workflow"
)

type BaristaOrderWorfklow struct {
	Status           *proto.BaristaOrderStatus
	fulfilmentSignal bool
	err              error
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

func (s *BaristaOrderWorfklow) waitForItems(ctx workflow.Context) error {
	if err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		proto.BaristaOrderItemStatusSignal,
		s.updateItemHandler,
		workflow.UpdateHandlerOptions{Validator: func(ctx workflow.Context, req *proto.BaristaOrderItemStatusUpdate) error {
			return s.updateItemValidator(req)
		}},
	); err != nil {
		return err
	}

	err := workflow.Await(ctx, func() bool {
		return !s.Status.Open
	})
	if err == nil {
		err = s.err
	}

	if errors.Is(err, workflow.ErrCanceled) {
		return nil
	}

	return err
}

func (s *BaristaOrderWorfklow) updateItemValidator(req *proto.BaristaOrderItemStatusUpdate) error {
	if req.Line < 1 || req.Line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid order item line number %d", req.Line)
	}

	return nil
}

func (s *BaristaOrderWorfklow) updateItemHandler(ctx workflow.Context, req *proto.BaristaOrderItemStatusUpdate) (*proto.BaristaOrderStatus, error) {
	// Adjust item line number because array is 0-indexed.
	s.Status.Items[req.Line-1].Status = req.Status

	switch req.Status {
	case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_STARTED:
		if err := s.signalFulfilmentStarted(ctx); err != nil {
			return nil, err
		}
	case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED:
		if s.isOrderCompleted() {
			s.Status.Open = false
		}
	case proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_FAILED:
		s.Status.Open = false
		s.err = fmt.Errorf("item %d (%s) failed", req.Line, s.Status.Items[req.Line-1].Name)
	}

	return s.Status, nil
}

func (s *BaristaOrderWorfklow) isOrderCompleted() bool {
	for _, v := range s.Status.Items {
		if v.Status != proto.BaristaOrderItemStatus_BARISTA_ORDER_ITEM_STATUS_COMPLETED {
			return false
		}
	}
	return true
}

func (s *BaristaOrderWorfklow) signalFulfilmentStarted(ctx workflow.Context) error {
	if s.fulfilmentSignal {
		return nil
	}

	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	if we == nil {
		return nil
	}
	signal := workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, proto.OrderFulfilmentStartedSignal, nil)
	if err := signal.Get(ctx, nil); err != nil {
		return err
	}
	s.fulfilmentSignal = true

	return nil
}
