package model

import (
	"fmt"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

type PropertyResult struct {
	Success bool
	Message string
	Name    string
}

type Property interface {
	Check(state *interp.State) (PropertyResult, error)
}

type InterpProperty struct {
	Name       string
	Executor   *Executor
	ExprString string // The property expression to evaluate
}

func (ip *InterpProperty) Check(state *interp.State) (PropertyResult, error) {
	// Compile and execute the expression with the current state's globals
	exprProg, err := vm.CompileExpr(ip.ExprString)
	if err != nil {
		return PropertyResult{}, fmt.Errorf("Property %s: failed to compile expression: %w", ip.Name, err)
	}

	// Create an overlay that uses the expression's main but has access to all functions
	// from the original program (same approach as FunctionCallFromString)
	overlay := &interp.OverlayMain{
		Program: ip.Executor.Program,
		Main:    exprProg.Main,
	}

	frame := &interp.StackFrame{Stack: []vm.Value{}}
	val, err := interp.RunToEnd(overlay, state.Globals, frame)
	if err != nil {
		// Execution error - something went wrong running the property check
		return PropertyResult{}, err
	}
	if _, ok := val.Cmp(vm.None); ok {
		// Property returned None - this is an error in the property definition
		return PropertyResult{}, fmt.Errorf("Property %s: check is returning None", ip.Name)
	}

	result := val.AsBool()
	if result {
		return PropertyResult{
			Success: true,
			Name:    ip.Name,
			Message: fmt.Sprintf("Property %s satisfied", ip.Name),
		}, nil
	} else {
		return PropertyResult{
			Success: false,
			Name:    ip.Name,
			Message: fmt.Sprintf("Property %s violated: returned false", ip.Name),
		}, nil
	}
}

// TemporalOperator represents the type of temporal logic operator
type TemporalOperator int

const (
	Always TemporalOperator = iota
	Eventually
	EventuallyAlways
	AlwaysEventually
)

func (op TemporalOperator) String() string {
	switch op {
	case Always:
		return "Always"
	case Eventually:
		return "Eventually"
	case EventuallyAlways:
		return "EventuallyAlways"
	case AlwaysEventually:
		return "AlwaysEventually"
	default:
		return "Unknown"
	}
}

// TemporalConstraint wraps a Property with temporal semantics
type TemporalConstraint struct {
	Name     string
	Operator TemporalOperator // The temporal operator (Always, Eventually, etc.)
	Property Property          // The underlying boolean property to evaluate
}
