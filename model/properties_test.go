package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// Helper function to create a test state with global variables
func createTestState(globals map[string]vm.Value) *interp.State {
	frame := &interp.StackFrame{
		Variables: make(map[string]vm.Value),
	}
	for k, v := range globals {
		frame.Variables[k] = v
	}
	return &interp.State{
		Globals: frame,
	}
}

// Helper function to compile a simple property expression and create an InterpProperty
func createTestProperty(t *testing.T, name string, expression string, globals map[string]vm.Value) (*InterpProperty, *interp.State) {
	// Create a simple Starlark program that defines the property function
	source := "def " + name + "():\n    return " + expression

	prog, err := vm.CompileLiteral(source)
	require.NoError(t, err, "Failed to compile property function")

	// Initialize global state
	frame := &interp.StackFrame{}
	_, err = interp.RunToEnd(prog, nil, frame)
	require.NoError(t, err, "Failed to initialize globals")

	state := &interp.State{Globals: frame}

	// Add any additional global variables
	if len(globals) > 0 {
		if state.Globals.Variables == nil {
			state.Globals.Variables = make(map[string]vm.Value)
		}
		for k, v := range globals {
			state.Globals.Variables[k] = v
		}
	}

	// Create executor
	executor := &Executor{
		Program: prog,
	}

	prop := &InterpProperty{
		Name:       name,
		ExprString: name + "()",
		Executor:   executor,
	}

	return prop, state
}

func TestInterpProperty_Check_ReturnsTrue(t *testing.T) {
	prop, state := createTestProperty(t, "always_true", "True", nil)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "always_true", result.Name)
	assert.Contains(t, result.Message, "satisfied")
}

func TestInterpProperty_Check_ReturnsFalse(t *testing.T) {
	prop, state := createTestProperty(t, "always_false", "False", nil)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "always_false", result.Name)
	assert.Contains(t, result.Message, "violated")
}

