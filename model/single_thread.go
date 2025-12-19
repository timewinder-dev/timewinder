package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

type SingleThreadEngine struct {
	Queue           []*Thunk
	NextQueue       []*Thunk
	Executor        *Executor
	stateCount      int // Total number of states explored
	depth           int // Current BFS depth
	prunedThisDepth int // Number of states pruned at current depth
}

func InitSingleThread(exec *Executor) (*SingleThreadEngine, error) {
	allStates, err := interp.Canonicalize(exec.InitialState)
	if err != nil {
		return nil, err
	}
	st := &SingleThreadEngine{
		Executor: exec,
	}
	// Only check "Always" properties at initial state (not EventuallyAlways, etc.)
	alwaysProperties := FilterPropertiesByOperator(exec.TemporalConstraints, Always)

	for _, s := range allStates {
		err := CheckProperties(s, alwaysProperties)
		if err != nil {
			violation := PropertyViolation{
				PropertyName: "InitialState",
				Message:      err.Error(),
				StateHash:    0, // No hash yet for initial state
				Depth:        0,
				StateNumber:  0,
				Trace:        nil,
				State:        s,
				Program:      exec.Program,
				ThreadID:     -1, // No thread ran yet
				ThreadName:   "(initial state)",
				ShowDetails:  exec.ShowDetails,
				CAS:          exec.CAS,
			}

			if exec.KeepGoing {
				// Track violation but continue
				exec.Violations = append(exec.Violations, violation)
			} else {
				// Stop on initial state violation
				return nil, err
			}
		}
		for i := 0; i < len(exec.Threads); i++ {
			// Convert flat index to ThreadID
			threadID := interp.ThreadID{}
			tmpIdx := i
			for setIdx := range s.ThreadSets {
				if tmpIdx < len(s.ThreadSets[setIdx].Stacks) {
					threadID = interp.ThreadID{SetIdx: setIdx, LocalIdx: tmpIdx}
					break
				}
				tmpIdx -= len(s.ThreadSets[setIdx].Stacks)
			}

			st.Queue = append(st.Queue, &Thunk{
				ToRun: threadID,
				State: s.Clone(),
			})
		}
	}
	return st, nil
}

// computeStatistics builds ModelStatistics from current engine state
func (s *SingleThreadEngine) computeStatistics() ModelStatistics {
	return ModelStatistics{
		TotalTransitions: s.stateCount,
		UniqueStates:     len(s.Executor.VisitedStates),
		DuplicateStates:  s.stateCount - len(s.Executor.VisitedStates),
		MaxDepth:         s.depth,
		ViolationCount:   len(s.Executor.Violations),
	}
}

// formatDepthReport builds a colorized depth report string
func formatDepthReport(depth, explored, pruned, remaining int, elapsed time.Duration) string {
	var b strings.Builder

	b.WriteString(color.Gray.Sprint("→"))
	b.WriteString(" ")
	b.WriteString(color.Cyan.Sprint("Depth"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprint(depth))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("•"))
	b.WriteString(" ")
	b.WriteString(color.Yellow.Sprint("Explored"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprint(explored))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("states"))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("•"))
	b.WriteString(" ")
	b.WriteString(color.Magenta.Sprint("Pruned"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprint(pruned))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("states"))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("•"))
	b.WriteString(" ")
	b.WriteString(color.Green.Sprint("Remaining"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprint(remaining))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("states"))
	b.WriteString(" ")
	b.WriteString(color.Gray.Sprint("•"))
	b.WriteString(" ")
	b.WriteString(color.Blue.Sprint("Time"))
	b.WriteString(" ")
	// Format time appropriately based on duration
	if elapsed < time.Second {
		b.WriteString(fmt.Sprintf("%dms", elapsed.Milliseconds()))
	} else {
		b.WriteString(fmt.Sprintf("%.2fs", elapsed.Seconds()))
	}
	b.WriteString("\n")

	return b.String()
}

// handleCyclicState handles a state that's been visited before (cycle detected)
func (s *SingleThreadEngine) handleCyclicState(t *Thunk, st *interp.State, stateHash cas.Hash) error {
	fmt.Fprintf(s.Executor.DebugWriter, "State already visited (pruning this branch)\n")

	// Check if this is a true cycle within the current trace
	// (i.e., does this state appear earlier in THIS trace?)
	isTrueCycle := false
	for _, step := range t.Trace {
		if step.StateHash == stateHash {
			isTrueCycle = true
			fmt.Fprintf(s.Executor.DebugWriter, "  → True cycle detected within this trace\n")
			break
		}
	}

	// Only check temporal constraints if this is a true cycle within the trace
	// If it's just revisiting a state from a different branch, don't check
	// (the trace hasn't truly terminated - it's just being pruned)
	if isTrueCycle && len(s.Executor.TemporalConstraints) > 0 {
		return CheckTemporalConstraints(t, st, s.Executor, true) // isCycle=true
	}

	return nil
}

