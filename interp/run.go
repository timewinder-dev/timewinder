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
		case Return:
			if len(frames) == 1 {
				val := start.Pop()
				start.Stack = nil
				return val, nil
			} else {
				f := frames[len(frames)-1]
				frames = frames[:len(frames)-1]
				frames[len(frames)-1].Push(f.Pop())
			}
		case End:
			if len(frames) == 1 {
				start.Stack = nil
				return vm.None, nil
			} else {
				frames = frames[:len(frames)-1]
				frames[len(frames)-1].Push(vm.None)

			}
		case Call:
			newf, err := BuildCallFrame(prog, frames[len(frames)-1], n)
			if err != nil {
				return nil, err
			}
			frames = append(frames, newf)
		}
	}
}
