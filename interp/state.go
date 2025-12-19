package interp

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/vm"
)

func NewState() *State {
	return &State{
		Globals: &StackFrame{
			Stack: []vm.Value{},
		},
		ThreadSets: []ThreadSet{},
	}
}

// ThreadCount returns the total number of threads across all ThreadSets
func (s *State) ThreadCount() int {
	total := 0
	for _, ts := range s.ThreadSets {
		total += len(ts.Stacks)
	}
	return total
}

// GetStackFrames returns the stack frames for the specified thread
func (s *State) GetStackFrames(tid ThreadID) StackFrames {
	return s.ThreadSets[tid.SetIdx].Stacks[tid.LocalIdx]
}

// GetPauseReason returns the pause reason for the specified thread
func (s *State) GetPauseReason(tid ThreadID) Pause {
	return s.ThreadSets[tid.SetIdx].PauseReason[tid.LocalIdx]
}

// SetPauseReason sets the pause reason for the specified thread
func (s *State) SetPauseReason(tid ThreadID, pause Pause) {
	s.ThreadSets[tid.SetIdx].PauseReason[tid.LocalIdx] = pause
}

func (s *State) Clone() *State {
	out := &State{
		Globals:    s.Globals.Clone(),
		ThreadSets: make([]ThreadSet, len(s.ThreadSets)),
	}
	for i, ts := range s.ThreadSets {
		// Clone each thread's stack in the set
		stacks := make([]StackFrames, len(ts.Stacks))
		for j, stack := range ts.Stacks {
			var newStack []*StackFrame
			for _, frame := range stack {
				newStack = append(newStack, frame.Clone())
			}
			stacks[j] = newStack
		}
		// Copy pause reasons
		pauseReasons := make([]Pause, len(ts.PauseReason))
		copy(pauseReasons, ts.PauseReason)

		out.ThreadSets[i] = ThreadSet{
			Stacks:      stacks,
			PauseReason: pauseReasons,
		}
	}
	return out
}

func (s *State) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, s)
}

func (s *State) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

func (s *State) AddThread(frame *StackFrame) {
	// Create a singleton ThreadSet for the new thread (Phase 1 approach)
	threadSet := ThreadSet{
		Stacks:      []StackFrames{[]*StackFrame{frame}},
		PauseReason: []Pause{Start},
	}
	s.ThreadSets = append(s.ThreadSets, threadSet)
}

func (f *StackFrame) Pop() vm.Value {
	if len(f.Stack) == 0 {
		panic("Stack underrun")
	}
	v := f.Stack[len(f.Stack)-1]
	f.Stack = f.Stack[:len(f.Stack)-1]
	return v
}

func (f *StackFrame) Push(v vm.Value) {
	f.Stack = append(f.Stack, v)
}

func (f *StackFrame) Clone() *StackFrame {
	out := &StackFrame{
		PC:    f.PC,
		Stack: []vm.Value{},
	}
	for _, v := range f.Stack {
		out.Stack = append(out.Stack, v.Clone())
	}
	for k, v := range f.Variables {
		out.StoreVar(k, v.Clone())
	}
	for _, i := range f.IteratorStack {
		out.IteratorStack = append(out.IteratorStack, i.Clone())
	}
	if f.WaitCondition != nil {
		out.WaitCondition = &WaitConditionInfo{
			ConditionPC:  f.WaitCondition.ConditionPC,
			IsWeaklyFair: f.WaitCondition.IsWeaklyFair,
		}
	}
	return out
}

func (f *StackFrame) StoreVar(key string, value vm.Value) {
	if f.Variables == nil {
		f.Variables = make(map[string]vm.Value)
	}
	f.Variables[key] = value
}

func (f *StackFrame) Has(key string) bool {
	if f.Variables == nil {
		return false
	}
	_, ok := f.Variables[key]
	return ok
}

