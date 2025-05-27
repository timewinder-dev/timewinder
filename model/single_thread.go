package model

import "github.com/timewinder-dev/timewinder/interp"

type SingleThreadEngine struct {
	Queue     []*Thunk
	NextQueue []*Thunk
	Executor  *Executor
}

func InitSingleThread(exec *Executor) (*SingleThreadEngine, error) {
	allStates, err := interp.Canonicalize(exec.InitialState)
	if err != nil {
		return nil, err
	}
	st := &SingleThreadEngine{
		Executor: exec,
	}
	for _, s := range allStates {
		err := CheckProperties(s, exec.Properties)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(exec.Threads); i++ {
			st.Queue = append(st.Queue, &Thunk{
				ToRun: i,
				State: s.Clone(),
			})
		}
	}
	return st, nil
}

func (s *SingleThreadEngine) RunModel() error {
	for {
		for len(s.Queue) != 0 {
			t := s.Queue[0]
			s.Queue = s.Queue[1:]
			st, err := RunTrace(t, s.Executor.Program)
			if err != nil {
				return err
			}
			err = CheckProperties(st, s.Executor.Properties)
			if err != nil {
				return err
			}
			next, err := BuildRunnable(t, st, 0)
			if err != nil {
				return err
			}
			s.NextQueue = append(s.NextQueue, next...)
		}
		if len(s.NextQueue) == 0 {
			break
		}
		s.Queue = s.NextQueue
		s.NextQueue = nil
	}
	return nil
}
