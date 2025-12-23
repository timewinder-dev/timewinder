package model

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gookit/color"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// FormatPropertyViolation formats a single property violation for display
func FormatPropertyViolation(v PropertyViolation) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(color.Gray.Sprint("================================================================================"))
	b.WriteString("\n")
	b.WriteString(color.Red.Sprint("PROPERTY VIOLATION"))
	b.WriteString("\n")
	b.WriteString(color.Gray.Sprint("================================================================================"))
	b.WriteString("\n")
	b.WriteString(color.Bold.Sprint("Property: "))
	b.WriteString(color.Yellow.Sprintf("%s\n", v.PropertyName))

	// Show property type if specified and not "Always" (default)
	if v.PropertyType != "" && v.PropertyType != "Always" {
		b.WriteString(color.Bold.Sprint("Type:     "))
		b.WriteString(fmt.Sprintf("%s\n", v.PropertyType))
	}

	// Prominently display which thread caused the violation
	if v.ThreadID >= 0 {
		b.WriteString(color.Bold.Sprint("Thread:   "))
		b.WriteString(fmt.Sprintf("Thread %d (%s)\n", v.ThreadID, v.ThreadName))
	} else {
		b.WriteString(color.Bold.Sprint("Thread:   "))
		b.WriteString(fmt.Sprintf("%s\n", v.ThreadName))
	}

	b.WriteString(color.Bold.Sprint("Message:  "))
	b.WriteString(color.Red.Sprintf("%s\n", v.Message))
	b.WriteString(color.Bold.Sprint("State:    "))
	b.WriteString(fmt.Sprintf("#%d\n", v.StateNumber))
	b.WriteString(color.Bold.Sprint("Depth:    "))
	b.WriteString(fmt.Sprintf("%d\n", v.Depth))
	b.WriteString(color.Bold.Sprint("Hash:     "))
	b.WriteString(fmt.Sprintf("0x%x\n", v.StateHash))
	b.WriteString("\n")
	b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
	b.WriteString("\n")
	b.WriteString(color.Cyan.Sprint("Execution Trace:"))
	b.WriteString("\n")
	b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
	b.WriteString("\n")

	if len(v.Trace) == 0 {
		b.WriteString("  (Initial state - no execution yet)\n")

		// Show initial state variables to help understand the violation
		if v.State != nil {
			b.WriteString("\n")
			b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
			b.WriteString("\n")
			b.WriteString(color.Cyan.Sprint("Initial State Variables:"))
			b.WriteString("\n")
			b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
			b.WriteString("\n")
			stateStr := v.State.PrettyPrint(v.Program)
			b.WriteString(stateStr)
		}
	} else {
		// Show detailed trace if --details flag is set
		if v.ShowDetails && v.Program != nil && v.CAS != nil {
			reconstructTrace(&b, v)
		} else {
			// Simple trace - just show thread and state hash
			for i, step := range v.Trace {
				b.WriteString(fmt.Sprintf("  %2d. Thread %d → State 0x%x\n",
					i+1, step.ThreadRan, step.StateHash))
			}

			// Show final state information in simple mode
			if v.State != nil {
				b.WriteString("\n")
				b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
				b.WriteString("\n")
				b.WriteString(color.Cyan.Sprint("Final State:"))
				b.WriteString("\n")
				b.WriteString(color.Gray.Sprint("--------------------------------------------------------------------------------"))
				b.WriteString("\n")
				stateStr := v.State.PrettyPrint(v.Program)
				b.WriteString(stateStr)
			}
		}
	}

	b.WriteString(color.Gray.Sprint("================================================================================"))
	b.WriteString("\n")
	return b.String()
}

