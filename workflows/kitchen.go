package workflows

import (
	"fmt"

	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/workflow"
)

const KitchenOrderStatusQueryName = "getStatus"

const KitchenOrderItemStartedSignalName = "kitchen-item-started"
const KitchenOrderItemCompletedSignalName = "kitchen-item-completed"
const KitchenOrderItemFailedSignalName = "kitchen-item-failed"

const KitchenOrderItemStatusPending = "pending"
const KitchenOrderItemStatusStarted = "started"
const KitchenOrderItemStatusCompleted = "completed"
const KitchenOrderItemStatusFailed = "failed"

type KitchenOrderLineItem struct {
	Name   string
	Status string
}

type KitchenOrderWorkflowInput struct {
	Items []api.OrderLineItem
}

type KitchenOrderItemStartedSignal struct {
	Line int
}

type KitchenOrderItemCompletedSignal struct {
	Line int
}

type KitchenOrderItemFailedSignal struct {
	Line int
}

type KitchenOrderWorfklowStatus struct {
	Open          bool
	Items         []KitchenOrderLineItem
	startNotified bool
}

type KitchenOrderWorfklowResult struct {
}

func (s *KitchenOrderWorfklowStatus) signalOrderStarted(ctx workflow.Context) {
	if s.startNotified {
		return
	}

	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, api.OrderStartedSignalName, nil)

	s.startNotified = true
}

func NewKitchenOrderWorkflowStatus(items []api.OrderLineItem) *KitchenOrderWorfklowStatus {
	var kitchenItems []KitchenOrderLineItem
	for _, li := range items {
		for i := 0; i < li.Count; i++ {
			kitchenItems = append(kitchenItems, KitchenOrderLineItem{Name: li.Name, Status: KitchenOrderItemStatusPending})
		}
	}

	return &KitchenOrderWorfklowStatus{Open: true, Items: kitchenItems}
}

func (s *KitchenOrderWorfklowStatus) updateItem(ctx workflow.Context, line int, status string) {
	if line < 1 || line > len(s.Items) {
		return
	}

	switch status {
	case KitchenOrderItemStatusPending:
	case KitchenOrderItemStatusStarted:
	case KitchenOrderItemStatusFailed:
	case KitchenOrderItemStatusCompleted:
	default:
		return
	}

	// Adjust item number because array is 0-indexed.
	s.Items[line-1].Status = status
}

func (s *KitchenOrderWorfklowStatus) checkForOrderCompleted() {
	for _, v := range s.Items {
		if v.Status != KitchenOrderItemStatusCompleted {
			return
		}
	}
	s.Open = false
}

func (s *KitchenOrderWorfklowStatus) waitForItems(ctx workflow.Context) error {
	sel := workflow.NewSelector(ctx)

	var err error

	// Listen for signals from Kitchen staff
	ch := workflow.GetSignalChannel(ctx, KitchenOrderItemStartedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var startedSignal KitchenOrderItemStartedSignal
		c.Receive(ctx, &startedSignal)

		s.updateItem(ctx, startedSignal.Line, KitchenOrderItemStatusStarted)
		s.signalOrderStarted(ctx)
	})
	ch = workflow.GetSignalChannel(ctx, KitchenOrderItemCompletedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var completedSignal KitchenOrderItemCompletedSignal
		c.Receive(ctx, &completedSignal)

		s.updateItem(ctx, completedSignal.Line, KitchenOrderItemStatusCompleted)
		s.checkForOrderCompleted()
	})
	ch = workflow.GetSignalChannel(ctx, KitchenOrderItemFailedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var failedSignal KitchenOrderItemFailedSignal
		c.Receive(ctx, &failedSignal)

		s.updateItem(ctx, failedSignal.Line, KitchenOrderItemStatusFailed)
		err = fmt.Errorf("item %s failed", s.Items[failedSignal.Line].Name)

		s.Open = false
	})

	// Listen for Workflow cancellation
	sel.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, _ bool) {
		s.Open = false
	})

	for s.Open {
		sel.Select(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func KitchenOrder(ctx workflow.Context, input *KitchenOrderWorkflowInput) (*KitchenOrderWorfklowResult, error) {
	status := NewKitchenOrderWorkflowStatus(input.Items)

	err := workflow.SetQueryHandler(ctx, KitchenOrderStatusQueryName, func() (*KitchenOrderWorfklowStatus, error) {
		return status, nil
	})
	if err != nil {
		return &KitchenOrderWorfklowResult{}, err
	}

	err = status.waitForItems(ctx)

	return &KitchenOrderWorfklowResult{}, err
}
