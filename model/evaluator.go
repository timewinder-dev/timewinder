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
	Name     string
	Start    *interp.StackFrame
	Executor *Executor
}

func (ip *InterpProperty) Check(state *interp.State) (PropertyResult, error) {
	// Clone the start frame so we don't modify the original
	frame := ip.Start.Clone()

	val, err := interp.RunToEnd(ip.Executor.Program, state.Globals, frame)
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
