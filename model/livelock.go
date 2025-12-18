package model

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"

	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

// WeakStateHash computes a hash based only on "semantic" state:
// - Global variables
// - Thread pause reasons (not PC or stack)
// This helps detect livelocks where threads cycle through different PCs
// but the actual program state is equivalent
func WeakStateHash(st *interp.State) (cas.Hash, error) {
	type WeakState struct {
		Globals      map[string]interface{}
		PauseReasons []interp.Pause
	}

	// Extract just the global variables (serialize to get consistent representation)
	globalsJSON, err := json.Marshal(st.Globals.Variables)
	if err != nil {
		return 0, err
	}

	weak := WeakState{
		Globals:      make(map[string]interface{}),
		PauseReasons: st.PauseReason,
	}

	// Parse globals back to get normalized representation
	json.Unmarshal(globalsJSON, &weak.Globals)

	// Compute hash
	data, err := json.Marshal(weak)
	if err != nil {
		return 0, err
	}

	hash := sha256.Sum256(data)
	return cas.Hash(hash[0])<<56 | cas.Hash(hash[1])<<48 | cas.Hash(hash[2])<<40 | cas.Hash(hash[3])<<32 |
		cas.Hash(hash[4])<<24 | cas.Hash(hash[5])<<16 | cas.Hash(hash[6])<<8 | cas.Hash(hash[7]), nil
}

// DetectLivelock checks if we're seeing the same weak state repeatedly,
// which indicates a livelock (cycling through states with different PCs but same semantics)
func DetectLivelock(exec *Executor, st *interp.State, currentDepth int) (bool, error) {
	weakHash, err := WeakStateHash(st)
	if err != nil {
		return false, err
	}

	depths, seen := exec.WeakStateHistory[weakHash]
	if !seen {
		// First time seeing this weak state - store a sample
		exec.WeakStateHistory[weakHash] = []int{currentDepth}
		exec.WeakStateSamples[weakHash] = st.Clone()
		return false, nil
	}

	// We've seen this weak state before - add current depth
	exec.WeakStateHistory[weakHash] = append(depths, currentDepth)

	// Check if we're in a repeating pattern (seen at least 3 times)
	if len(exec.WeakStateHistory[weakHash]) >= 3 {
		allDepths := exec.WeakStateHistory[weakHash]
		// Check if there's a consistent cycle length
		lastThree := allDepths[len(allDepths)-3:]
		cycle1 := lastThree[1] - lastThree[0]
		cycle2 := lastThree[2] - lastThree[1]

		if cycle1 == cycle2 && cycle1 > 0 {
			fmt.Fprintf(exec.DebugWriter, "\n⚠ Potential livelock detected!\n")
			fmt.Fprintf(exec.DebugWriter, "  Weak state hash: %x\n", weakHash)
			fmt.Fprintf(exec.DebugWriter, "  Seen at depths: %v\n", allDepths)
			fmt.Fprintf(exec.DebugWriter, "  Cycle length: %d\n", cycle1)

			// Show differences between first and current state
			if sample, ok := exec.WeakStateSamples[weakHash]; ok {
				fmt.Fprintf(exec.DebugWriter, "\n  Differences between states:\n")
				ShowStateDifferences(exec.DebugWriter, sample, st, exec.Threads)
			}

			return true, nil
		}
	}

	return false, nil
}

// ShowStateDifferences displays the differences between two states
func ShowStateDifferences(w io.Writer, state1, state2 *interp.State, threadNames []string) {
	// Compare thread states
	for i := 0; i < len(state1.Stacks); i++ {
		if i >= len(state2.Stacks) {
			fmt.Fprintf(w, "    Thread %d (%s): missing in state2\n", i, threadNames[i])
			continue
		}

		frame1 := state1.Stacks[i].CurrentStack()
		frame2 := state2.Stacks[i].CurrentStack()

		if frame1 == nil && frame2 == nil {
			continue
		}

		if frame1 == nil || frame2 == nil {
			fmt.Fprintf(w, "    Thread %d (%s): frame difference (nil vs non-nil)\n", i, threadNames[i])
			continue
		}

		// Check PC differences
		if frame1.PC != frame2.PC {
			fmt.Fprintf(w, "    Thread %d (%s): PC differs (%v vs %v)\n",
				i, threadNames[i], frame1.PC, frame2.PC)
		}

		// Check stack depth differences
		if len(frame1.Stack) != len(frame2.Stack) {
			fmt.Fprintf(w, "    Thread %d (%s): Stack depth differs (%d vs %d)\n",
				i, threadNames[i], len(frame1.Stack), len(frame2.Stack))
		}

		// Check local variables
		if len(frame1.Variables) != len(frame2.Variables) {
			fmt.Fprintf(w, "    Thread %d (%s): Local variable count differs (%d vs %d)\n",
				i, threadNames[i], len(frame1.Variables), len(frame2.Variables))
		} else {
			for k, v1 := range frame1.Variables {
				if v2, ok := frame2.Variables[k]; ok {
					v1JSON, _ := json.Marshal(v1)
					v2JSON, _ := json.Marshal(v2)
					if string(v1JSON) != string(v2JSON) {
						fmt.Fprintf(w, "    Thread %d (%s): Variable '%s' differs (%s vs %s)\n",
							i, threadNames[i], k, v1JSON, v2JSON)
					}
				} else {
					fmt.Fprintf(w, "    Thread %d (%s): Variable '%s' only in state1\n",
						i, threadNames[i], k)
				}
			}
		}

		// Check call stack depth
		if len(state1.Stacks[i]) != len(state2.Stacks[i]) {
			fmt.Fprintf(w, "    Thread %d (%s): Call stack depth differs (%d vs %d frames)\n",
				i, threadNames[i], len(state1.Stacks[i]), len(state2.Stacks[i]))
		}
	}
}
