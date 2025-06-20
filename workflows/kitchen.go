package workflows

import (
	"errors"
	"fmt"

	"github.com/temporalio/temporal-cafe/proto"
	"go.temporal.io/sdk/workflow"
)

type KitchenOrderWorfklow struct {
	Status           *proto.KitchenOrderStatus
	fulfilmentSignal bool
	err              error
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
	if err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		proto.KitchenOrderItemStatusSignal,
		s.updateItemHandler,
		workflow.UpdateHandlerOptions{Validator: func(ctx workflow.Context, req *proto.KitchenOrderItemStatusUpdate) error {
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

func (s *KitchenOrderWorfklow) updateItemValidator(req *proto.KitchenOrderItemStatusUpdate) error {
	if req.Line < 1 || req.Line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid order item line number %d", req.Line)
	}

	return nil
}

func (s *KitchenOrderWorfklow) updateItemHandler(ctx workflow.Context, req *proto.KitchenOrderItemStatusUpdate) (*proto.KitchenOrderStatus, error) {
	// Adjust item line number because array is 0-indexed.
	s.Status.Items[req.Line-1].Status = req.Status

	switch req.Status {
	case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_STARTED:
		if err := s.signalFulfilmentStarted(ctx); err != nil {
			return nil, err
		}
	case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_COMPLETED:
		if s.isOrderCompleted() {
			s.Status.Open = false
		}
	case proto.KitchenOrderItemStatus_KITCHEN_ORDER_ITEM_STATUS_FAILED:
		s.Status.Open = false
		s.err = fmt.Errorf("item %d (%s) failed", req.Line, s.Status.Items[req.Line-1].Name)
	}

	return s.Status, nil
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

func (s *KitchenOrderWorfklow) updateItem(ctx workflow.Context, line uint32, status proto.KitchenOrderItemStatus) error {
	if line < 1 || line > uint32(len(s.Status.Items)) {
		return fmt.Errorf("invalid line item: %d", line)
	}

	// Adjust item number because array is 0-indexed.
	s.Status.Items[line-1].Status = status

	return nil
}
