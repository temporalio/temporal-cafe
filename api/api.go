package api

import "fmt"

const OrderStartedSignalName = "order-started"
const OrderLineItemTypeFood = "food"
const OrderLineItemTypeBeverage = "beverage"

type OrderWorkflowInput struct {
	Email        string
	PaymentToken string
	Items        []OrderLineItem
}

type OrderWorfklowResult struct {
}

type OrderLineItem struct {
	Name  string
	Type  string
	Count int
}

func CustomerWorkflowID(email string) string {
	return fmt.Sprintf("customer:%s", email)
}

type CustomerPointsBalanceQuery struct {
	Points uint
}

type CustomerPointsAddSignal struct {
	Points uint
}

type CustomerWorkflowInput struct {
	Email string
}

type CustomerWorkflowState struct {
	Points uint
}

const CustomerPointsBalanceQueryName = "customer-points-balance"
const CustomerPointsAddSignalName = "customer-points-add"
