package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/temporal-cafe/api"
	"github.com/temporalio/temporal-cafe/workflows"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCustomerWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Customer)

	input := &api.CustomerInput{
		Email: "test@example.com",
	}

	env.RegisterDelayedCallback(func() {
		env.SetContinueAsNewSuggested(true)

		env.SignalWorkflow(
			api.CustomerLoyaltyPointsEarnedSignal,
			api.CustomerLoyaltyPointsEarned{Points: 1},
		)

		env.SignalWorkflow(
			api.CustomerLoyaltyPointsEarnedSignal,
			api.CustomerLoyaltyPointsEarned{Points: 3},
		)

		env.SignalWorkflow(
			api.CustomerLoyaltyPointsEarnedSignal,
			api.CustomerLoyaltyPointsEarned{Points: 1},
		)
	}, 0)

	env.ExecuteWorkflow(workflows.Customer, input, nil)

	assert.True(t, workflow.IsContinueAsNewError(env.GetWorkflowError()))

	v, err := env.QueryWorkflow(api.CustomerLoyaltyPointsBalanceQuery)
	assert.NoError(t, err)
	var result api.CustomerLoyaltyPointsBalance
	err = v.Get(&result)
	assert.NoError(t, err)

	assert.Equal(t, uint32(workflows.CustomerStartingBalance+5), result.Points)
}

func TestCustomerWorkflowContinue(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflows.Customer)

	input := &api.CustomerInput{
		Email: "test@example.com",
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(
			api.CustomerLoyaltyPointsEarnedSignal,
			api.CustomerLoyaltyPointsEarned{Points: 3},
		)
	}, 0)

	env.ExecuteWorkflow(workflows.Customer, input, &workflows.CustomerWorkflowState{Points: 1})

	v, err := env.QueryWorkflow(api.CustomerLoyaltyPointsBalanceQuery)
	assert.NoError(t, err)
	var result api.CustomerLoyaltyPointsBalance
	err = v.Get(&result)
	assert.NoError(t, err)

	assert.Equal(t, uint32(4), result.Points)
}
