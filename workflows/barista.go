package workflows

import (
	"fmt"

	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/workflow"
)

const BaristaOrderStatusQueryName = "getStatus"

const BaristaOrderItemStartedSignalName = "barista-item-started"
const BaristaOrderItemCompletedSignalName = "barista-item-completed"
const BaristaOrderItemFailedSignalName = "barista-item-failed"

const BaristaOrderItemStatusPending = "pending"
const BaristaOrderItemStatusStarted = "started"
const BaristaOrderItemStatusCompleted = "completed"
const BaristaOrderItemStatusFailed = "failed"

type BaristaOrderLineItem struct {
	Name   string
	Status string
}

type BaristaOrderWorkflowInput struct {
	Items []api.OrderLineItem
}

type BaristaOrderItemStartedSignal struct {
	Line int
}

type BaristaOrderItemCompletedSignal struct {
	Line int
}

type BaristaOrderItemFailedSignal struct {
	Line int
}

type BaristaOrderWorfklowStatus struct {
	Open          bool
	Items         []BaristaOrderLineItem
	startNotified bool
}

type BaristaOrderWorfklowResult struct {
}

func (s *BaristaOrderWorfklowStatus) signalOrderStarted(ctx workflow.Context) {
	if s.startNotified {
		return
	}

	we := workflow.GetInfo(ctx).ParentWorkflowExecution
	workflow.SignalExternalWorkflow(ctx, we.ID, we.RunID, api.OrderStartedSignalName, nil)

	s.startNotified = true
}

func NewBaristaOrderWorkflowStatus(items []api.OrderLineItem) *BaristaOrderWorfklowStatus {
	var baristaItems []BaristaOrderLineItem
	for _, li := range items {
		for i := 0; i < li.Count; i++ {
			baristaItems = append(baristaItems, BaristaOrderLineItem{Name: li.Name, Status: BaristaOrderItemStatusPending})
		}
	}

	return &BaristaOrderWorfklowStatus{Open: true, Items: baristaItems}
}

func (s *BaristaOrderWorfklowStatus) updateItem(ctx workflow.Context, line int, status string) {
	if line < 1 || line > len(s.Items) {
		return
	}

	switch status {
	case BaristaOrderItemStatusPending:
	case BaristaOrderItemStatusStarted:
	case BaristaOrderItemStatusFailed:
	case BaristaOrderItemStatusCompleted:
	default:
		return
	}

	// Adjust item number because array is 0-indexed.
	s.Items[line-1].Status = status
}

func (s *BaristaOrderWorfklowStatus) checkForOrderCompleted() {
	for _, v := range s.Items {
		if v.Status != BaristaOrderItemStatusCompleted {
			return
		}
	}
	s.Open = false
}

func (s *BaristaOrderWorfklowStatus) waitForItems(ctx workflow.Context) error {
	sel := workflow.NewSelector(ctx)

	var err error

	// Listen for signals from Barista staff
	ch := workflow.GetSignalChannel(ctx, BaristaOrderItemStartedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var startedSignal BaristaOrderItemStartedSignal
		c.Receive(ctx, &startedSignal)

		s.updateItem(ctx, startedSignal.Line, BaristaOrderItemStatusStarted)
		s.signalOrderStarted(ctx)
	})
	ch = workflow.GetSignalChannel(ctx, BaristaOrderItemCompletedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var completedSignal BaristaOrderItemCompletedSignal
		c.Receive(ctx, &completedSignal)

		s.updateItem(ctx, completedSignal.Line, BaristaOrderItemStatusCompleted)
		s.checkForOrderCompleted()
	})
	ch = workflow.GetSignalChannel(ctx, BaristaOrderItemFailedSignalName)
	sel.AddReceive(ch, func(c workflow.ReceiveChannel, _ bool) {
		var failedSignal BaristaOrderItemFailedSignal
		c.Receive(ctx, &failedSignal)

		s.updateItem(ctx, failedSignal.Line, BaristaOrderItemStatusFailed)
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

func BaristaOrder(ctx workflow.Context, input *BaristaOrderWorkflowInput) (*BaristaOrderWorfklowResult, error) {
	status := NewBaristaOrderWorkflowStatus(input.Items)

	err := workflow.SetQueryHandler(ctx, BaristaOrderStatusQueryName, func() (*BaristaOrderWorfklowStatus, error) {
		return status, nil
	})
	if err != nil {
		return &BaristaOrderWorfklowResult{}, err
	}

	err = status.waitForItems(ctx)

	return &BaristaOrderWorfklowResult{}, err
}
