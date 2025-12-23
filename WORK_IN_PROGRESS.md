# Multi-Threaded Model Checker - Work In Progress

**Date**: 2024-12-22
**Status**: CAS contention fixed, but new race condition discovered

## What We've Accomplished Today

1. ✅ **Implemented Multi-Threaded Engine** (`model/multi_thread.go`)
   - Separate execution and checking workers
   - Channel-based work distribution
   - Depth barrier for strict BFS semantics
   - Double-checked locking for visited states

2. ✅ **Fixed CAS Mutex Contention Bottleneck**
   - Added `sync.RWMutex` to `MemoryCAS` struct (cas/memory.go:13)
   - Protected all map operations internally
   - Removed external `casMu` from MultiThreadEngine
   - Tests that previously timed out now pass in ~0.26 seconds

3. ✅ **Created Supporting Infrastructure**
   - `model/work_items.go` - WorkItem and CheckItem types
   - `model/depth_barrier.go` - Synchronization primitive
   - `model/multi_thread_test.go` - Equivalence tests
   - CLI flags: `--parallel`, `--exec-threads`, `--check-threads`

## Current Issue: Race Condition in Livelock Detection

**Error**: `fatal error: concurrent map writes` in `model/livelock.go:66`

**Root Cause**: Multiple check workers call `DetectLivelock()` concurrently, which accesses/modifies:
- `exec.WeakStateHistory map[cas.Hash][]int` (lines 63, 66, 72, 75)
- `exec.WeakStateSamples map[cas.Hash]*interp.State` (lines 67, 89)

These maps are currently in the `Executor` struct (executor.go:59-60) with no synchronization.

## Architectural Decision Needed

**User's insight**: "this history should probably be part of the cas"

### Why This Makes Sense:
- CAS already handles state storage and hashing
- CAS is now thread-safe (we just added RWMutex)
- Weak state tracking is conceptually metadata about stored states
- Would eliminate the need for separate synchronization in Executor
- Cleaner separation of concerns

### Two Paths Forward:

**Option A: Quick Fix (30 minutes)**
- Add `WeakStateMu sync.Mutex` to Executor
- Wrap map accesses in `DetectLivelock()` with mutex
- Gets tests passing immediately
- Technical debt: synchronization spread across codebase

**Option B: Architectural Refactor (2-3 hours)**
- Move weak state tracking into CAS layer
- Add methods like `CAS.TrackWeakState(hash, depth, sample)`
- Update `DetectLivelock()` to use CAS methods
- Clean architecture, proper encapsulation
- More invasive but better long-term

## What to Do Tomorrow

### Recommended Approach:
1. **Discuss**: Which approach - quick fix or architectural refactor?
2. **If refactor**: Design the CAS API for weak state tracking
3. **Implement**: Move functionality to CAS
4. **Test**: Run full equivalence test suite with `-race` flag
5. **Validate**: Performance test to ensure 3-6x speedup on multi-core

### Files to Work With:

**If quick fix**:
- `model/executor.go` - Add WeakStateMu
- `model/livelock.go` - Protect map accesses

**If refactor**:
- `cas/interface.go` - Add weak state tracking methods to CAS interface
- `cas/memory.go` - Implement weak state tracking in MemoryCAS
- `model/livelock.go` - Refactor DetectLivelock to use CAS methods
- `model/executor.go` - Remove WeakStateHistory/WeakStateSamples fields

## Key Implementation Details

### Thread-Safe CAS (cas/memory.go)
```go
type MemoryCAS struct {
    mu   sync.RWMutex  // Protects data map
    data map[Hash][]byte
}

// Put() holds mu.Lock() for entire operation (including recursive decomposition)
// getValue(), Has(), Hash() use mu.RLock() for concurrent reads
```

### Multi-Threaded Engine Architecture
- **Execution workers**: Process `workQueue`, generate successors via `BuildRunnable`
- **Check workers**: Process `checkQueue`, validate properties, detect cycles/violations
- **Depth barrier**: Ensures all workers complete current depth before transitioning
- **Double-checked locking**: Optimistic read for visited states, write lock only if needed

### Test Results So Far
- ✅ Single worker tests: Pass
- ✅ Multi-worker (4 exec, 2 check): Pass (0.26s)
- ❌ Full equivalence suite: Race in DetectLivelock

## Commands to Run Tests

```bash
# Single test
go test ./model -run TestMultiThreadWorkerCounts/4_exec,_2_check -v

# Full equivalence suite
go test ./model -run TestMultiThreadEquivalence -v -timeout 2m

# With race detector
go test ./model -run TestMultiThreadEquivalence -v -race

# All multi-thread tests
go test ./model -run TestMultiThread -v
```

## Notes

- Main branch is clean: `git status` shows no uncommitted changes
- All single-threaded tests still pass
- CAS is now thread-safe at the implementation level
- Contention bottleneck successfully eliminated