// FormatValue formats a vm.Value for display
func FormatValue(v vm.Value) string {
	switch val := v.(type) {
	case vm.IntValue:
		return fmt.Sprintf("%d", val)
	case vm.FloatValue:
		return fmt.Sprintf("%g", val)
	case vm.BoolValue:
		if val {
			return "true"
		}
		return "false"
	case vm.StrValue:
		return fmt.Sprintf("%q", string(val))
	case vm.NoneValue:
		return "None"
	case vm.FnPtrValue:
		return fmt.Sprintf("<function@0x%x>", val)
	case vm.BuiltinValue:
		return fmt.Sprintf("<builtin:%s>", val.Name)
	case vm.ArrayValue:
		if len(val) == 0 {
			return "[]"
		}
		result := "["
		for i, elem := range val {
			if i > 0 {
				result += ", "
			}
			if i >= 5 {
				result += fmt.Sprintf("... (%d more)", len(val)-i)
				break
			}
			result += FormatValue(elem)
		}
		result += "]"
		return result
	case vm.StructValue:
		if len(val) == 0 {
			return "{}"
		}
		result := "{"
		count := 0
		// Sort keys for consistent output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if count > 0 {
				result += ", "
			}
			if count >= 5 {
				result += fmt.Sprintf("... (%d more)", len(val)-count)
				break
			}
			result += fmt.Sprintf("%s: %s", k, FormatValue(val[k]))
			count++
		}
		result += "}"
		return result
	case vm.NonDetValue:
		return fmt.Sprintf("<nondet:%d choices>", len(val.Choices))
	default:
		return fmt.Sprintf("<%T>", v)
	}
}

// PrettyPrint returns a formatted string representation of the State
func (s *State) PrettyPrint(prog *vm.Program) string {
	var b strings.Builder
	s.PrettyPrintTo(&b, prog)
	return b.String()
}

// PrettyPrintTo writes a formatted representation of the State directly to w
func (s *State) PrettyPrintTo(w io.Writer, prog *vm.Program) {
	// Print global variables (excluding builtins and functions)
	if s.Globals != nil && len(s.Globals.Variables) > 0 {
		fmt.Fprint(w, "Global Variables:\n")

		// Sort keys for consistent output
		keys := make([]string, 0, len(s.Globals.Variables))
		for k := range s.Globals.Variables {
			// Skip builtins and functions
			v := s.Globals.Variables[k]
			if _, isBuiltin := v.(vm.BuiltinValue); isBuiltin {
				continue
			}
			if _, isFn := v.(vm.FnPtrValue); isFn {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if len(keys) == 0 {
			fmt.Fprint(w, "  (none)\n")
		} else {
			for _, k := range keys {
				v := s.Globals.Variables[k]
				fmt.Fprintf(w, "  %s = %s\n", k, FormatValue(v))
			}
		}
	}

	// Print per-thread state
	if s.ThreadCount() > 0 {
		fmt.Fprint(w, "\nThread States:\n")
		threadIdx := 0
		for setIdx, threadSet := range s.ThreadSets {
			for localIdx, stack := range threadSet.Stacks {
				pauseReason := threadSet.PauseReason[localIdx]
				fmt.Fprintf(w, "  Thread %d (Set %d, Local %d) [%s]:\n", threadIdx, setIdx, localIdx, pauseReason)

				// Show pause location and context
				if len(stack) > 0 {
					currentFrame := stack[len(stack)-1]
					pc := currentFrame.PC

					// Show source location if available
					if prog != nil {
						lineNum := prog.GetLineNumber(pc)
						filename := prog.GetFilename(pc)
						if lineNum > 0 && filename != "" {
							basename := filepath.Base(filename)
							fmt.Fprintf(w, "    Location: %s:%d\n", basename, lineNum)
						} else if lineNum > 0 {
							fmt.Fprintf(w, "    Location: line %d\n", lineNum)
						} else {
							fmt.Fprintf(w, "    Location: %s\n", pc)
						}
					} else {
						fmt.Fprintf(w, "    Location: %s\n", pc)
					}

					// If yielded, show the step name from top of stack
					if pauseReason == Yield && len(currentFrame.Stack) > 0 {
						topValue := currentFrame.Stack[len(currentFrame.Stack)-1]
						if stepName, ok := topValue.(vm.StrValue); ok {
							fmt.Fprintf(w, "    Step: %s\n", stepName)
						}
					}
				}

				// Show thread-local variables from all frames
				hasLocalVars := false
				for frameIdx, frame := range stack {
					if len(frame.Variables) > 0 {
						hasLocalVars = true
						if frameIdx > 0 {
							fmt.Fprintf(w, "    Frame %d:\n", frameIdx)
						} else {
							fmt.Fprint(w, "    Local variables:\n")
						}

						// Sort keys
						keys := make([]string, 0, len(frame.Variables))
						for k := range frame.Variables {
							keys = append(keys, k)
						}
						sort.Strings(keys)

						for _, k := range keys {
							v := frame.Variables[k]
							if frameIdx > 0 {
								fmt.Fprintf(w, "      %s = %s\n", k, FormatValue(v))
							} else {
								fmt.Fprintf(w, "      %s = %s\n", k, FormatValue(v))
							}
						}
					}
				}

				if !hasLocalVars {
					fmt.Fprint(w, "    (no local variables)\n")
				}

				threadIdx++
			}
		}
	}
}
