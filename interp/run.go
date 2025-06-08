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
			}
		case EndStep:
			if len(frames) == 1 {
				start.Stack = nil
				return vm.None, nil
			} else {
				frames = frames[:len(frames)-1]
				frames[len(frames)-1].Push(vm.None)

			}
		case CallStep:
			newf, err := BuildCallFrame(prog, frames[len(frames)-1], n)
			if err != nil {
				return nil, err
			}
			frames = append(frames, newf)
		}
	}
}

func RunToPause(prog *vm.Program, s *State, thread int) (StepResult, error) {
	for {
		res, n, err := Step(prog, s.Globals, s.Stacks[thread])
		if err != nil {
			return ErrorStep, err
		}
		switch res {
		case ReturnStep:
			if len(s.Stacks[thread]) == 1 {
				s.PauseReason[thread] = Finished
				return EndStep, nil
			}
			f := s.Stacks[thread].PopStack()
			val := f.Pop()
			s.Stacks[thread].CurrentStack().Push(val)
		case CallStep:
			f, err := BuildCallFrame(prog, s.Stacks[thread].CurrentStack(), n)
			if err != nil {
				return ErrorStep, err
			}
			s.Stacks[thread].Append(f)
		case ContinueStep:
			continue
		case EndStep:
			s.PauseReason[thread] = Finished
			return EndStep, nil
		case YieldStep:
			s.PauseReason[thread] = Yield
			return YieldStep, nil
		default:
			panic("unhandled intermediate step")
		}
	}
}
