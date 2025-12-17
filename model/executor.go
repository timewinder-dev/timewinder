package model

import (
	"fmt"
	"io"
	"path/filepath"

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
	RunModel() error
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

func (e *Executor) RunModel() error {
	return e.Engine.RunModel()
}

// FormatPropertyViolation formats a single property violation for display
func FormatPropertyViolation(v PropertyViolation) string {
	var result string
	result += fmt.Sprintf("\n================================================================================\n")
	result += fmt.Sprintf("PROPERTY VIOLATION\n")
	result += fmt.Sprintf("================================================================================\n")
	result += fmt.Sprintf("Property: %s\n", v.PropertyName)

	// Prominently display which thread caused the violation
	if v.ThreadID >= 0 {
		result += fmt.Sprintf("Thread:   Thread %d (%s)\n", v.ThreadID, v.ThreadName)
	} else {
		result += fmt.Sprintf("Thread:   %s\n", v.ThreadName)
	}

	result += fmt.Sprintf("Message:  %s\n", v.Message)
	result += fmt.Sprintf("State:    #%d\n", v.StateNumber)
	result += fmt.Sprintf("Depth:    %d\n", v.Depth)
	result += fmt.Sprintf("Hash:     0x%x\n", v.StateHash)
	result += fmt.Sprintf("\n--------------------------------------------------------------------------------\n")
	result += fmt.Sprintf("Execution Trace:\n")
	result += fmt.Sprintf("--------------------------------------------------------------------------------\n")

	if len(v.Trace) == 0 {
		result += fmt.Sprintf("  (Initial state - no execution yet)\n")
	} else {
		// Show detailed trace if --details flag is set
		if v.ShowDetails && v.Program != nil && v.CAS != nil {
			result += reconstructTrace(v)
		} else {
			// Simple trace - just show thread and state hash
			for i, step := range v.Trace {
				result += fmt.Sprintf("  %2d. Thread %d → State 0x%x\n",
					i+1, step.ThreadRan, step.StateHash)
			}

			// Show final state information in simple mode
			if v.State != nil {
				result += fmt.Sprintf("\n--------------------------------------------------------------------------------\n")
				result += fmt.Sprintf("Final State:\n")
				result += fmt.Sprintf("--------------------------------------------------------------------------------\n")
				stateStr := v.State.PrettyPrint(v.Program)
				result += stateStr
			}
		}
	}

	result += fmt.Sprintf("================================================================================\n")
	return result
}

// reconstructTrace reconstructs detailed trace from CAS, showing state at each step
func reconstructTrace(v PropertyViolation) string {
	var result string

	for i, step := range v.Trace {
		// Retrieve state from CAS
		state, err := cas.Retrieve[*interp.State](v.CAS, step.StateHash)
		if err != nil {
			// If we can't retrieve, fall back to simple display
			result += fmt.Sprintf("\n  Step %d: Thread %d → State 0x%x (unavailable)\n",
				i+1, step.ThreadRan, step.StateHash)
			continue
		}

		// Header for this step
		result += fmt.Sprintf("\n  Step %d:\n", i+1)
		result += fmt.Sprintf("  ├─ Thread: %d\n", step.ThreadRan)

		// Get thread info
		if step.ThreadRan >= 0 && step.ThreadRan < len(state.Stacks) {
			stack := state.Stacks[step.ThreadRan]
			if len(stack) > 0 {
				currentFrame := stack[len(stack)-1]
				pc := currentFrame.PC
				pauseReason := state.PauseReason[step.ThreadRan]

				// Get location info
				lineNum := v.Program.GetLineNumber(pc)
				filename := v.Program.GetFilename(pc)

				// Get step name if yielded
				stepName := ""
				if pauseReason == interp.Yield && len(currentFrame.Stack) > 0 {
					if topValue, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.StrValue); ok {
						stepName = string(topValue)
					}
				}

				// Show location and step
				if stepName != "" {
					result += fmt.Sprintf("  ├─ Action: %s\n", stepName)
				}
				if filename != "" && lineNum > 0 {
					basename := filepath.Base(filename)
					result += fmt.Sprintf("  ├─ Location: %s:%d\n", basename, lineNum)
				}
				result += fmt.Sprintf("  ├─ Status: %s\n", pauseReason)
			}
		}

		// Show state information
		result += fmt.Sprintf("  └─ State:\n")
		stateStr := state.PrettyPrint(v.Program)
		// Indent the state output
		lines := splitIntoLines(stateStr)
		for _, line := range lines {
			result += fmt.Sprintf("     %s\n", line)
		}
	}

	// Show the final state that caused the violation
	if v.State != nil {
		result += fmt.Sprintf("\n  Final State (violation):\n")
		result += fmt.Sprintf("  ├─ Thread: %d\n", v.ThreadID)

		// Get info about the violating thread
		if v.ThreadID >= 0 && v.ThreadID < len(v.State.Stacks) {
			stack := v.State.Stacks[v.ThreadID]
			if len(stack) > 0 {
				currentFrame := stack[len(stack)-1]
				pc := currentFrame.PC
				pauseReason := v.State.PauseReason[v.ThreadID]

				// Get location info
				lineNum := v.Program.GetLineNumber(pc)
				filename := v.Program.GetFilename(pc)

				// Get step name if yielded
				stepName := ""
				if pauseReason == interp.Yield && len(currentFrame.Stack) > 0 {
					if topValue, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.StrValue); ok {
						stepName = string(topValue)
					}
				}

				// Show location and step
				if stepName != "" {
					result += fmt.Sprintf("  ├─ Action: %s\n", stepName)
				}
				if filename != "" && lineNum > 0 {
					basename := filepath.Base(filename)
					result += fmt.Sprintf("  ├─ Location: %s:%d\n", basename, lineNum)
				}
				result += fmt.Sprintf("  ├─ Status: %s\n", pauseReason)
			}
		}

		result += fmt.Sprintf("  └─ State:\n")
		stateStr := v.State.PrettyPrint(v.Program)
		lines := splitIntoLines(stateStr)
		for _, line := range lines {
			result += fmt.Sprintf("     %s\n", line)
		}
	}

	return result
}

// splitIntoLines splits a string into lines
func splitIntoLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := []string{}
	currentLine := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
		} else {
			currentLine += string(ch)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

// FormatAllViolations formats all property violations for display
func FormatAllViolations(violations []PropertyViolation) string {
	if len(violations) == 0 {
		return ""
	}

	var result string
	result += fmt.Sprintf("\n\n")
	result += fmt.Sprintf("================================================================================\n")
	result += fmt.Sprintf("PROPERTY VIOLATIONS FOUND: %d\n", len(violations))
	result += fmt.Sprintf("================================================================================\n")

	for i, v := range violations {
		result += fmt.Sprintf("\nViolation #%d:\n", i+1)
		result += FormatPropertyViolation(v)
	}

	return result
}
