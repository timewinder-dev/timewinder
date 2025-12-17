package interp

import "github.com/timewinder-dev/timewinder/vm"

func RunToEnd(prog *vm.Program, global *StackFrame, start *StackFrame) (vm.Value, error) {
	frames := []*StackFrame{start}
	for {
		c, n, err := Step(prog, global, frames)
		if err != nil {
			return nil, err
		}
		switch c {
		case ReturnStep:
			if len(frames) == 1 {
				val := start.Pop()
				start.Stack = nil
				return val, nil
			} else {
				f := frames[len(frames)-1]
				frames = frames[:len(frames)-1]
				frames[len(frames)-1].Push(f.Pop())
		// Increment PC to move past CALL instruction
		frames[len(frames)-1].PC = frames[len(frames)-1].PC.Inc()
			}
		case EndStep:
			if len(frames) == 1 {
				start.Stack = nil
				return vm.None, nil
			} else {
				frames = frames[:len(frames)-1]
				frames[len(frames)-1].Push(vm.None)
			// Increment PC to move past CALL instruction
			frames[len(frames)-1].PC = frames[len(frames)-1].PC.Inc()

			}
		case CallStep:
			newf, err := BuildCallFrame(prog, frames[len(frames)-1], n)
			if err != nil {
				return nil, err
			}
			// Only append frame if it's not nil (builtins return nil)
		if newf != nil {
			frames = append(frames, newf)
		}
		}
	}
}

func RunToPause(prog *vm.Program, s *State, thread int) ([]vm.Value, error) {
	for {
		res, n, err := Step(prog, s.Globals, s.Stacks[thread])
		if err != nil {
			return nil, err
		}
		switch res {
		case ReturnStep:
			if len(s.Stacks[thread]) == 1 {
				s.PauseReason[thread] = Finished
				return nil, nil
			}
			f := s.Stacks[thread].PopStack()
			val := f.Pop()
			s.Stacks[thread].CurrentStack().Push(val)
		case CallStep:
			currentFrame := s.Stacks[thread].CurrentStack()
			f, err := BuildCallFrame(prog, currentFrame, n)
			if err != nil {
				return nil, err
			}

			// Check if builtin returned NonDetValue (immediate expansion needed)
			// For builtins, f will be nil and result will be on stack
			if f == nil && len(currentFrame.Stack) > 0 {
				if nonDet, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.NonDetValue); ok {
					// Pop the NonDetValue from stack
					currentFrame.Pop()
					s.PauseReason[thread] = NonDet
					return nonDet.Choices, nil
				}
			}

			// Normal function call
			if f != nil {
				s.Stacks[thread].Append(f)
			}
		case ContinueStep:
			continue
		case EndStep:
			s.PauseReason[thread] = Finished
			return nil, nil
		case YieldStep:
			s.PauseReason[thread] = Yield
			return nil, nil
		default:
			panic("unhandled intermediate step")
		}
	}
}
