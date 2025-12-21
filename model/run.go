package model

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func RunTrace(t *Thunk, prog *vm.Program) (*interp.State, []vm.Value, error) {
	state := t.State.Clone()
	choices, err := interp.RunToPause(prog, state, t.ToRun)
	if err != nil {
		return nil, nil, err
	}
	return state, choices, nil
}

// BuildRunnable generates successor states for a given thunk
// Note: Cycle detection is now handled in RunModel(), not here
func BuildRunnable(t *Thunk, state *interp.State, exec *Executor) ([]*Thunk, error) {
	log.Trace().Interface("from_thread", t.ToRun).Msg("BuildRunnable: generating successors")

	// Hash the state (CAS handles decomposition internally)
	stateHash, err := exec.CAS.Put(state)
	if err != nil {
		return nil, fmt.Errorf("hashing state: %w", err)
	}

	// Canonicalize the state to handle non-determinism
	states, err := interp.Canonicalize(state)
	if err != nil {
		return nil, err
	}

	log.Trace().Int("canonical_states", len(states)).Msg("BuildRunnable: canonicalized states")
	if len(states) > 1 {
		log.Debug().Int("canonical_states", len(states)).Msg("BuildRunnable: multiple canonical states from Canonicalize")
	}

	// Build trace step for this execution
	trace := TraceStep{ThreadRan: t.ToRun, StateHash: stateHash}

	// Generate successor thunks for each runnable thread
	var out []*Thunk
	for setIdx, threadSet := range state.ThreadSets {
		for localIdx, r := range threadSet.PauseReason {
			threadID := interp.ThreadID{SetIdx: setIdx, LocalIdx: localIdx}

			log.Trace().Interface("thread", threadID).Str("pause_reason", r.String()).Msg("BuildRunnable: checking thread")

			switch r {
			case interp.Finished:
				// Thread has finished - no successors
				log.Trace().Interface("thread", threadID).Msg("BuildRunnable: thread finished, skipping")
				continue

			case interp.Waiting:
				fallthrough
			case interp.WeaklyFairWaiting:
				// Re-check if wait condition is now satisfied
				log.Trace().Interface("thread", threadID).Msg("BuildRunnable: thread waiting, re-checking condition")
				satisfied, err := evaluateWaitCondition(state, threadID, exec.Program, exec.CAS)
				if err != nil {
					return nil, fmt.Errorf("Error evaluating wait condition for thread (%d,%d): %w", setIdx, localIdx, err)
				}
				if !satisfied {
					// Still waiting - not runnable
					log.Trace().Interface("thread", threadID).Msg("BuildRunnable: condition still false, thread not runnable")
					continue
				}
					// Condition now satisfied - thread is runnable
				// CRITICAL: Rewind PC to re-check condition atomically when resuming
				log.Trace().Interface("thread", threadID).Msg("BuildRunnable: condition now true, rewinding PC")
				for _, s := range states {
					sClone := s.Clone()
					frame := sClone.GetStackFrames(threadID).CurrentStack()
					if frame.WaitCondition != nil {
						// Rewind to condition start so thread re-checks atomically
						frame.PC = frame.WaitCondition.ConditionPC
						frame.WaitCondition = nil
						log.Trace().Interface("thread", threadID).Str("pc", frame.PC.String()).Msg("BuildRunnable: PC rewound")
					}
					log.Debug().Interface("thread", threadID).Msg("BuildRunnable: adding successor for waiting thread")
					out = append(out, &Thunk{
						ToRun: threadID,
						State: sClone,
						Trace: append(t.Trace, trace),
					})
				}

			case interp.Yield:
				fallthrough
			case interp.WeaklyFairYield:
				fallthrough
			case interp.Start:
				// Thread is runnable - create successor thunks
				for _, s := range states {
					newThunk := &Thunk{
						ToRun: threadID,
						State: s,
						Trace: append(t.Trace, trace),
					}
					out = append(out, newThunk)
				}
			}
		}
	}
	return out, nil
}

