package activities

import "context"

type Payment struct {
	AuthCode string
}

type ProcessPaymentInput struct {
	Token string
}

type ProcessPaymentResult struct {
	Payment Payment
}

type ProcessPaymentRefundInput struct {
	Payment Payment
}

type ProcessPaymentRefundResult struct {
}

func (a *Activities) ProcessPayment(ctx context.Context, input *ProcessPaymentInput) (*ProcessPaymentResult, error) {
	return &ProcessPaymentResult{Payment: Payment{AuthCode: "x"}}, nil
}

func (a *Activities) ProcessPaymentRefund(ctx context.Context, input *ProcessPaymentRefundInput) (*ProcessPaymentRefundResult, error) {
	return &ProcessPaymentRefundResult{}, nil
}
