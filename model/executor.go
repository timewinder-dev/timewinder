package model

import "github.com/timewinder-dev/timewinder/vm"

// An Executor is the context and entrypoint for runnign a model
type Executor struct {
	Program    *vm.Program
	Properties []*Property
	states     []*State
}

func (e *Executor) Initialize() error {

}