// evaluateWaitCondition re-executes the wait condition to check if it's now true
func evaluateWaitCondition(state *interp.State, threadID interp.ThreadID, prog *vm.Program, casStore cas.CAS) (bool, error) {
	currentFrame := state.GetStackFrames(threadID).CurrentStack()
	if currentFrame.WaitCondition == nil {
		return false, fmt.Errorf("Thread (%d,%d) in Waiting state but WaitCondition is nil", threadID.SetIdx, threadID.LocalIdx)
	}

	log.Trace().
		Interface("thread", threadID).
		Str("condition_pc", currentFrame.WaitCondition.ConditionPC.String()).
		Str("current_pc", currentFrame.PC.String()).
		Msg("evaluateWaitCondition: re-checking wait condition")

	// Clone state for test evaluation (don't modify original)
	testState := state.Clone()
	testFrame := testState.GetStackFrames(threadID).CurrentStack()

	// Log global and local state before rewinding
	log.Trace().
		Interface("thread", threadID).
		Interface("globals", testState.Globals.Variables).
		Interface("locals", testFrame.Variables).
		Int("stack_frames", len(testState.GetStackFrames(threadID))).
		Msg("evaluateWaitCondition: global and local state before condition check")

	// Hash the state BEFORE running to compare later
	originalStateHash, err := casStore.Put(testState)
	if err != nil {
		return false, fmt.Errorf("failed to hash state: %w", err)
	}

	// Rewind PC to condition start
	testFrame.PC = currentFrame.WaitCondition.ConditionPC
	originalConditionPC := currentFrame.WaitCondition.ConditionPC
	log.Trace().Interface("thread", threadID).Str("rewound_pc", testFrame.PC.String()).Msg("evaluateWaitCondition: rewound PC")

	// Execute this thread until it pauses
	_, err = interp.RunToPause(prog, testState, threadID)
	if err != nil {
		log.Trace().Interface("thread", threadID).Err(err).Msg("evaluateWaitCondition: error during condition check")
		return false, fmt.Errorf("Error re-evaluating wait condition: %w", err)
	}

	// Check the result:
	newReason := testState.GetPauseReason(threadID)
	newFrame := testState.GetStackFrames(threadID).CurrentStack()

	// Hash the state AFTER running to see if it changed
	newStateHash, err := casStore.Put(testState)
	if err != nil {
		return false, fmt.Errorf("failed to hash new state: %w", err)
	}

	stateChanged := originalStateHash != newStateHash

	log.Trace().
		Interface("thread", threadID).
		Str("new_reason", newReason.String()).
		Bool("has_wait_condition", newFrame.WaitCondition != nil).
		Str("original_pc", originalConditionPC.String()).
		Str("new_pc", func() string {
			if newFrame.WaitCondition != nil {
				return newFrame.WaitCondition.ConditionPC.String()
			}
			return "nil"
		}()).
		Bool("state_changed", stateChanged).
		Msg("evaluateWaitCondition: checking result")

	// Logic for determining if condition is satisfied:
	// 1. If thread is NOT Waiting anymore (Yield, Finished, etc.), condition was TRUE
	// 2. If thread is Waiting at the SAME PC AND state didn't change, condition is FALSE
	// 3. If thread is Waiting at the SAME PC BUT state changed, condition was TRUE (thread made progress, then looped back)
	// 4. If thread is Waiting at a DIFFERENT PC, condition was TRUE (moved to different condition)

	if newReason == interp.Waiting || newReason == interp.WeaklyFairWaiting {
		if newFrame.WaitCondition != nil && newFrame.WaitCondition.ConditionPC == originalConditionPC {
			// Waiting at the same condition
			if !stateChanged {
				// State didn't change - condition is FALSE
				log.Trace().Interface("thread", threadID).Msg("evaluateWaitCondition: condition false (same PC, no state change)")
				return false, nil
			}
			// State changed - condition was TRUE, thread made progress and looped back
			log.Trace().Interface("thread", threadID).Msg("evaluateWaitCondition: condition true (same PC, but state changed)")
			return true, nil
		}
		// Waiting at a different condition - original condition was TRUE
		log.Trace().Interface("thread", threadID).Msg("evaluateWaitCondition: condition true (different PC)")
		return true, nil
	}

	// Not waiting - condition was TRUE
	log.Trace().Interface("thread", threadID).Msg("evaluateWaitCondition: condition true (not waiting)")
	return true, nil
}

// FilterPropertiesByOperator returns properties that match the given operator
func FilterPropertiesByOperator(constraints []TemporalConstraint, operator TemporalOperator) []Property {
	var result []Property
	for _, constraint := range constraints {
		if constraint.Operator == operator {
			result = append(result, constraint.Property)
		}
	}
	return result
}

func CheckProperties(state *interp.State, props []Property) error {
	for _, prop := range props {
		result, err := prop.Check(state)
		if err != nil {
			// Execution error - propagate it
			return fmt.Errorf("Error checking property: %w", err)
		}
		if !result.Success {
			// Property violation - return error with details
			return fmt.Errorf("Property violation: %s", result.Message)
		}
	}
	return nil
}