// reconstructTrace reconstructs detailed trace from CAS, writing directly to w
func reconstructTrace(w io.Writer, v PropertyViolation) {
	for i, step := range v.Trace {
		// Retrieve state from CAS
		state, err := cas.Retrieve[*interp.State](v.CAS, step.StateHash)
		if err != nil {
			// If we can't retrieve, fall back to simple display
			fmt.Fprintf(w, "\n  Step %d: Thread %d → State 0x%x (unavailable)\n",
				i+1, step.ThreadRan, step.StateHash)
			continue
		}

		// Header for this step
		fmt.Fprintf(w, "\n  Step %d:\n", i+1)

		// Convert ThreadID to flat index for display
		flatIdx := 0
		for si := 0; si < step.ThreadRan.SetIdx; si++ {
			flatIdx += len(state.ThreadSets[si].Stacks)
		}
		flatIdx += step.ThreadRan.LocalIdx
		fmt.Fprintf(w, "  ├─ Thread: %d (Set %d, Local %d)\n", flatIdx, step.ThreadRan.SetIdx, step.ThreadRan.LocalIdx)

		// Get thread info
		if flatIdx < state.ThreadCount() {
			stack := state.GetStackFrames(step.ThreadRan)
			if len(stack) > 0 {
				currentFrame := stack[len(stack)-1]
				pc := currentFrame.PC
				pauseReason := state.GetPauseReason(step.ThreadRan)

				// Get location info
				lineNum := v.Program.GetLineNumber(pc)
				filename := v.Program.GetFilename(pc)

				// Get step name if yielded
				stepName := ""
				if pauseReason == interp.Runnable && len(currentFrame.Stack) > 0 {
					if topValue, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.StrValue); ok {
						stepName = string(topValue)
					}
				}

				// Show location and step
				if stepName != "" {
					fmt.Fprintf(w, "  ├─ Action: %s\n", stepName)
				}
				if filename != "" && lineNum > 0 {
					basename := filepath.Base(filename)
					fmt.Fprintf(w, "  ├─ Location: %s:%d\n", basename, lineNum)
				}
				fmt.Fprintf(w, "  ├─ Status: %s\n", pauseReason)
			}
		}

		// Show state information - write indented output
		fmt.Fprint(w, "  └─ State:\n")
		indentedWriter := &indentWriter{w: w, indent: "     ", atLineStart: true}
		state.PrettyPrintTo(indentedWriter, v.Program)
	}

	// Show the final state that caused the violation
	if v.State != nil {
		fmt.Fprint(w, "\n  Final State (violation):\n")
		fmt.Fprintf(w, "  ├─ Thread: %d\n", v.ThreadID)

		// Get info about the violating thread
		// Convert flat ThreadID to ThreadID struct
		if v.ThreadID >= 0 && v.ThreadID < v.State.ThreadCount() {
			threadID := interp.ThreadID{}
			tmpIdx := v.ThreadID
			for si := range v.State.ThreadSets {
				if tmpIdx < len(v.State.ThreadSets[si].Stacks) {
					threadID = interp.ThreadID{SetIdx: si, LocalIdx: tmpIdx}
					break
				}
				tmpIdx -= len(v.State.ThreadSets[si].Stacks)
			}

			stack := v.State.GetStackFrames(threadID)
			if len(stack) > 0 {
				currentFrame := stack[len(stack)-1]
				pc := currentFrame.PC
				pauseReason := v.State.GetPauseReason(threadID)

				// Get location info
				lineNum := v.Program.GetLineNumber(pc)
				filename := v.Program.GetFilename(pc)

				// Get step name if yielded
				stepName := ""
				if pauseReason == interp.Runnable && len(currentFrame.Stack) > 0 {
					if topValue, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.StrValue); ok {
						stepName = string(topValue)
					}
				}

				// Show location and step
				if stepName != "" {
					fmt.Fprintf(w, "  ├─ Action: %s\n", stepName)
				}
				if filename != "" && lineNum > 0 {
					basename := filepath.Base(filename)
					fmt.Fprintf(w, "  ├─ Location: %s:%d\n", basename, lineNum)
				}
				fmt.Fprintf(w, "  ├─ Status: %s\n", pauseReason)
			}
		}

		fmt.Fprint(w, "  └─ State:\n")
		indentedWriter := &indentWriter{w: w, indent: "     ", atLineStart: true}
		v.State.PrettyPrintTo(indentedWriter, v.Program)
	}
}

// indentWriter wraps an io.Writer to add indentation to each line
type indentWriter struct {
	w          io.Writer
	indent     string
	atLineStart bool
}

func (iw *indentWriter) Write(p []byte) (n int, err error) {
	totalWritten := 0

	for len(p) > 0 {
		// Write indent if we're at the start of a line
		if iw.atLineStart {
			if _, err := io.WriteString(iw.w, iw.indent); err != nil {
				return totalWritten, err
			}
			iw.atLineStart = false
		}

		// Find next newline
		idx := 0
		for idx < len(p) && p[idx] != '\n' {
			idx++
		}

		// Include the newline if found
		if idx < len(p) {
			idx++ // Include '\n'
			iw.atLineStart = true
		}

		// Write this chunk
		written, err := iw.w.Write(p[:idx])
		totalWritten += written
		if err != nil {
			return totalWritten, err
		}

		p = p[idx:]
	}

	return totalWritten, nil
}

// FormatAllViolations formats all property violations for display
func FormatAllViolations(violations []PropertyViolation) string {
	if len(violations) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(color.Gray.Sprint("================================================================================"))
	b.WriteString("\n")
	b.WriteString(color.Red.Sprintf("PROPERTY VIOLATIONS FOUND: %d\n", len(violations)))
	b.WriteString(color.Gray.Sprint("================================================================================"))
	b.WriteString("\n")

	for i, v := range violations {
		b.WriteString(color.Yellow.Sprintf("\nViolation #%d:\n", i+1))
		b.WriteString(FormatPropertyViolation(v))
	}

	return b.String()
}

// FormatStatistics formats model checking statistics
func FormatStatistics(stats ModelStatistics) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(color.Cyan.Sprint("=== Model checking statistics ==="))
	b.WriteString("\n")
	b.WriteString(color.Bold.Sprint("Total state transitions attempted: "))
	b.WriteString(fmt.Sprintf("%d\n", stats.TotalTransitions))
	b.WriteString(color.Bold.Sprint("Unique states found: "))
	b.WriteString(fmt.Sprintf("%d\n", stats.UniqueStates))
	b.WriteString(color.Bold.Sprint("Duplicate states pruned: "))
	b.WriteString(fmt.Sprintf("%d\n", stats.DuplicateStates))
	b.WriteString(color.Bold.Sprint("Maximum depth: "))
	b.WriteString(fmt.Sprintf("%d\n", stats.MaxDepth))

	// Color the violation count based on whether there are violations
	b.WriteString(color.Bold.Sprint("Property violations found: "))
	if stats.ViolationCount > 0 {
		b.WriteString(color.Red.Sprintf("%d\n", stats.ViolationCount))
	} else {
		b.WriteString(color.Green.Sprintf("%d\n", stats.ViolationCount))
	}

	// Show livelock count
	b.WriteString(color.Bold.Sprint("Livelocks detected: "))
	if stats.LivelockCount > 0 {
		b.WriteString(color.Yellow.Sprintf("%d\n", stats.LivelockCount))
	} else {
		b.WriteString(fmt.Sprintf("%d\n", stats.LivelockCount))
	}
	return b.String()
}