// handleTerminatingState handles a state with no runnable successors
func (s *SingleThreadEngine) handleTerminatingState(t *Thunk, st *interp.State) error {
	fmt.Fprintf(s.Executor.DebugWriter, "Terminating state (no runnable threads)\n")

	// Check for deadlock: no runnable threads but not all threads finished
	if !s.Executor.NoDeadlocks {
		allFinished := true
		flatIdx := 0
		for _, threadSet := range st.ThreadSets {
			for _, reason := range threadSet.PauseReason {
				if reason != interp.Finished {
					allFinished = false
					fmt.Fprintf(s.Executor.DebugWriter, "  Thread %d (%s) is not finished: %v\n",
						flatIdx, s.Executor.Threads[flatIdx], reason)
				}
				flatIdx++
			}
		}

		if !allFinished {
			// Deadlock detected: threads are stuck waiting
			return fmt.Errorf("Deadlock detected: no threads can make progress, but not all threads have finished")
		}
	}

	// Check temporal constraints for terminating trace
	if len(s.Executor.TemporalConstraints) > 0 {
		return CheckTemporalConstraints(t, st, s.Executor, false) // isCycle=false
	}
	return nil
}

// handleStutterCheck checks temporal properties as if the process terminates at this state
// This is skipped for WeaklyFairYield states (weak fairness assumption)
func (s *SingleThreadEngine) handleStutterCheck(t *Thunk, st *interp.State) (*ModelResult, error) {
	w := s.Executor.DebugWriter
	threadPauseReason := st.GetPauseReason(t.ToRun)

	// Check stutter for:
	// - Yield (normal step())
	// - Waiting (until() - user requested stutter checking)
	// Skip for:
	// - WeaklyFairYield (fstep())
	// - WeaklyFairWaiting (funtil())
	shouldCheck := (threadPauseReason == interp.Yield || threadPauseReason == interp.Waiting)

	if !shouldCheck || len(s.Executor.TemporalConstraints) == 0 {
		return nil, nil
	}

	// Compute flat thread index for logging
	flatIdx := 0
	for i := 0; i < t.ToRun.SetIdx; i++ {
		flatIdx += len(st.ThreadSets[i].Stacks)
	}
	flatIdx += t.ToRun.LocalIdx

	fmt.Fprintf(w, "Checking stutter for thread %d (%s) - as if process terminates here...\n",
		flatIdx, s.Executor.Threads[flatIdx])

	// Clone the state and mark the thread as Stuttering for clarity in output
	stutterState := st.Clone()
	stutterState.SetPauseReason(t.ToRun, interp.Stuttering)

	err := CheckTemporalConstraints(t, stutterState, s.Executor, false) // isCycle=false for stutter
	if err != nil {
		fmt.Fprintf(w, "⚠ Stutter check failed for thread %d: %s\n", flatIdx, err.Error())

		// Get state hash for violation tracking (use stutter state)
		stateHash, hashErr := s.Executor.CAS.Put(stutterState)
		if hashErr != nil {
			stateHash = 0
		}

		// Convert ThreadID to flat index
		threadFlatIdx := 0
		for i := 0; i < t.ToRun.SetIdx; i++ {
			threadFlatIdx += len(stutterState.ThreadSets[i].Stacks)
		}
		threadFlatIdx += t.ToRun.LocalIdx

		violation := PropertyViolation{
			PropertyName: "Stutter Check",
			Message: fmt.Sprintf("Stutter check failed at thread %d (%s): %s",
				threadFlatIdx, s.Executor.Threads[threadFlatIdx], err.Error()),
			StateHash:   stateHash,
			Depth:       s.depth,
			StateNumber: s.stateCount,
			Trace:       t.Trace,
			State:       stutterState, // Use stutter state to show [Stuttering] in output
			Program:     s.Executor.Program,
			ThreadID:    threadFlatIdx,
			ThreadName:  s.Executor.Threads[threadFlatIdx],
			ShowDetails: s.Executor.ShowDetails,
			CAS:         s.Executor.CAS,
		}

		// Always add violation to the list for statistics
		s.Executor.Violations = append(s.Executor.Violations, violation)

		if s.Executor.KeepGoing {
			return nil, nil
		} else {
			return &ModelResult{
				Statistics: s.computeStatistics(),
				Violations: s.Executor.Violations, // Use the full violations list
				Success:    false,
			}, nil
		}
	}

	fmt.Fprintf(w, "✓ Stutter check passed for thread %d\n", t.ToRun)
	return nil, nil
}

