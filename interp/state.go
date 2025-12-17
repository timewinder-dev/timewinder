package interp

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/vm"
)

func NewState() *State {
	return &State{
		Globals: &StackFrame{},
	}
}

func (s *State) Clone() *State {
	out := &State{
		Globals: s.Globals.Clone(),
	}
	for _, stack := range s.Stacks {
		var new []*StackFrame
		for _, x := range stack {
			new = append(new, x.Clone())
		}
		out.Stacks = append(out.Stacks, new)
	}
	for _, p := range s.PauseReason {
		out.PauseReason = append(out.PauseReason, p)
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
	s.Stacks = append(s.Stacks, []*StackFrame{frame})
	s.PauseReason = append(s.PauseReason, Start)
}

func (f *StackFrame) Pop() vm.Value {
	if len(f.Stack) == 0 {
		panic("Stack underrun")
		//return vm.None
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
		PC: f.PC,
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
	var result string

	// Print global variables (excluding builtins and functions)
	if s.Globals != nil && len(s.Globals.Variables) > 0 {
		result += "Global Variables:\n"

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
			result += "  (none)\n"
		} else {
			for _, k := range keys {
				v := s.Globals.Variables[k]
				result += fmt.Sprintf("  %s = %s\n", k, FormatValue(v))
			}
		}
	}

	// Print per-thread state
	if len(s.Stacks) > 0 {
		result += "\nThread States:\n"
		for i, stack := range s.Stacks {
			pauseReason := s.PauseReason[i]
			result += fmt.Sprintf("  Thread %d [%s]:\n", i, pauseReason)

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
						result += fmt.Sprintf("    Location: %s:%d\n", basename, lineNum)
					} else if lineNum > 0 {
						result += fmt.Sprintf("    Location: line %d\n", lineNum)
					} else {
						result += fmt.Sprintf("    Location: %s\n", pc)
					}
				} else {
					result += fmt.Sprintf("    Location: %s\n", pc)
				}

				// If yielded, show the step name from top of stack
				if pauseReason == Yield && len(currentFrame.Stack) > 0 {
					topValue := currentFrame.Stack[len(currentFrame.Stack)-1]
					if stepName, ok := topValue.(vm.StrValue); ok {
						result += fmt.Sprintf("    Step: %s\n", stepName)
					}
				}
			}

			// Show thread-local variables from all frames
			hasLocalVars := false
			for frameIdx, frame := range stack {
				if len(frame.Variables) > 0 {
					hasLocalVars = true
					if frameIdx > 0 {
						result += fmt.Sprintf("    Frame %d:\n", frameIdx)
					} else {
						result += "    Local variables:\n"
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
							result += fmt.Sprintf("      %s = %s\n", k, FormatValue(v))
						} else {
							result += fmt.Sprintf("      %s = %s\n", k, FormatValue(v))
						}
					}
				}
			}

			if !hasLocalVars {
				result += "    (no local variables)\n"
			}
		}
	}

	return result
}