// CheckTemporalConstraints checks temporal constraints (Always, EventuallyAlways, etc.)
// at the end of a trace (either cycle or termination)
func CheckTemporalConstraints(t *Thunk, finalState *interp.State, exec *Executor, isCycle bool) error {
	if len(exec.TemporalConstraints) == 0 {
		return nil
	}

	// Build the complete trace of state hashes
	// Include: initial state + all intermediate states + final state
	stateHashes := make([]interface{}, len(t.Trace)+2)

	// Initial state (before any steps)
	initialHash, err := exec.CAS.Put(exec.InitialState)
	if err != nil {
		return fmt.Errorf("hashing initial state: %w", err)
	}
	stateHashes[0] = initialHash

	// All intermediate states from trace
	for i, step := range t.Trace {
		stateHashes[i+1] = step.StateHash
	}

	// Final state (the terminating or cyclic state)
	finalHash, err := exec.CAS.Put(finalState)
	if err != nil {
		return fmt.Errorf("hashing final state: %w", err)
	}
	stateHashes[len(t.Trace)+1] = finalHash

	// Check each temporal constraint
	for _, constraint := range exec.TemporalConstraints {
		var result PropertyResult
		var err error

		// Dispatch based on operator type
		switch constraint.Operator {
		case Always:
			// Already checked at each state during execution
			continue
		case EventuallyAlways:
			result, err = checkEventuallyAlways(constraint, stateHashes, exec.CAS, isCycle)
		case Eventually:
			// Future: implement Eventually
			continue
		case AlwaysEventually:
			// Future: implement AlwaysEventually
			continue
		default:
			return fmt.Errorf("unknown temporal operator: %s", constraint.Operator)
		}

		if err != nil {
			return err
		}
		if !result.Success {
			return fmt.Errorf("Property violation: %s", result.Message)
		}
	}

	return nil
}

// checkEventuallyAlways implements the EventuallyAlways (◇□P) operator
// Checks if there exists a point k where property becomes true and stays true forever
func checkEventuallyAlways(constraint TemporalConstraint, stateHashes []interface{}, casStore cas.CAS, isCycle bool) (PropertyResult, error) {
	n := len(stateHashes)
	if n == 0 {
		return PropertyResult{Success: false, Name: constraint.Name, Message: "Empty trace"}, nil
	}

	// Evaluate property at each state in the trace
	propValues := make([]bool, n)
	for i, hashInterface := range stateHashes {
		// Type assert the hash to cas.Hash
		var hash cas.Hash
		switch h := hashInterface.(type) {
		case cas.Hash:
			hash = h
		default:
			return PropertyResult{}, fmt.Errorf("invalid hash type at position %d: %T", i, hashInterface)
		}

		// Retrieve state from CAS
		state, err := cas.Retrieve[*interp.State](casStore, hash)
		if err != nil {
			return PropertyResult{}, fmt.Errorf("failed to retrieve state %d: %w", i, err)
		}

		// Evaluate property at this state
		result, err := constraint.Property.Check(state)
		if err != nil {
			return PropertyResult{}, fmt.Errorf("error evaluating property at state %d: %w", i, err)
		}
		propValues[i] = result.Success
	}

	// Check EventuallyAlways: ∃k. ∀j≥k. P(j) = true
	// "There exists a point k where property becomes true and stays true forever"

	if isCycle {
		// For cycles: find the loop start point within the current trace
		finalHash := stateHashes[n-1]
		loopStart := -1
		for i := 0; i < n-1; i++ {
			if stateHashes[i] == finalHash {
				loopStart = i
				break
			}
		}

		if loopStart == -1 {
			// The "cycle" is actually a revisit to a state from a different branch
			// Treat this as a terminating trace for the purpose of EventuallyAlways
			// (This trace ends by reaching an already-explored state)
			for k := 0; k < n; k++ {
				allTrueFromK := true
				for j := k; j < n; j++ {
					if !propValues[j] {
						allTrueFromK = false
						break
					}
				}
				if allTrueFromK {
					return PropertyResult{Success: true, Name: constraint.Name}, nil
				}
			}

			return PropertyResult{
				Success: false,
				Name:    constraint.Name,
				Message: fmt.Sprintf("%s: property never becomes permanently true (checked %d states, reaches previously-visited state)", constraint.Name, n),
			}, nil
		}

		// True cycle within this trace: check if property is true from some point onwards
		// including all states in the loop
		for k := 0; k < n; k++ {
			allTrue := true
			// Check from k onwards (including loop)
			for j := k; j < n; j++ {
				if !propValues[j] {
					allTrue = false
					break
				}
			}
			if allTrue {
				return PropertyResult{Success: true, Name: constraint.Name}, nil
			}
		}

		return PropertyResult{
			Success: false,
			Name:    constraint.Name,
			Message: fmt.Sprintf("%s: property never becomes permanently true (checked %d states with loop at %d)", constraint.Name, n, loopStart),
		}, nil

	} else {
		// For terminating traces: check if there's a point where property is true
		// from there to the end
		for k := 0; k < n; k++ {
			allTrueFromK := true
			for j := k; j < n; j++ {
				if !propValues[j] {
					allTrueFromK = false
					break
				}
			}
			if allTrueFromK {
				return PropertyResult{Success: true, Name: constraint.Name}, nil
			}
		}

		return PropertyResult{
			Success: false,
			Name:    constraint.Name,
			Message: fmt.Sprintf("%s: property never becomes permanently true (checked %d states, terminating)", constraint.Name, n),
		}, nil
	}
}