func TestInterpProperty_Check_WithGlobalVariables(t *testing.T) {
	globals := map[string]vm.Value{
		"balance": vm.IntValue(100),
	}

	prop, state := createTestProperty(t, "positive_balance", "balance > 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "positive_balance", result.Name)
}

func TestInterpProperty_Check_WithGlobalVariables_Violation(t *testing.T) {
	globals := map[string]vm.Value{
		"balance": vm.IntValue(-50),
	}

	prop, state := createTestProperty(t, "no_overdraft", "balance >= 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "no_overdraft", result.Name)
	assert.Contains(t, result.Message, "violated")
}

func TestInterpProperty_Check_ComplexExpression(t *testing.T) {
	globals := map[string]vm.Value{
		"x": vm.IntValue(5),
		"y": vm.IntValue(10),
	}

	prop, state := createTestProperty(t, "complex_prop", "(x + y) == 15", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestInterpProperty_Check_WithLogicalOperators(t *testing.T) {
	globals := map[string]vm.Value{
		"a": vm.IntValue(5),
		"b": vm.IntValue(10),
		"c": vm.IntValue(15),
	}

	prop, state := createTestProperty(t, "logical_prop", "a < b and b < c", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCheckProperties_AllPass(t *testing.T) {
	globals := map[string]vm.Value{
		"balance": vm.IntValue(100),
	}

	prop1, state := createTestProperty(t, "prop1", "balance > 0", globals)
	prop2, _ := createTestProperty(t, "prop2", "balance < 1000", globals)

	properties := []Property{prop1, prop2}

	err := CheckProperties(state, properties)

	assert.NoError(t, err)
}

func TestCheckProperties_OneViolation(t *testing.T) {
	globals := map[string]vm.Value{
		"balance": vm.IntValue(-50),
	}

	prop1, state := createTestProperty(t, "positive", "balance > 0", globals)
	prop2, _ := createTestProperty(t, "valid", "True", globals)

	properties := []Property{prop1, prop2}

	err := CheckProperties(state, properties)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Property violation")
	assert.Contains(t, err.Error(), "positive")
}

func TestCheckProperties_MultipleProperties_FirstFails(t *testing.T) {
	globals := map[string]vm.Value{
		"x": vm.IntValue(5),
	}

	prop1, state := createTestProperty(t, "fails", "x > 10", globals)
	prop2, _ := createTestProperty(t, "passes", "x > 0", globals)

	properties := []Property{prop1, prop2}

	err := CheckProperties(state, properties)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fails")
}

func TestCheckProperties_EmptyList(t *testing.T) {
	state := createTestState(nil)
	properties := []Property{}

	err := CheckProperties(state, properties)

	assert.NoError(t, err)
}

func TestCheckProperties_StateNotModified(t *testing.T) {
	globals := map[string]vm.Value{
		"balance": vm.IntValue(100),
	}

	prop, state := createTestProperty(t, "check_balance", "balance > 0", globals)
	originalBalance := state.Globals.Variables["balance"]

	properties := []Property{prop}
	err := CheckProperties(state, properties)

	require.NoError(t, err)
	// Verify the state wasn't modified
	assert.Equal(t, originalBalance, state.Globals.Variables["balance"])
}

// Test with different value types
func TestInterpProperty_Check_StringComparison(t *testing.T) {
	globals := map[string]vm.Value{
		"status": vm.StrValue("active"),
	}

	prop, state := createTestProperty(t, "is_active", "status == \"active\"", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestInterpProperty_Check_BooleanValue(t *testing.T) {
	globals := map[string]vm.Value{
		"enabled": vm.BoolTrue,
	}

	prop, state := createTestProperty(t, "is_enabled", "enabled", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestInterpProperty_Check_NegatedBoolean(t *testing.T) {
	globals := map[string]vm.Value{
		"disabled": vm.BoolFalse,
	}

	prop, state := createTestProperty(t, "not_disabled", "not disabled", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

// Test boundary conditions
func TestInterpProperty_Check_ZeroValue(t *testing.T) {
	globals := map[string]vm.Value{
		"count": vm.IntValue(0),
	}

	prop, state := createTestProperty(t, "is_zero", "count == 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestInterpProperty_Check_NegativeNumbers(t *testing.T) {
	globals := map[string]vm.Value{
		"debt": vm.IntValue(-100),
	}

	prop, state := createTestProperty(t, "has_debt", "debt < 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

// Test that None return is caught as an error
func TestInterpProperty_Check_ReturnsNone_Error(t *testing.T) {
	prop, state := createTestProperty(t, "returns_none", "None", nil)

	_, err := prop.Check(state)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "returning None")
}

// Integration test: simulate a banking scenario
func TestCheckProperties_BankingScenario(t *testing.T) {
	// Create a state with two accounts
	globals := map[string]vm.Value{
		"account1": vm.IntValue(100),
		"account2": vm.IntValue(200),
		"total":    vm.IntValue(300),
	}

	// Create properties
	noOverdraft1, state := createTestProperty(t, "no_overdraft1", "account1 >= 0", globals)
	noOverdraft2, _ := createTestProperty(t, "no_overdraft2", "account2 >= 0", globals)
	conservesMoney, _ := createTestProperty(t, "conserves_money", "account1 + account2 == total", globals)

	properties := []Property{noOverdraft1, noOverdraft2, conservesMoney}

	err := CheckProperties(state, properties)

	assert.NoError(t, err)
}

func TestCheckProperties_BankingScenario_Violation(t *testing.T) {
	// Create a state where account1 has overdraft
	globals := map[string]vm.Value{
		"account1": vm.IntValue(-50),
		"account2": vm.IntValue(200),
	}

	noOverdraft1, state := createTestProperty(t, "no_overdraft1", "account1 >= 0", globals)
	noOverdraft2, _ := createTestProperty(t, "no_overdraft2", "account2 >= 0", globals)

	properties := []Property{noOverdraft1, noOverdraft2}

	err := CheckProperties(state, properties)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no_overdraft1")
}

// Test with struct/dict values
func TestInterpProperty_Check_WithStruct(t *testing.T) {
	account := vm.StructValue{
		"balance": vm.IntValue(100),
		"owner":   vm.StrValue("Alice"),
	}
	globals := map[string]vm.Value{
		"account": account,
	}

	prop, state := createTestProperty(t, "valid_account", "account[\"balance\"] > 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestInterpProperty_Check_WithArray(t *testing.T) {
	accounts := vm.ArrayValue{
		vm.IntValue(100),
		vm.IntValue(200),
		vm.IntValue(300),
	}
	globals := map[string]vm.Value{
		"accounts": accounts,
	}

	prop, state := createTestProperty(t, "first_positive", "accounts[0] > 0", globals)

	result, err := prop.Check(state)

	require.NoError(t, err)
	assert.True(t, result.Success)
}