// handlePropertyViolation creates and records a property violation
func (s *SingleThreadEngine) handlePropertyViolation(t *Thunk, st *interp.State, err error) (*ModelResult, error) {
	// Get state hash for violation tracking
	stateHash, hashErr := s.Executor.CAS.Put(st)
	if hashErr != nil {
		stateHash = 0 // Use 0 if hashing fails
	}

	// Convert ThreadID to flat index
	threadFlatIdx := 0
	for i := 0; i < t.ToRun.SetIdx; i++ {
		threadFlatIdx += len(st.ThreadSets[i].Stacks)
	}
	threadFlatIdx += t.ToRun.LocalIdx

	violation := PropertyViolation{
		PropertyName: "Property", // Will be updated by CheckProperties
		Message:      err.Error(),
		StateHash:    stateHash,
		Depth:        s.depth,
		StateNumber:  s.stateCount,
		Trace:        t.Trace,
		State:        st,
		Program:      s.Executor.Program,
		ThreadID:     threadFlatIdx,
		ThreadName:   s.Executor.Threads[threadFlatIdx],
		ShowDetails:  s.Executor.ShowDetails,
		CAS:          s.Executor.CAS,
	}

	if s.Executor.KeepGoing {
		// Track violation but continue exploring
		s.Executor.Violations = append(s.Executor.Violations, violation)
		fmt.Fprintf(s.Executor.DebugWriter, "⚠ Property violation detected (continuing due to --keep-going)\n")
		fmt.Fprintf(s.Executor.DebugWriter, "  %s\n", err.Error())
		return nil, nil
	} else {
		// Stop on first violation
		stats := s.computeStatistics()
		stats.ViolationCount = 1 // Fix count to reflect the actual violation being returned
		return &ModelResult{
			Statistics: stats,
			Violations: []PropertyViolation{violation},
			Success:    false,
		}, nil
	}
}

