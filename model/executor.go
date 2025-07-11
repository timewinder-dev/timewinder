package model

import (
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// An Executor is the context and entrypoint for runnign a model
type Executor struct {
	Program      *vm.Program
	Properties   []Property
	InitialState *interp.State
	Engine       Engine
	Spec         *Spec
	Threads      []string
}

type Engine interface {
	RunModel() error
}

func (e *Executor) Initialize() error {
	err := e.initializeGlobal()
	if err != nil {
		return err
	}
	for name, s := range e.Spec.Threads {
		err = e.spawnThread(name, s.Entrypoint)
		if err != nil {
			return err
		}
	}
	err = e.initEngine()
	if err != nil {
		return err
	}
	return nil
}

func (e *Executor) initializeGlobal() error {
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

func (e *Executor) spawnThread(name string, entrypoint string) error {
	f, err := interp.FunctionCallFromString(e.Program, e.InitialState.Globals, entrypoint)
	if err != nil {
		return err
	}
	e.InitialState.AddThread(f)
	e.Threads = append(e.Threads, name)
	return nil
}

func (e *Executor) initEngine() error {
	var err error
	e.Engine, err = InitSingleThread(e)
	if err != nil {
		return err
	}
	return nil
}

func (e *Executor) RunModel() error {
	return e.Engine.RunModel()
}
