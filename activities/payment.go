package activities

import (
	"context"

	"github.com/temporalio/temporal-cafe/proto"
)

func (a *Activities) ProcessPayment(ctx context.Context, input *proto.ProcessPaymentInput) (*proto.ProcessPaymentResult, error) {
	return &proto.ProcessPaymentResult{Payment: &proto.Payment{Authcode: "x"}}, nil
}

func (a *Activities) ProcessPaymentRefund(ctx context.Context, input *proto.ProcessPaymentRefundInput) (*proto.ProcessPaymentRefundResult, error) {
	return &proto.ProcessPaymentRefundResult{}, nil
}