func (s *SingleThreadEngine) RunModel() (*ModelResult, error) {
	s.stateCount = 0
	s.depth = 0
	w := s.Executor.DebugWriter

	for {
		fmt.Fprintf(w, "\n=== Depth %d: Exploring %d states ===\n", s.depth, len(s.Queue))

		// Track timing for this depth
		depthStart := time.Now()

		// Reset pruned counter for this depth
		s.prunedThisDepth = 0
		queueSize := len(s.Queue)

		for len(s.Queue) != 0 {
			t := s.Queue[0]
			s.Queue = s.Queue[1:]
			s.stateCount++

			// Convert ThreadID to flat index for display
			threadFlatIdx := 0
			for i := 0; i < t.ToRun.SetIdx; i++ {
				threadFlatIdx += len(t.State.ThreadSets[i].Stacks)
			}
			threadFlatIdx += t.ToRun.LocalIdx

			fmt.Fprintf(w, "\n--- State #%d: Running thread %d (Set %d, Local %d) %s ---\n",
				s.stateCount, threadFlatIdx, t.ToRun.SetIdx, t.ToRun.LocalIdx, s.Executor.Threads[threadFlatIdx])
			fmt.Fprintf(w, "Trace so far: %d steps\n", len(t.Trace))

			// Execute thread
			st, choices, err := RunTrace(t, s.Executor.Program)
			if err != nil {
				return nil, err
			}

			// Handle non-deterministic choice (oneof) - immediate expansion
			if choices != nil {
				fmt.Fprintf(w, "Non-deterministic choice: expanding into %d branches\n", len(choices))
				for i, choice := range choices {
					successor := t.Clone()
					successor.State = st.Clone()
					// Push the concrete choice onto the stack
					currentFrame := successor.State.GetStackFrames(t.ToRun).CurrentStack()
					currentFrame.Push(choice)
					// Add to current queue for immediate processing
					s.Queue = append(s.Queue, successor)
					fmt.Fprintf(w, "  Branch %d: %v\n", i+1, choice)
				}
				continue // Skip normal BuildRunnable handling
			}

			b, _ := json.Marshal(st)
			fmt.Fprintf(w, "After execution:\n%s\n", string(b))
			// Flatten pause reasons for display
			var pauseReasons []interp.Pause
			for _, threadSet := range st.ThreadSets {
				pauseReasons = append(pauseReasons, threadSet.PauseReason...)
			}
			fmt.Fprintf(w, "Pause reasons: %v\n", pauseReasons)

			// Check for cycles BEFORE generating successors
			stateHash, err := s.Executor.CAS.Put(st)
			if err != nil {
				return nil, fmt.Errorf("hashing state: %w", err)
			}

			if s.Executor.VisitedStates[stateHash] {
				// State already visited - prune this branch
				s.prunedThisDepth++
				err := s.handleCyclicState(t, st, stateHash)
				if err != nil {
					result, err := s.handlePropertyViolation(t, st, err)
					if result != nil {
						return result, err
					}
				}
				continue // Don't generate successors for already-visited states
			}

			// Mark state as visited
			s.Executor.VisitedStates[stateHash] = true

			// Check for livelock (same weak state recurring)
			isLivelock, err := DetectLivelock(s.Executor, st, s.depth)
			if err != nil {
				fmt.Fprintf(w, "Warning: livelock detection failed: %v\n", err)
			}
			if isLivelock && s.Executor.Reporter != nil {
				s.Executor.Reporter.Printf("%s Livelock detected - cycling through equivalent states\n",
					color.Yellow.Sprint("⚠"))
			}

			// Check invariant properties (Always) - not EventuallyAlways
			alwaysProps := FilterPropertiesByOperator(s.Executor.TemporalConstraints, Always)
			err = CheckProperties(st, alwaysProps)
			if err != nil {
				result, err := s.handlePropertyViolation(t, st, err)
				if result != nil {
					return result, err
				}
			} else {
				fmt.Fprintf(w, "✓ All properties satisfied\n")
			}

			// Check stutter: temporal properties as if process terminates here
			// Skipped for WeaklyFairYield (weak fairness assumption)
			result, err := s.handleStutterCheck(t, st)
			if result != nil || err != nil {
				return result, err
			}

			// Generate successors (BuildRunnable no longer checks cycles)
			next, err := BuildRunnable(t, st, s.Executor)
			if err != nil {
				return nil, err
			}

			// Check if this is a terminating state (no runnable successors)
			if len(next) == 0 {
				fmt.Fprintf(w, "No successors generated (terminating state)\n")
				err := s.handleTerminatingState(t, st)
				if err != nil {
					fmt.Fprintf(w, "  → Terminating state error: %v\n", err)
					result, err := s.handlePropertyViolation(t, st, err)
					if result != nil {
						return result, err
					}
					// With --keep-going, we recorded the violation but should NOT generate successors
					fmt.Fprintf(w, "  → Continuing after violation (no successors will be added)\n")
				}
			} else {
				fmt.Fprintf(w, "Generated %d successor states\n", len(next))
			}

			fmt.Fprintf(w, "Adding %d states to NextQueue (queue will have %d total)\n", len(next), len(s.NextQueue)+len(next))
			s.NextQueue = append(s.NextQueue, next...)
		}

		// Report progress after processing this depth
		if s.Executor.Reporter != nil {
			elapsed := time.Since(depthStart)
			report := formatDepthReport(s.depth, queueSize, s.prunedThisDepth, len(s.NextQueue), elapsed)
			s.Executor.Reporter.Printf("%s", report)
		}

		if len(s.NextQueue) == 0 {
			break
		}
		s.Queue = s.NextQueue
		s.NextQueue = nil
		s.depth++

		// Check if we've reached max depth
		if s.Executor.MaxDepth > 0 && s.depth >= s.Executor.MaxDepth {
			fmt.Fprintf(s.Executor.DebugWriter, "\n⚠ Reached maximum depth %d, stopping exploration\n", s.Executor.MaxDepth)
			if s.Executor.Reporter != nil {
				s.Executor.Reporter.Printf("%s Reached maximum depth %d, stopping exploration\n",
					color.Yellow.Sprint("⚠"),
					s.Executor.MaxDepth)
			}
			break
		}
	}

	// Build result using helper function
	result := &ModelResult{
		Statistics: s.computeStatistics(),
		Violations: s.Executor.Violations,
		Success:    len(s.Executor.Violations) == 0,
	}

	// Violations are reported in the result, not as errors
	return result, nil
}
