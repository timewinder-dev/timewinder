package model

import (
	"fmt"

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

	// Build trace step for this execution
	trace := TraceStep{ThreadRan: t.ToRun, StateHash: stateHash}

	// Generate successor thunks for each runnable thread
	var out []*Thunk
	for i, r := range state.PauseReason {
		switch r {
		case interp.Finished:
			// Thread has finished - no successors
			continue

		case interp.Waiting:
			fallthrough
		case interp.WeaklyFairWaiting:
			// Re-check if wait condition is now satisfied
			satisfied, err := evaluateWaitCondition(state, i, exec.Program)
			if err != nil {
				return nil, fmt.Errorf("Error evaluating wait condition for thread %d: %w", i, err)
			}
			if !satisfied {
				// Still waiting - not runnable
				continue
			}
			// Condition now satisfied - thread is runnable
			for _, s := range states {
				out = append(out, &Thunk{
					ToRun: i,
					State: s,
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
					ToRun: i,
					State: s,
					Trace: append(t.Trace, trace),
				}
				out = append(out, newThunk)
			}
		}
	}
	return out, nil
}

// evaluateWaitCondition re-executes the wait condition to check if it's now true
func evaluateWaitCondition(state *interp.State, threadIdx int, prog *vm.Program) (bool, error) {
	currentFrame := state.Stacks[threadIdx].CurrentStack()
	if currentFrame.WaitCondition == nil {
		return false, fmt.Errorf("Thread %d in Waiting state but WaitCondition is nil", threadIdx)
	}

	// Clone state for test evaluation (don't modify original)
	testState := state.Clone()
	testFrame := testState.Stacks[threadIdx].CurrentStack()

	// Rewind PC to condition start
	testFrame.PC = currentFrame.WaitCondition.ConditionPC

	// Execute this thread until it pauses
	_, err := interp.RunToPause(prog, testState, threadIdx)
	if err != nil {
		return false, fmt.Errorf("Error re-evaluating wait condition: %w", err)
	}

	// Check why it paused
	newReason := testState.PauseReason[threadIdx]
	if newReason == interp.Waiting || newReason == interp.WeaklyFairWaiting {
		// Still waiting - condition is still false
		return false, nil
	}

	// Paused for a different reason or finished - condition must be true
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
