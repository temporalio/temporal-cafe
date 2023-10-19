package activities

import (
	"context"

	"github.com/temporalio/temporal-cafe/api"
)

func (a *Activities) ProcessPayment(ctx context.Context, input *api.ProcessPaymentInput) (*api.ProcessPaymentResult, error) {
	return &api.ProcessPaymentResult{Payment: &api.Payment{Authcode: "x"}}, nil
}

func (a *Activities) ProcessPaymentRefund(ctx context.Context, input *api.ProcessPaymentRefundInput) (*api.ProcessPaymentRefundResult, error) {
	return &api.ProcessPaymentRefundResult{}, nil
}
