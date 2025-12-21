# Fixed: Await/Until Deadlock Detection

## Problem Summary

The ch5 test suite (ch5_await, ch5_deadlock, ch5_process_set) was failing with incorrect deadlock detection:

1. **ch5_await** - Falsely detected deadlock when threads were actually runnable
2. **ch5_deadlock** - Failed to detect actual deadlocks
3. **ch5_process_set** - Failed to detect actual deadlocks

## Root Causes

### 1. CONDITIONAL_YIELD Atomicity Issue

**Problem**: The CONDITIONAL_YIELD instruction was yielding even when the condition was satisfied, allowing other threads to interleave between the condition check and subsequent operations.

**Example Bug**:
```python
def reader():
    until(len(queue) != 0)  # Thread yields here even when TRUE
    current_msg = queue[0]  # Another thread could empty queue here
    queue = queue[1:]       # ERROR: Index out of bounds!
```

**Fix**: Changed CONDITIONAL_YIELD to return `ContinueStep` (not `YieldStep`) when condition is true:

```go
// interp/step.go:404
if condResult.AsBool() {
    frame.WaitCondition = nil
    return ContinueStep, 0, nil  // Don't yield - continue atomically
}
```

This ensures operations after `until()` execute atomically without interleaving.

### 2. PC Advancement Check Bug in evaluateWaitCondition

**Problem**: The original implementation checked if PC advanced to determine if a condition was satisfied. However, CONDITIONAL_YIELD **always** increments the PC (line 396-397 in interp/step.go), regardless of whether the condition is true or false.

**Original (Broken) Logic**:
```go
// WRONG: PC advances even when condition is false!
if newPC.Offset() > originalConditionPC.Offset() {
    return true, nil  // Incorrectly thinks condition is satisfied
}
```

**Why This Fails**:
```go
// CONDITIONAL_YIELD always does this:
newPC := frame.PC.Inc()  // PC incremented regardless of condition
frame.PC = newPC

if condResult.AsBool() {
    return ContinueStep  // Condition true
} else {
    return YieldStep     // Condition false, but PC already advanced!
}
```

### 3. Loop-Back Detection Bug

**Problem**: When a thread's condition is satisfied, it may execute, process data, loop back, and hit the **same** condition again (now false). The naive PC comparison couldn't distinguish between:

1. Condition false → still waiting at same PC (no progress)
2. Condition true → thread processed data, looped back to same PC (made progress)

**Example**:
```python
def reader():
    while True:
        until(len(queue) != 0)  # Condition at PC fn1:9
        msg = queue[0]
        queue = queue[1:]       # Process item
        # Loop back to same until() at PC fn1:9
```

When evaluateWaitCondition runs:
1. Rewinds to PC fn1:9
2. Condition is TRUE (queue has 1 item)
3. Thread continues, processes item, empties queue
4. Thread loops back to PC fn1:9
5. Condition is now FALSE (queue empty)
6. Thread ends at **same PC** as it started

Original code incorrectly concluded: "Same PC = condition false"

## The Complete Fix

### Solution: State-Change-Based Detection

The correct approach is to detect if the **state changed** during execution, not just check PC or pause reason:

```go
// model/run.go:evaluateWaitCondition

// Hash state BEFORE running
originalStateHash, err := casStore.Put(testState)

// Run thread until it pauses
_, err = interp.RunToPause(prog, testState, threadID)

// Hash state AFTER running
newStateHash, err := casStore.Put(testState)
stateChanged := originalStateHash != newStateHash

// Decision logic:
if thread is Waiting at SAME PC {
    if stateChanged {
        // Thread made progress, then looped back → condition was TRUE
        return true
    } else {
        // No progress made → condition is FALSE
        return false
    }
}
```

### Key Insights

1. **Atomicity**: CONDITIONAL_YIELD returns ContinueStep when true to prevent interleaving
2. **PC is unreliable**: PC always advances, can't be used to detect satisfaction
3. **State change is reliable**: If state changed, thread executed past the condition
4. **Same PC ≠ no progress**: Thread may process data and loop back to same condition

## Test Results

After the fix:

✅ **ch5_await** - Completes successfully (0 violations)
- Queue with 1 item, max_queue_size=1
- Reader condition `len(queue) != 0` correctly detected as TRUE
- Writer condition `len(queue) < 1` correctly detected as FALSE
- Only reader is runnable → not a deadlock

✅ **ch5_deadlock** - Correctly detects deadlock (1 violation)
- Both threads in `add_to_queue()` waiting for space
- Queue is full (14 items)
- Both conditions FALSE → 0 successors → deadlock

✅ **ch5_process_set** - Correctly detects deadlock (1 violation)
- Multiple reader threads and one writer
- All waiting with full queue → deadlock

## Files Modified

1. **interp/step.go** - CONDITIONAL_YIELD returns ContinueStep when true
2. **model/run.go** - evaluateWaitCondition uses state-change detection
3. **model/single_thread.go** - handleCyclicState uses BuildRunnable for deadlock detection

## Related Issues

This fix also resolved:
- Method dispatch system (queue.append() instead of functional append)
- PC rewinding for waiting threads in BuildRunnable
- Deadlock detection in cyclic states

## Lessons Learned

1. **Don't rely on PC advancement**: PC changes are implementation details, not semantic indicators
2. **State equality is semantic**: Comparing state hashes correctly captures "did anything happen?"
3. **Atomicity matters**: Operations after conditionals must execute without interleaving
4. **Test with loops**: Simple one-shot tests miss loop-back edge cases
