package model

import (
	"fmt"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

type Property interface {
	Check(state *interp.State) (bool, error)
}

type InterpProperty struct {
	Name     string
	Start    *interp.StackFrame
	Executor *Executor
}

func (ip *InterpProperty) Check(state *interp.State) (bool, error) {
	val, err := interp.RunToEnd(ip.Executor.Program, state.Globals, ip.Start)
	if err != nil {
		return false, err
	}
	if _, ok := val.Cmp(vm.None); ok {
		return false, fmt.Errorf("Property %s: check is returning None", ip.Name)
	}
	return val.AsBool(), nil
}
