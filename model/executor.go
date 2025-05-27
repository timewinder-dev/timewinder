package model

import (
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// An Executor is the context and entrypoint for runnign a model
type Executor struct {
	Program      *vm.Program
	Properties   []*Property
	InitialState *interp.State
	Threads      []string
	Engine       Engine
}

type Engine interface {
	RunModel() error
}

func (e *Executor) InitializeGlobal() error {
	f := &interp.StackFrame{}
	_, err := interp.RunToEnd(e.Program, nil, f)
	if err != nil {
		return err
	}
	e.InitialState = &interp.State{
		Globals: f,
	}
	return nil
}

func (e *Executor) SpawnThread(name string, entrypoint string) error {
	f, err := interp.FunctionCallFromString(e.Program, e.InitialState.Globals, entrypoint)
	if err != nil {
		return err
	}
	e.InitialState.AddThread(f)
	e.Threads = append(e.Threads, name)
	return nil
}

func (e *Executor) InitEngine() error {
	e.Engine = InitSingleThread(e)
	return nil
}

func (e *Executor) RunModel() error {
	return e.Engine.RunModel()
}
