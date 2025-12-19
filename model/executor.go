package model

import (
	"fmt"
	"io"

	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// PropertyViolation represents a property that failed at a specific state
type PropertyViolation struct {
	PropertyName string
	PropertyType string         // Type of property: "Always", "EventuallyAlways", etc.
	Message      string
	StateHash    cas.Hash
	Depth        int
	StateNumber  int
	Trace        []TraceStep
	State        *interp.State  // The actual state that violated the property
	Program      *vm.Program    // The program being checked
	ThreadID     int            // Which thread caused the violation (-1 for initial state)
	ThreadName   string         // Name of the thread that caused the violation
	ShowDetails  bool           // Whether to show detailed trace reconstruction
	CAS          cas.CAS        // For retrieving states during trace reconstruction
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
	Program            *vm.Program
	Properties         []Property
	TemporalConstraints []TemporalConstraint // Temporal properties to check
	InitialState       *interp.State
	Engine             Engine
	Spec               *Spec
	Threads            []string
	DebugWriter        io.Writer
	Reporter           Reporter            // Progress reporter
	CAS                cas.CAS
	VisitedStates      map[cas.Hash]bool
	WeakStateHistory   map[cas.Hash][]int           // Track depths where weak states were seen
	WeakStateSamples   map[cas.Hash]*interp.State   // Sample state for each weak hash
	KeepGoing          bool
	ShowDetails        bool                // Show detailed trace reconstruction
	Violations         []PropertyViolation // Track all violations found
	NoDeadlocks        bool                // Disable deadlock detection
	MaxDepth           int                 // Maximum depth to explore (0 = unlimited)
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
	e.VisitedStates = make(map[cas.Hash]bool)
	e.WeakStateHistory = make(map[cas.Hash][]int)
	e.WeakStateSamples = make(map[cas.Hash]*interp.State)

	err = e.initializeProperties()
	if err != nil {
		return err
	}
	for name, s := range e.Spec.Threads {
		err = e.spawnThread(name, s.Entrypoint, s.Replicas)
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
	f := &interp.StackFrame{
		Stack: []vm.Value{},
	}

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
	// Initialize stack frames for each property using TemporalConstraints
	for _, constraint := range e.TemporalConstraints {
		if ip, ok := constraint.Property.(*InterpProperty); ok {
			// Get the property function name from the spec based on operator
			propSpec := e.Spec.Properties[constraint.Name]
			var funcName string

			switch constraint.Operator {
			case Always:
				funcName = propSpec.Always
			case EventuallyAlways:
				funcName = propSpec.EventuallyAlways
			case Eventually:
				funcName = propSpec.Eventually
			case AlwaysEventually:
				funcName = propSpec.AlwaysEventually
			default:
				return fmt.Errorf("unknown temporal operator: %s", constraint.Operator)
			}

			callExpr := funcName + "()"

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

func (e *Executor) spawnThread(name string, entrypoint string, replicas int) error {
	// Default to 1 replica if not specified
	if replicas <= 0 {
		replicas = 1
	}

	// Create a ThreadSet with the specified number of replicas
	threadSet := interp.ThreadSet{
		Stacks:      make([]interp.StackFrames, replicas),
		PauseReason: make([]interp.Pause, replicas),
	}

	// Initialize each replica with the same entrypoint
	for i := 0; i < replicas; i++ {
		f, err := interp.FunctionCallFromString(e.Program, e.InitialState.Globals, entrypoint)
		if err != nil {
			return err
		}
		threadSet.Stacks[i] = interp.StackFrames{f}
		threadSet.PauseReason[i] = interp.Start
	}

	// Add the thread set to the state
	e.InitialState.ThreadSets = append(e.InitialState.ThreadSets, threadSet)

	// Add thread names for each replica
	for i := 0; i < replicas; i++ {
		if replicas == 1 {
			e.Threads = append(e.Threads, name)
		} else {
			e.Threads = append(e.Threads, fmt.Sprintf("%s[%d]", name, i))
		}
	}

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
