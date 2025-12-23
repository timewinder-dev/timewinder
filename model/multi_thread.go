package model

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gookit/color"
	"github.com/rs/zerolog/log"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

// MultiThreadEngine implements parallel model checking with separate execution and checking workers.
type MultiThreadEngine struct {
	// Configuration
	Executor        *Executor
	numExecThreads  int
	numCheckThreads int

	// Cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Work channels
	workQueue  chan *WorkItem
	checkQueue chan *CheckItem

	// Next depth accumulation
	nextDepthMu    sync.Mutex
	nextDepthQueue []*Thunk

	// Synchronized shared state
	visitedMu    sync.RWMutex
	violationsMu sync.Mutex

	// Statistics (atomic)
	stateCount      int64
	prunedThisDepth int64
	livelockCount   int64
	depth           int

	// Depth coordination
	currentDepthQueue  []*Thunk
	remainingWorkItems int64

	// Coordination
	execWg  sync.WaitGroup
	checkWg sync.WaitGroup
}

// NewMultiThread creates a new multi-threaded model checking engine.
// numExecThreads: number of workers generating successor states
// numCheckThreads: number of workers checking properties
func NewMultiThread(executor *Executor, numExecThreads, numCheckThreads int) (*MultiThreadEngine, error) {
	// Default to NumCPU if not specified
	if numExecThreads <= 0 {
		numExecThreads = runtime.NumCPU()
	}
	if numCheckThreads <= 0 {
		numCheckThreads = runtime.NumCPU() / 2
		if numCheckThreads < 1 {
			numCheckThreads = 1
		}
	}

	m := &MultiThreadEngine{
		Executor:        executor,
		numExecThreads:  numExecThreads,
		numCheckThreads: numCheckThreads,
	}

	return m, nil
}

// Close cancels the model checking and waits for RunModel to complete.
// With long-lived workers, RunModel handles channel closing and worker shutdown.
func (m *MultiThreadEngine) Close() {
	// Cancel context - this will cause RunModel to exit and clean up
	if m.cancel != nil {
		m.cancel()
	}

	// RunModel will close channels and wait for workers when it exits
	// No additional cleanup needed here
}

// computeStatistics builds ModelStatistics from current engine state.
func (m *MultiThreadEngine) computeStatistics() ModelStatistics {
	return ModelStatistics{
		TotalTransitions: int(atomic.LoadInt64(&m.stateCount)),
		UniqueStates:     len(m.Executor.VisitedStates),
		DuplicateStates:  int(atomic.LoadInt64(&m.stateCount)) - len(m.Executor.VisitedStates),
		MaxDepth:         m.depth,
		ViolationCount:   len(m.Executor.Violations),
		LivelockCount:    int(atomic.LoadInt64(&m.livelockCount)),
	}
}

// recordViolation safely records a violation and cancels if not in keep-going mode.
func (m *MultiThreadEngine) recordViolation(violation PropertyViolation) {
	m.violationsMu.Lock()
	m.Executor.Violations = append(m.Executor.Violations, violation)
	shouldCancel := !m.Executor.KeepGoing
	m.violationsMu.Unlock()

	if shouldCancel {
		m.cancel()
	}
}

// initializeQueues creates the initial work items from the initial state.
func (m *MultiThreadEngine) initializeQueues() error {
	allStates, err := interp.Canonicalize(m.Executor.InitialState)
	if err != nil {
		return err
	}

	// Check Always properties at initial state
	alwaysProperties := FilterPropertiesByOperator(m.Executor.TemporalConstraints, Always)

	for _, s := range allStates {
		err := CheckProperties(s, alwaysProperties)
		if err != nil {
			violation := PropertyViolation{
				PropertyName: "InitialState",
				Message:      err.Error(),
				StateHash:    0,
				Depth:        0,
				StateNumber:  0,
				Trace:        nil,
				State:        s,
				Program:      m.Executor.Program,
				ThreadID:     -1,
				ThreadName:   "(initial state)",
				ShowDetails:  m.Executor.ShowDetails,
				CAS:          m.Executor.CAS,
			}

			if m.Executor.KeepGoing {
				m.Executor.Violations = append(m.Executor.Violations, violation)
			} else {
				return err
			}
		}

		// Create initial thunks for all threads and add to nextDepthQueue
		for i := 0; i < len(m.Executor.Threads); i++ {
			threadID := interp.ThreadID{}
			tmpIdx := i
			for setIdx := range s.ThreadSets {
				if tmpIdx < len(s.ThreadSets[setIdx].Stacks) {
					threadID = interp.ThreadID{SetIdx: setIdx, LocalIdx: tmpIdx}
					break
				}
				tmpIdx -= len(s.ThreadSets[setIdx].Stacks)
			}

			thunk := &Thunk{
				ToRun: threadID,
				State: s.Clone(),
			}

			m.nextDepthQueue = append(m.nextDepthQueue, thunk)
		}
	}

	return nil
}

