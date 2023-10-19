package workflows

import (
	"fmt"
	"time"

	"github.com/temporalio/temporal-cafe/api"
	workflowEnums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

const OrderFulfilmentWindow = 15 * time.Minute

func processPayment(ctx workflow.Context, token string) (*api.ProcessPaymentResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	var result api.ProcessPaymentResult
	err := workflow.ExecuteActivity(ctx, a.ProcessPayment, &api.ProcessPaymentInput{Token: token}).Get(ctx, &result)

	return &result, err
}

func refundPayment(ctx workflow.Context, payment *api.Payment) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	err := workflow.ExecuteActivity(
		ctx,
		a.ProcessPaymentRefund,
		api.ProcessPaymentRefundInput{Payment: payment},
	).Get(ctx, nil)

	return err
}

func fulfilOrder(ctx workflow.Context, items []*api.OrderLineItem) workflow.Future {
	future, settable := workflow.NewFuture(ctx)
	itemsByType := make(map[api.ProductType][]*api.OrderLineItem)

	for _, v := range items {
		itemsByType[v.Type] = append(itemsByType[v.Type], v)
	}

	childCtx, cancelChildren := workflow.WithCancel(ctx)
	childCtx = workflow.WithChildOptions(childCtx, workflow.ChildWorkflowOptions{
		ParentClosePolicy: workflowEnums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})

	workflow.Go(childCtx, func(gctx workflow.Context) {
		var err error

		s := workflow.NewSelector(gctx)

		for t, items := range itemsByType {
			var cw interface{}
			var input interface{}
			switch t {
			case api.ProductType_PRODUCT_TYPE_FOOD:
				cw = KitchenOrder
				input = api.KitchenOrderInput{Items: items}
			case api.ProductType_PRODUCT_TYPE_BEVERAGE:
				cw = BaristaOrder
				input = api.BaristaOrderInput{Items: items}
			}
			s.AddFuture(workflow.ExecuteChildWorkflow(gctx, cw, input), func(f workflow.Future) {
				err = f.Get(gctx, nil)
				if err != nil {
					cancelChildren()
				}
			})
		}

		for i := 0; i < len(itemsByType); i++ {
			s.Select(gctx)
			if err != nil {
				break
			}
		}

		settable.Set(nil, err)
	})

	return future
}

func fulfilmentTimer(ctx workflow.Context) workflow.Future {
	future, settable := workflow.NewFuture(ctx)

	workflow.Go(ctx, func(ctx workflow.Context) {
		ch := workflow.GetSignalChannel(ctx, api.OrderFulfilmentStartedSignal)
		ch.Receive(ctx, nil)
		timer := workflow.NewTimer(ctx, OrderFulfilmentWindow)
		timer.Get(ctx, nil)
		settable.Set(nil, fmt.Errorf("order not fulfilled within window"))
	})

	return future
}

func calculateLoyaltyPoints(input *api.OrderInput) uint32 {
	var i uint32 = 0

	for _, item := range input.Items {
		i += item.Count
	}

	return i
}

func addLoyaltyPoints(ctx workflow.Context, input *api.OrderInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	})

	points := calculateLoyaltyPoints(input)
	err := workflow.ExecuteActivity(
		ctx,
		a.AddLoyaltyPoints,
		api.AddLoyaltyPointsInput{Email: input.Email, Points: points},
	).Get(ctx, nil)

	return err
}

func Order(ctx workflow.Context, input *api.OrderInput) (*api.OrderResult, error) {
	p, err := processPayment(ctx, input.PaymentToken)
	if err != nil {
		return &api.OrderResult{}, err
	}
	defer func() {
		if err != nil {
			refundPayment(ctx, p.Payment)
		}
	}()

	order := fulfilOrder(ctx, input.Items)
	timer := fulfilmentTimer(ctx)

	s := workflow.NewSelector(ctx)
	s.AddFuture(order, func(f workflow.Future) {
		err = f.Get(ctx, nil)
	})
	s.AddFuture(timer, func(f workflow.Future) {
		err = f.Get(ctx, nil)
	})

	s.Select(ctx)
	if err != nil {
		return &api.OrderResult{}, err
	}

	if input.Email != "" {
		_ = addLoyaltyPoints(ctx, input)
	}

	return &api.OrderResult{}, nil
}
