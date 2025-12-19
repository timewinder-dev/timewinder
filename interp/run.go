package interp

import (
	"github.com/timewinder-dev/timewinder/vm"
)

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

func RunToPause(prog *vm.Program, s *State, thread ThreadID) ([]vm.Value, error) {
	threadStack := s.GetStackFrames(thread)

	// Helper to save threadStack back to state
	saveStack := func() {
		s.ThreadSets[thread.SetIdx].Stacks[thread.LocalIdx] = threadStack
	}
	defer saveStack() // Always save on return

	for {
		res, n, err := Step(prog, s.Globals, threadStack)
		if err != nil {
			return nil, err
		}
		switch res {
		case ReturnStep:
			if len(threadStack) == 1 {
				s.SetPauseReason(thread, Finished)
				return nil, nil
			}
			f := threadStack.PopStack()
			val := f.Pop()
			threadStack.CurrentStack().Push(val)
		case CallStep:
			currentFrame := threadStack.CurrentStack()
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
					s.SetPauseReason(thread, NonDet)
					return nonDet.Choices, nil
				}
			}

			// Normal function call - increment caller's PC and push new frame
			if f != nil {
				currentFrame.PC = currentFrame.PC.Inc()
				threadStack.Append(f)
			}
		case ContinueStep:
			continue
		case EndStep:
			if len(threadStack) == 1 {
				s.SetPauseReason(thread, Finished)
				return nil, nil
			}
			// Function ended without explicit return - pop frame and push None
			threadStack.PopStack()
			threadStack.CurrentStack().Push(vm.None)
		case YieldStep:
			// Check yield type to set appropriate PauseReason
			switch YieldType(n) {
			case YieldWaiting:
				s.SetPauseReason(thread, Waiting)
			case YieldWeaklyFairWaiting:
				s.SetPauseReason(thread, WeaklyFairWaiting)
			case YieldWeaklyFair:
				s.SetPauseReason(thread, WeaklyFairYield)
			default:
				s.SetPauseReason(thread, Yield)
			}
			return nil, nil
		default:
			panic("unhandled intermediate step")
		}
	}
}