// startWorkers launches all execution and checking worker goroutines.
func (m *MultiThreadEngine) startWorkers() {
	// Start execution workers
	for i := 0; i < m.numExecThreads; i++ {
		m.execWg.Add(1)
		go m.execWorker(i)
	}

	// Start checking workers
	for i := 0; i < m.numCheckThreads; i++ {
		m.checkWg.Add(1)
		go m.checkWorker(i)
	}
}

// RunModel executes the parallel BFS model checking algorithm.
func (m *MultiThreadEngine) RunModel() (*ModelResult, error) {
	// Initialize context for cancellation
	m.ctx, m.cancel = context.WithCancel(context.Background())
	defer m.cancel()

	atomic.StoreInt64(&m.stateCount, 0)
	m.depth = 0
	w := m.Executor.DebugWriter

	// Create buffered channels
	m.workQueue = make(chan *WorkItem, m.numExecThreads*2)
	m.checkQueue = make(chan *CheckItem, m.numCheckThreads*2)

	// Initialize with initial state thunks
	err := m.initializeQueues()
	if err != nil {
		return nil, err
	}

	// Start workers
	m.startWorkers()

	// Main depth loop
	for {
		fmt.Fprintf(w, "\n=== Depth %d: Starting parallel exploration ===\n", m.depth)
		depthStart := time.Now()
		atomic.StoreInt64(&m.prunedThisDepth, 0)

		// Transfer nextDepthQueue to currentDepthQueue
		m.nextDepthMu.Lock()
		m.currentDepthQueue = m.nextDepthQueue
		m.nextDepthQueue = nil
		m.nextDepthMu.Unlock()

		// Set remaining work items counter
		atomic.StoreInt64(&m.remainingWorkItems, int64(len(m.currentDepthQueue)))

		// Push all work items to workQueue
		for _, thunk := range m.currentDepthQueue {
			m.workQueue <- NewWorkItem(thunk, m.depth)
		}

		// Wait for all work items to be processed
		for atomic.LoadInt64(&m.remainingWorkItems) > 0 {
			// Check for cancellation
			select {
			case <-m.ctx.Done():
				fmt.Fprintf(w, "\n⚠ Model checking cancelled\n")
				return m.buildResult(), nil
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}

		// Report statistics
		if m.Executor.Reporter != nil {
			elapsed := time.Since(depthStart)
			explored := int(atomic.LoadInt64(&m.stateCount))
			pruned := int(atomic.LoadInt64(&m.prunedThisDepth))

			m.nextDepthMu.Lock()
			remaining := len(m.nextDepthQueue)
			m.nextDepthMu.Unlock()

			report := formatDepthReport(m.depth, explored, pruned, remaining, elapsed)
			m.Executor.Reporter.Printf("%s", report)
		}

		// Check for termination
		m.nextDepthMu.Lock()
		hasWork := len(m.nextDepthQueue) > 0
		m.nextDepthMu.Unlock()

		if !hasWork {
			break
		}

		// Prepare next depth
		m.depth++

		// Check max depth
		if m.Executor.MaxDepth > 0 && m.depth >= m.Executor.MaxDepth {
			fmt.Fprintf(w, "\n⚠ Reached maximum depth %d, stopping exploration\n", m.Executor.MaxDepth)
			if m.Executor.Reporter != nil {
				m.Executor.Reporter.Printf("%s Reached maximum depth %d, stopping exploration\n",
					color.Yellow.Sprint("⚠"),
					m.Executor.MaxDepth)
			}
			break
		}
	}

	// Shutdown workers
	close(m.workQueue)
	m.execWg.Wait()

	close(m.checkQueue)
	m.checkWg.Wait()

	return m.buildResult(), nil
}

// buildResult constructs the final ModelResult.
func (m *MultiThreadEngine) buildResult() *ModelResult {
	return &ModelResult{
		Statistics: m.computeStatistics(),
		Violations: m.Executor.Violations,
		Success:    len(m.Executor.Violations) == 0,
	}
}

// execWorker processes work items and generates successor states.
func (m *MultiThreadEngine) execWorker(workerID int) {
	defer m.execWg.Done()

	w := m.Executor.DebugWriter

	for {
		workItem, ok := <-m.workQueue
		if !ok {
			// Channel closed, model checking complete
			return
		}

		m.processWorkItem(workerID, workItem, w)

		// Decrement remaining work items counter
		atomic.AddInt64(&m.remainingWorkItems, -1)
	}
}

// processWorkItem handles a single work item: execute thread, check visited, generate successors.
func (m *MultiThreadEngine) processWorkItem(workerID int, workItem *WorkItem, w io.Writer) {
	t := workItem.Thunk
	atomic.AddInt64(&m.stateCount, 1)

	// Convert ThreadID to flat index for display
	threadFlatIdx := 0
	for i := 0; i < t.ToRun.SetIdx; i++ {
		threadFlatIdx += len(t.State.ThreadSets[i].Stacks)
	}
	threadFlatIdx += t.ToRun.LocalIdx

	if w != io.Discard {
		fmt.Fprintf(w, "[Worker %d] Processing thread %d (%s)\n",
			workerID, threadFlatIdx, m.Executor.Threads[threadFlatIdx])
	}

	// Execute thread
	st, choices, err := RunTrace(t, m.Executor.Program)
	if err != nil {
		log.Error().Err(err).Int("worker", workerID).Msg("Thread execution error")
		return
	}

	// Handle non-deterministic choice
	if choices != nil {
		fmt.Fprintf(w, "[Worker %d] Non-deterministic choice: expanding %d branches\n", workerID, len(choices))
		successors := make([]*Thunk, 0, len(choices))
		for _, choice := range choices {
			successor := t.Clone()
			successor.State = st.Clone()
			currentFrame := successor.State.GetStackFrames(t.ToRun).CurrentStack()
			currentFrame.Push(choice)
			successors = append(successors, successor)
		}
		// Add non-det branches to next depth queue (they're explored as successors)
		m.nextDepthMu.Lock()
		m.nextDepthQueue = append(m.nextDepthQueue, successors...)
		m.nextDepthMu.Unlock()
		return
	}

	// Hash the state (CAS is now thread-safe internally)
	stateHash, err := m.Executor.CAS.Put(st)

	if err != nil {
		log.Error().Err(err).Int("worker", workerID).Msg("Failed to hash state")
		return
	}

	// Check if visited with double-checked locking
	// Step 1: Optimistic read
	m.visitedMu.RLock()
	visited := m.Executor.VisitedStates[stateHash]
	m.visitedMu.RUnlock()

	if visited {
		// Already visited, send to check queue for cycle handling
		atomic.AddInt64(&m.prunedThisDepth, 1)
		checkItem := NewCheckItem(t, st, stateHash, false, workItem.DepthNumber, 0)
		select {
		case m.checkQueue <- checkItem:
		case <-m.ctx.Done():
			return
		}
		return
	}

	// Step 2: Acquire write lock and double-check
	m.visitedMu.Lock()
	if m.Executor.VisitedStates[stateHash] {
		// Lost the race - someone else marked it
		m.visitedMu.Unlock()
		atomic.AddInt64(&m.prunedThisDepth, 1)
		checkItem := NewCheckItem(t, st, stateHash, false, workItem.DepthNumber, 0)
		select {
		case m.checkQueue <- checkItem:
		case <-m.ctx.Done():
			return
		}
		return
	}

	// Step 3: We won the race, mark as visited
	m.Executor.VisitedStates[stateHash] = true
	m.visitedMu.Unlock()

	// Generate successors
	successors, err := BuildRunnable(t, st, stateHash, m.Executor)
	if err != nil {
		log.Error().Err(err).Int("worker", workerID).Msg("Failed to build successors")
		return
	}

	// Append successors to next depth queue
	if len(successors) > 0 {
		m.nextDepthMu.Lock()
		m.nextDepthQueue = append(m.nextDepthQueue, successors...)
		m.nextDepthMu.Unlock()
	}

	// Send to check queue (new state)
	checkItem := NewCheckItem(t, st, stateHash, true, workItem.DepthNumber, len(successors))
	select {
	case m.checkQueue <- checkItem:
	case <-m.ctx.Done():
		return
	}
}

// checkWorker processes check items and validates properties.
func (m *MultiThreadEngine) checkWorker(workerID int) {
	defer m.checkWg.Done()

	for {
		checkItem, ok := <-m.checkQueue
		if !ok {
			// Channel closed, checking complete
			return
		}

		m.processCheckItem(workerID, checkItem)
	}
}

// processCheckItem handles property checking, cycle detection, and violation recording.
func (m *MultiThreadEngine) processCheckItem(workerID int, checkItem *CheckItem) {
	if checkItem.IsNewState {
		m.checkNewState(workerID, checkItem)
	} else {
		m.checkCyclicState(workerID, checkItem)
	}
}

// checkNewState validates a newly discovered state.
func (m *MultiThreadEngine) checkNewState(workerID int, checkItem *CheckItem) {
	t := checkItem.Thunk
	st := checkItem.State

	// Check for livelock
	isLivelock, err := DetectLivelock(m.Executor, st, checkItem.DepthNumber)
	if err != nil {
		log.Warn().Err(err).Int("worker", workerID).Msg("Livelock detection failed")
	}
	if isLivelock {
		atomic.AddInt64(&m.livelockCount, 1)
	}

	// Check Always properties
	alwaysProps := FilterPropertiesByOperator(m.Executor.TemporalConstraints, Always)
	err = CheckProperties(st, alwaysProps)
	if err != nil {
		m.handlePropertyViolation(t, st, checkItem.StateHash, checkItem.DepthNumber, err)
		return
	}

	// Check stutter for non-fair threads
	err = m.checkStutter(t, st, checkItem.DepthNumber)
	if err != nil {
		return // Violation already recorded
	}

	// Handle terminating state
	if checkItem.SuccessorCount == 0 {
		m.handleTerminatingState(t, st, checkItem.DepthNumber)
	}
}

// checkCyclicState handles cycle detection and temporal property checking.
func (m *MultiThreadEngine) checkCyclicState(workerID int, checkItem *CheckItem) {
	t := checkItem.Thunk
	st := checkItem.State
	stateHash := checkItem.StateHash

	// Check for strongly fair threads
	hasStronglyFairEnabled, _ := st.HasEnabledStronglyFairThreads()
	if hasStronglyFairEnabled {
		// Invalid cycle, prune silently
		return
	}

	// Check if this is a true cycle within the trace
	isTrueCycle := false
	for _, step := range t.Trace {
		if step.StateHash == stateHash {
			isTrueCycle = true
			break
		}
	}

	// Check for deadlock
	if !m.Executor.NoDeadlocks {
		allFinished := true
		for _, threadSet := range st.ThreadSets {
			for _, reason := range threadSet.PauseReason {
				if reason != interp.Finished {
					allFinished = false
					break
				}
			}
			if !allFinished {
				break
			}
		}

		if !allFinished {
			// Build successors to check if any threads can make progress
			successors, err := BuildRunnable(t, st, stateHash, m.Executor)
			if err != nil {
				log.Error().Err(err).Int("worker", workerID).Msg("Failed to check deadlock")
				return
			}
			if len(successors) == 0 {
				err := fmt.Errorf("Deadlock detected: no threads can make progress, but not all threads have finished")
				m.handlePropertyViolation(t, st, stateHash, checkItem.DepthNumber, err)
				return
			}
		}
	}

	// Check for termination violation
	if m.Executor.Termination && isTrueCycle {
		allFinished := true
		for _, threadSet := range st.ThreadSets {
			for _, reason := range threadSet.PauseReason {
				if reason != interp.Finished {
					allFinished = false
					break
				}
			}
			if !allFinished {
				break
			}
		}
		if !allFinished {
			err := fmt.Errorf("Termination violation: cycle detected but not all threads have finished")
			m.handlePropertyViolation(t, st, stateHash, checkItem.DepthNumber, err)
			return
		}
	}

	// Check temporal constraints
	if isTrueCycle && len(m.Executor.TemporalConstraints) > 0 {
		err := CheckTemporalConstraints(t, st, m.Executor, true)

		if err != nil {
			m.handlePropertyViolation(t, st, stateHash, checkItem.DepthNumber, err)
		}
	}
}

// checkStutter checks temporal properties assuming the thread never runs again.
func (m *MultiThreadEngine) checkStutter(t *Thunk, st *interp.State, depth int) error {
	threadPauseReason := st.GetPauseReason(t.ToRun)
	weaklyFair := st.GetWeaklyFair(t.ToRun)
	stronglyFair := st.GetStronglyFair(t.ToRun)

	// Check if any strongly fair threads enabled globally
	hasStronglyFairEnabled, _ := st.HasEnabledStronglyFairThreads()
	if hasStronglyFairEnabled {
		return nil
	}

	// Only check for non-fair threads
	shouldCheck := ((threadPauseReason == interp.Runnable && !weaklyFair && !stronglyFair) ||
		(threadPauseReason == interp.Blocked && !weaklyFair && !stronglyFair))

	if !shouldCheck || len(m.Executor.TemporalConstraints) == 0 {
		return nil
	}

	// Clone state and mark as stuttering
	stutterState := st.Clone()
	stutterState.SetPauseReason(t.ToRun, interp.Stuttering)

	// Check temporal constraints (CAS is now thread-safe internally)
	err := CheckTemporalConstraints(t, stutterState, m.Executor, false)
	if err == nil {
		return nil
	}

	stateHash, _ := m.Executor.CAS.Put(stutterState)

	threadFlatIdx := 0
	for i := 0; i < t.ToRun.SetIdx; i++ {
		threadFlatIdx += len(stutterState.ThreadSets[i].Stacks)
	}
	threadFlatIdx += t.ToRun.LocalIdx

	violation := PropertyViolation{
		PropertyName: "Stutter Check",
		Message: fmt.Sprintf("Stutter check failed at thread %d (%s): %s",
			threadFlatIdx, m.Executor.Threads[threadFlatIdx], err.Error()),
		StateHash:   stateHash,
		Depth:       depth,
		StateNumber: int(atomic.LoadInt64(&m.stateCount)),
		Trace:       t.Trace,
		State:       stutterState,
		Program:     m.Executor.Program,
		ThreadID:    threadFlatIdx,
		ThreadName:  m.Executor.Threads[threadFlatIdx],
		ShowDetails: m.Executor.ShowDetails,
		CAS:         m.Executor.CAS,
	}

	m.recordViolation(violation)
	return err
}

// handleTerminatingState checks properties when no successors exist.
func (m *MultiThreadEngine) handleTerminatingState(t *Thunk, st *interp.State, depth int) {
	// Check for strongly fair threads
	hasStronglyFairEnabled, _ := st.HasEnabledStronglyFairThreads()
	if hasStronglyFairEnabled {
		return
	}

	// Check for deadlock
	if !m.Executor.NoDeadlocks {
		allFinished := true
		for _, threadSet := range st.ThreadSets {
			for _, reason := range threadSet.PauseReason {
				if reason != interp.Finished {
					allFinished = false
					break
				}
			}
			if !allFinished {
				break
			}
		}

		if !allFinished {
			stateHash, _ := m.Executor.CAS.Put(st)

			err := fmt.Errorf("Deadlock detected: no threads can make progress, but not all threads have finished")
			m.handlePropertyViolation(t, st, stateHash, depth, err)
			return
		}
	}

	// Check temporal constraints
	if len(m.Executor.TemporalConstraints) > 0 {
		err := CheckTemporalConstraints(t, st, m.Executor, false)
		if err != nil {
			stateHash, _ := m.Executor.CAS.Put(st)

			m.handlePropertyViolation(t, st, stateHash, depth, err)
		}
	}
}

// handlePropertyViolation records a property violation.
func (m *MultiThreadEngine) handlePropertyViolation(t *Thunk, st *interp.State, stateHash cas.Hash, depth int, err error) {
	threadFlatIdx := 0
	for i := 0; i < t.ToRun.SetIdx; i++ {
		threadFlatIdx += len(st.ThreadSets[i].Stacks)
	}
	threadFlatIdx += t.ToRun.LocalIdx

	violation := PropertyViolation{
		PropertyName: "Property",
		Message:      err.Error(),
		StateHash:    stateHash,
		Depth:        depth,
		StateNumber:  int(atomic.LoadInt64(&m.stateCount)),
		Trace:        t.Trace,
		State:        st,
		Program:      m.Executor.Program,
		ThreadID:     threadFlatIdx,
		ThreadName:   m.Executor.Threads[threadFlatIdx],
		ShowDetails:  m.Executor.ShowDetails,
		CAS:          m.Executor.CAS,
	}

	m.recordViolation(violation)
}
