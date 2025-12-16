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
			return nil, err
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

			st, err := RunTrace(t, s.Executor.Program)
			if err != nil {
				return err
			}

			b, _ := json.Marshal(st)
			fmt.Fprintf(w, "After execution:\n%s\n", string(b))
			fmt.Fprintf(w, "Pause reasons: %v\n", st.PauseReason)

			err = CheckProperties(st, s.Executor.Properties)
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "✓ All properties satisfied\n")

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

	return nil
}
