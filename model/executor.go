package model

import (
	"io"

	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// PropertyViolation represents a property that failed at a specific state
type PropertyViolation struct {
	PropertyName string
	Message      string
	StateHash    cas.Hash
	Depth        int
	StateNumber  int
	Trace        []TraceStep
	State        *interp.State // The actual state that violated the property
	Program      *vm.Program   // The program being checked
	ThreadID     int           // Which thread caused the violation (-1 for initial state)
	ThreadName   string        // Name of the thread that caused the violation
	ShowDetails  bool          // Whether to show detailed trace reconstruction
	CAS          *cas.MemoryCAS // For retrieving states during trace reconstruction
}

// ModelStatistics holds statistics about the model checking run
type ModelStatistics struct {
	TotalTransitions int
	UniqueStates     int
	DuplicateStates  int
	MaxDepth         int
	ViolationCount   int
}

// ModelResult holds the result of a model checking run
type ModelResult struct {
	Statistics ModelStatistics
	Violations []PropertyViolation
	Success    bool // True if no violations found
}

// An Executor is the context and entrypoint for runnign a model
type Executor struct {
	Program       *vm.Program
	Properties    []Property
	InitialState  *interp.State
	Engine        Engine
	Spec          *Spec
	Threads       []string
	DebugWriter   io.Writer
	CAS           *cas.MemoryCAS
	VisitedStates map[cas.Hash]bool
	KeepGoing     bool
	ShowDetails   bool                // Show detailed trace reconstruction
	Violations    []PropertyViolation // Track all violations found
}

type Engine interface {
	RunModel() (*ModelResult, error)
}

func (e *Executor) Initialize() error {
	err := e.initializeGlobal()
	if err != nil {
		return err
	}

	// Initialize CAS and visited states tracking
	e.CAS = cas.NewMemoryCAS()
	e.VisitedStates = make(map[cas.Hash]bool)

	err = e.initializeProperties()
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

	// Inject builtin functions into global scope BEFORE running global code
	for name, builtin := range vm.AllBuiltins {
		f.StoreVar(name, builtin)
	}

	_, err := interp.RunToEnd(e.Program, nil, f)
	if err != nil {
		return err
	}

	e.InitialState = &interp.State{
		Globals: f,
	}
	return nil
}

func (e *Executor) initializeProperties() error {
	// Initialize stack frames for each property
	for _, prop := range e.Properties {
		if ip, ok := prop.(*InterpProperty); ok {
			// Get the property function name from the spec
			propSpec := e.Spec.Properties[ip.Name]
			callExpr := propSpec.Always + "()"

			// Create a stack frame to call the property function
			f, err := interp.FunctionCallFromString(e.Program, e.InitialState.Globals, callExpr)
			if err != nil {
				return err
			}
			ip.Start = f
		}
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

func (e *Executor) RunModel() (*ModelResult, error) {
	return e.Engine.RunModel()
}
