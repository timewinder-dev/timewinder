package interp

import (
	"github.com/rs/zerolog/log"
	"github.com/timewinder-dev/timewinder/vm"
)

func RunToEnd(prog Program, global *StackFrame, start *StackFrame) (vm.Value, error) {
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
			newf, err := buildCallFrameWithProgram(prog, frames[len(frames)-1], n)
			if err != nil {
				return nil, err
			}
			// Only append frame if it's not nil (builtins return nil)
		if newf != nil {
			frames = append(frames, newf)
		}
		case MethodCallStep:
			err := BuildMethodCallFrame(frames[len(frames)-1], n)
			if err != nil {
				return nil, err
			}
			// BuildMethodCallFrame already incremented PC, just continue
		}
	}
}

func RunToPause(prog *vm.Program, s *State, thread ThreadID) ([]vm.Value, error) {
	log.Trace().Interface("thread", thread).Msg("RunToPause: starting thread execution")
	threadStack := s.GetStackFrames(thread)

	// Helper to save threadStack back to state
	saveStack := func() {
		s.ThreadSets[thread.SetIdx].Stacks[thread.LocalIdx] = threadStack
	}
	defer saveStack() // Always save on return

	stepCount := 0
	for {
		stepCount++
		res, n, err := Step(prog, s.Globals, threadStack)
		if err != nil {
			log.Trace().Interface("thread", thread).Int("step", stepCount).Err(err).Msg("RunToPause: step error")
			return nil, err
		}

		log.Trace().Interface("thread", thread).Int("step", stepCount).Str("result", resultToString(res)).Int("n", n).Msg("RunToPause: step result")

		switch res {
		case ReturnStep:
			if len(threadStack) == 1 {
				log.Trace().Interface("thread", thread).Msg("RunToPause: thread finished")
				s.SetPauseReason(thread, Finished)
				return nil, nil
			}
			f := threadStack.PopStack()
			val := f.Pop()
			threadStack.CurrentStack().Push(val)
			log.Trace().Interface("thread", thread).Interface("return_value", val).Int("stack_depth", len(threadStack)).Msg("RunToPause: function returned")
		case CallStep:
			currentFrame := threadStack.CurrentStack()
			f, err := BuildCallFrame(prog, currentFrame, n)
			if err != nil {
				log.Trace().Interface("thread", thread).Err(err).Msg("RunToPause: call error")
				return nil, err
			}

			// Check if builtin returned NonDetValue (immediate expansion needed)
			// For builtins, f will be nil and result will be on stack
			//
			// IMPORTANT: NonDet handling is fundamentally different from Yield handling
			// - Yield: Creates an interleaving point where scheduler picks WHICH thread runs next
			//   Result: 1 successor state for each runnable thread
			// - NonDet: Branches the state space where THIS thread explores ALL choices
			//   Result: N successor states (one per choice) all running the same thread
			//
			// This is why NonDet cannot be unified with Yield - they represent different
			// types of non-determinism (scheduler choice vs. program choice)
			if f == nil && len(currentFrame.Stack) > 0 {
				if nonDet, ok := currentFrame.Stack[len(currentFrame.Stack)-1].(vm.NonDetValue); ok {
					// Pop the NonDetValue from stack
					currentFrame.Pop()
					log.Trace().Interface("thread", thread).Interface("choices", nonDet.Choices).Msg("RunToPause: non-deterministic choice")
					s.SetPauseReason(thread, NonDet)
					return nonDet.Choices, nil
				}
			}

			// Normal function call - increment caller's PC and push new frame
			if f != nil {
				currentFrame.PC = currentFrame.PC.Inc()
				threadStack.Append(f)
				log.Trace().Interface("thread", thread).Int("stack_depth", len(threadStack)).Msg("RunToPause: pushed call frame")
			}
		case MethodCallStep:
			currentFrame := threadStack.CurrentStack()
			err := BuildMethodCallFrame(currentFrame, n)
			if err != nil {
				log.Trace().Interface("thread", thread).Err(err).Msg("RunToPause: method call error")
				return nil, err
			}
			// Method already incremented PC, just continue
			log.Trace().Interface("thread", thread).Msg("RunToPause: method call completed")
		case ContinueStep:
			continue
		case EndStep:
			if len(threadStack) == 1 {
				log.Trace().Interface("thread", thread).Msg("RunToPause: thread finished (end)")
				s.SetPauseReason(thread, Finished)
				return nil, nil
			}
			// Function ended without explicit return - pop frame and push None
			threadStack.PopStack()
			threadStack.CurrentStack().Push(vm.None)
			log.Trace().Interface("thread", thread).Int("stack_depth", len(threadStack)).Msg("RunToPause: function ended without return")
		case YieldStep:
			// Check yield type to set appropriate pause reason
			// NEW: Use Runnable/Blocked instead of old pause reasons
			var pause Pause
			var weaklyFair bool
			var stronglyFair bool

			// Check if this thread is configured as fair
			threadIsFair := s.ThreadSets[thread.SetIdx].Fair
			threadIsStrongFair := s.ThreadSets[thread.SetIdx].StrongFair

			yieldType := YieldType(n)

			// Auto-upgrade based on thread configuration
			// Strong fairness supersedes weak fairness
			if threadIsStrongFair {
				switch yieldType {
				case YieldNormal:
					yieldType = YieldStronglyFair
				case YieldWaiting:
					yieldType = YieldStronglyFairWaiting
				case YieldWeaklyFair:
					yieldType = YieldStronglyFair // Strong supersedes weak
				case YieldWeaklyFairWaiting:
					yieldType = YieldStronglyFairWaiting // Strong supersedes weak
				}
			} else if threadIsFair {
				// If thread is fair, upgrade yield to weakly fair variant
				switch yieldType {
				case YieldNormal:
					yieldType = YieldWeaklyFair
				case YieldWaiting:
					yieldType = YieldWeaklyFairWaiting
				}
			}

			switch yieldType {
			case YieldStronglyFair:
				pause = Runnable
				weaklyFair = false
				stronglyFair = true
			case YieldStronglyFairWaiting:
				pause = Blocked
				weaklyFair = false
				stronglyFair = true
			case YieldWeaklyFair:
				pause = Runnable
				weaklyFair = true
				stronglyFair = false
			case YieldWeaklyFairWaiting:
				pause = Blocked
				weaklyFair = true
				stronglyFair = false
			case YieldWaiting:
				pause = Blocked
				weaklyFair = false
				stronglyFair = false
			default:
				pause = Runnable
				weaklyFair = false
				stronglyFair = false
			}
			log.Trace().Interface("thread", thread).Str("pause", pause.String()).Bool("weakly_fair", weaklyFair).Bool("strongly_fair", stronglyFair).Bool("thread_is_fair", threadIsFair).Bool("thread_is_strong_fair", threadIsStrongFair).Msg("RunToPause: thread yielded")
			s.SetPauseReason(thread, pause)
			s.SetWeaklyFair(thread, weaklyFair)
			s.SetStronglyFair(thread, stronglyFair)
			return nil, nil
		default:
			panic("unhandled intermediate step")
		}
	}
}

func resultToString(res StepResult) string {
	switch res {
	case ContinueStep:
		return "Continue"
	case ReturnStep:
		return "Return"
	case EndStep:
		return "End"
	case CallStep:
		return "Call"
	case ErrorStep:
		return "Error"
	case YieldStep:
		return "Yield"
	case NonDetStep:
		return "NonDet"
	default:
		return "Unknown"
	}
}
