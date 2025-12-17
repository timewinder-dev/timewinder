package model

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/timewinder-dev/timewinder/interp"
)

type SingleThreadEngine struct {
	Queue     []*Thunk
	NextQueue []*Thunk
	Executor  *Executor
}

func InitSingleThread(exec *Executor) (*SingleThreadEngine, error) {
	allStates, err := interp.Canonicalize(exec.InitialState)
	if err != nil {
		return nil, err
	}
	st := &SingleThreadEngine{
		Executor: exec,
	}
	for _, s := range allStates {
		err := CheckProperties(s, exec.Properties)
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
			}

			if exec.KeepGoing {
				// Track violation but continue
				exec.Violations = append(exec.Violations, violation)
			} else {
				// Print formatted violation and stop
				fmt.Fprintf(os.Stderr, "%s", FormatPropertyViolation(violation))
				return nil, err
			}
		}
		for i := 0; i < len(exec.Threads); i++ {
			st.Queue = append(st.Queue, &Thunk{
				ToRun: i,
				State: s.Clone(),
			})
		}
	}
	return st, nil
}

func (s *SingleThreadEngine) RunModel() error {
	stateCount := 0
	depth := 0
	w := s.Executor.DebugWriter

	// Always print statistics on exit, even if there's an error
	defer func() {
		fmt.Fprintf(os.Stderr, "\n=== Model checking statistics ===\n")
		fmt.Fprintf(os.Stderr, "Total state transitions attempted: %d\n", stateCount)
		fmt.Fprintf(os.Stderr, "Unique states found: %d\n", len(s.Executor.VisitedStates))
		fmt.Fprintf(os.Stderr, "Duplicate states pruned: %d\n", stateCount-len(s.Executor.VisitedStates))
		fmt.Fprintf(os.Stderr, "Maximum depth: %d\n", depth)
		fmt.Fprintf(os.Stderr, "Property violations found: %d\n", len(s.Executor.Violations))

		// Print all violations if KeepGoing was enabled
		if s.Executor.KeepGoing && len(s.Executor.Violations) > 0 {
			fmt.Fprintf(os.Stderr, "%s", FormatAllViolations(s.Executor.Violations))
		}
	}()

	for {
		fmt.Fprintf(w, "\n=== Depth %d: Exploring %d states ===\n", depth, len(s.Queue))

		for len(s.Queue) != 0 {
			t := s.Queue[0]
			s.Queue = s.Queue[1:]
			stateCount++

			fmt.Fprintf(w, "\n--- State #%d: Running thread %d (%s) ---\n",
				stateCount, t.ToRun, s.Executor.Threads[t.ToRun])
			fmt.Fprintf(w, "Trace so far: %d steps\n", len(t.Trace))

			st, choices, err := RunTrace(t, s.Executor.Program)
			if err != nil {
				return err
			}

			// Handle non-deterministic choice (oneof) - immediate expansion
			if choices != nil {
				fmt.Fprintf(w, "Non-deterministic choice: expanding into %d branches\n", len(choices))
				for i, choice := range choices {
					successor := t.Clone()
					successor.State = st.Clone()
					// Push the concrete choice onto the stack
					currentFrame := successor.State.Stacks[t.ToRun].CurrentStack()
					currentFrame.Push(choice)
					// Add to current queue for immediate processing
					s.Queue = append(s.Queue, successor)
					fmt.Fprintf(w, "  Branch %d: %v\n", i+1, choice)
				}
				continue // Skip normal BuildRunnable handling
			}

			b, _ := json.Marshal(st)
			fmt.Fprintf(w, "After execution:\n%s\n", string(b))
			fmt.Fprintf(w, "Pause reasons: %v\n", st.PauseReason)

			err = CheckProperties(st, s.Executor.Properties)
			if err != nil {
				// Get state hash for violation tracking
				stateHash, hashErr := s.Executor.CAS.Put(st)
				if hashErr != nil {
					stateHash = 0 // Use 0 if hashing fails
				}

				violation := PropertyViolation{
					PropertyName: "Property", // Will be updated by CheckProperties
					Message:      err.Error(),
					StateHash:    stateHash,
					Depth:        depth,
					StateNumber:  stateCount,
					Trace:        t.Trace,
					State:        st,
					Program:      s.Executor.Program,
					ThreadID:     t.ToRun,
					ThreadName:   s.Executor.Threads[t.ToRun],
				}

				if s.Executor.KeepGoing {
					// Track violation but continue exploring
					s.Executor.Violations = append(s.Executor.Violations, violation)
					fmt.Fprintf(w, "⚠ Property violation detected (continuing due to --keep-going)\n")
					fmt.Fprintf(w, "  %s\n", err.Error())
				} else {
					// Print formatted violation and stop
					fmt.Fprintf(os.Stderr, "%s", FormatPropertyViolation(violation))
					return err
				}
			} else {
				fmt.Fprintf(w, "✓ All properties satisfied\n")
			}

			next, err := BuildRunnable(t, st, s.Executor)
			if err != nil {
				return err
			}

			if next == nil {
				fmt.Fprintf(w, "State already visited (cycle detected)\n")
			} else {
				fmt.Fprintf(w, "Generated %d successor states\n", len(next))
			}

			s.NextQueue = append(s.NextQueue, next...)
		}
		if len(s.NextQueue) == 0 {
			break
		}
		s.Queue = s.NextQueue
		s.NextQueue = nil
		depth++
	}

	// If KeepGoing was enabled and violations were found, return an error
	if s.Executor.KeepGoing && len(s.Executor.Violations) > 0 {
		return fmt.Errorf("Model checking completed with %d property violation(s)", len(s.Executor.Violations))
	}

	return nil
}
