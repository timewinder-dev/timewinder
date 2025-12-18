package model

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gookit/color"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// FormatPropertyViolation formats a single property violation for display
func FormatPropertyViolation(v PropertyViolation) string {
	var result string
	result += "\n" + color.Gray.Sprint("================================================================================") + "\n"
	result += color.Red.Sprint("PROPERTY VIOLATION") + "\n"
	result += color.Gray.Sprint("================================================================================") + "\n"
	result += color.Bold.Sprintf("Property: ") + color.Yellow.Sprintf("%s\n", v.PropertyName)

	// Show property type if specified and not "Always" (default)
	if v.PropertyType != "" && v.PropertyType != "Always" {
		result += color.Bold.Sprintf("Type:     ") + fmt.Sprintf("%s\n", v.PropertyType)
	}

	// Prominently display which thread caused the violation
	if v.ThreadID >= 0 {
		result += color.Bold.Sprintf("Thread:   ") + fmt.Sprintf("Thread %d (%s)\n", v.ThreadID, v.ThreadName)
	} else {
		result += color.Bold.Sprintf("Thread:   ") + fmt.Sprintf("%s\n", v.ThreadName)
	}

	result += color.Bold.Sprintf("Message:  ") + color.Red.Sprintf("%s\n", v.Message)
	result += color.Bold.Sprintf("State:    ") + fmt.Sprintf("#%d\n", v.StateNumber)
	result += color.Bold.Sprintf("Depth:    ") + fmt.Sprintf("%d\n", v.Depth)
	result += color.Bold.Sprintf("Hash:     ") + fmt.Sprintf("0x%x\n", v.StateHash)
	result += "\n" + color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"
	result += color.Cyan.Sprint("Execution Trace:") + "\n"
	result += color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"

	if len(v.Trace) == 0 {
		result += fmt.Sprintf("  (Initial state - no execution yet)\n")

		// Show initial state variables to help understand the violation
		if v.State != nil {
			result += "\n" + color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"
			result += color.Cyan.Sprint("Initial State Variables:") + "\n"
			result += color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"
			stateStr := v.State.PrettyPrint(v.Program)
			result += stateStr
		}
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
				result += "\n" + color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"
				result += color.Cyan.Sprint("Final State:") + "\n"
				result += color.Gray.Sprint("--------------------------------------------------------------------------------") + "\n"
				stateStr := v.State.PrettyPrint(v.Program)
				result += stateStr
			}
		}
	}

	result += color.Gray.Sprint("================================================================================") + "\n"
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
		lines := strings.Split(stateStr, "\n")
		for _, line := range lines {
			if line != "" {
				result += fmt.Sprintf("     %s\n", line)
			}
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
		lines := strings.Split(stateStr, "\n")
		for _, line := range lines {
			if line != "" {
				result += fmt.Sprintf("     %s\n", line)
			}
		}
	}

	return result
}

// FormatAllViolations formats all property violations for display
func FormatAllViolations(violations []PropertyViolation) string {
	if len(violations) == 0 {
		return ""
	}

	var result string
	result += "\n\n"
	result += color.Gray.Sprint("================================================================================") + "\n"
	result += color.Red.Sprintf("PROPERTY VIOLATIONS FOUND: %d\n", len(violations))
	result += color.Gray.Sprint("================================================================================") + "\n"

	for i, v := range violations {
		result += color.Yellow.Sprintf("\nViolation #%d:\n", i+1)
		result += FormatPropertyViolation(v)
	}

	return result
}

// FormatStatistics formats model checking statistics
func FormatStatistics(stats ModelStatistics) string {
	var result string
	result += "\n" + color.Cyan.Sprint("=== Model checking statistics ===") + "\n"
	result += color.Bold.Sprint("Total state transitions attempted: ") + fmt.Sprintf("%d\n", stats.TotalTransitions)
	result += color.Bold.Sprint("Unique states found: ") + fmt.Sprintf("%d\n", stats.UniqueStates)
	result += color.Bold.Sprint("Duplicate states pruned: ") + fmt.Sprintf("%d\n", stats.DuplicateStates)
	result += color.Bold.Sprint("Maximum depth: ") + fmt.Sprintf("%d\n", stats.MaxDepth)

	// Color the violation count based on whether there are violations
	if stats.ViolationCount > 0 {
		result += color.Bold.Sprint("Property violations found: ") + color.Red.Sprintf("%d\n", stats.ViolationCount)
	} else {
		result += color.Bold.Sprint("Property violations found: ") + color.Green.Sprintf("%d\n", stats.ViolationCount)
	}
	return result
}
