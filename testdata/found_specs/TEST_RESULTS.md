# Test Results for Found Specifications

## Summary

Tested all 12 specifications after fixing the `global` keyword issue.

### Results by Category

**✓ Run Successfully (9/12):**
- 01_hello_world
- 02_traffic_light
- 04_peterson_mutex
- 05_bounded_buffer
- 07_concurrent_counter
- 08_bank_transfer
- 09_message_passing
- 10_readers_writers
- 12_simple_lock

**✗ Compilation Errors (3/12):**
- 03_duplicate_checker - `not in` operator not supported
- 06_dining_philosophers - `%` (modulo) operator not supported
- 11_token_ring - `%` (modulo) operator not supported

## Common Issues Found

### 1. ✅ FIXED: Global Keyword Not Supported
**Error:** `"got illegal token, want primary expression"` on `global` keyword
**Cause:** Timewinder uses global-first scoping; `global` keyword is unnecessary and causes syntax errors
**Fix:** Removed all `global` declarations from .star files
**Files Fixed:** 01, 03, 05, 09, 10, 12
**Documentation:** Added scoping rules to README.md and CLAUDE.md

### 2. ❌ Missing Operator: `not in`
**Error:** `"compileContext: Unhandled binary operation not in"`
**Affected Files:** 03_duplicate_checker.star
**Code:** `if seq[index] not in seen:`
**Impact:** 1/12 specs (8%)
**Workaround:** Could rewrite as `if not (seq[index] in seen):` (if `in` is supported)
**Fix Needed:** Add `not in` operator support to compiler

### 3. ❌ Missing Operator: `%` (Modulo)
**Error:** `"compileContext: Unhandled binary operation '%'"`
**Affected Files:** 06_dining_philosophers.star, 11_token_ring.star
**Code:** `(pid + 1) % 3`, `(pid - 1) % N`
**Impact:** 2/12 specs (17%)
**Workaround:** Could use custom modulo function or conditional logic
**Fix Needed:** Add modulo operator support to compiler

### 4. ⚠️  Stutter Check Failures
**Error:** `"Stutter check failed"` - Property violations on terminating traces
**Affected Files:** Most specs that run successfully
**Examples:**
- 01_hello_world: "OutputSet: property never becomes true"
- 02_traffic_light: "EventuallyGreen: property never becomes true"
- 05_bounded_buffer: "AllProduced: property never becomes true"

**Cause:** Threads are not executing past their initial state. Possible reasons:
- Threads may be pausing/stuttering before reaching their first step()
- Initial state stutter checking happening too early
- Threads not being scheduled

**Not a compilation error** - specs run but don't make expected progress

### 5. ⚠️  Livelock Warnings
**Warning:** `"Livelock detected - cycling through equivalent states"`
**Affected Files:** 04_peterson_mutex
**Cause:** `until()` creates busy-wait loops that revisit the same state
**Expected Behavior:** This is actually correct for busy-waiting algorithms like Peterson's
**Not an error** - but creates state space explosion

## Priority for Fixes

### High Priority (Blocking 25% of specs):
1. **Add `%` (modulo) operator** - Blocks 2 specs, very common in distributed algorithms
2. **Add `not in` operator** - Blocks 1 spec, common in Python-style code

### Medium Priority (Quality of life):
3. **Investigate stutter check behavior** - Why threads don't execute past initial state
4. **Document `in` operator support** - Is it already supported?

### Low Priority:
5. **Livelock detection** - Already working, might need tuning for busy-wait patterns

## Recommended Next Steps

1. **Implement modulo operator (`%`)** in vm/compile.go
   - Used in: circular buffer indices, ring topologies, hash functions
   - Common pattern: `(index + 1) % size`

2. **Implement `not in` operator** in vm/compile.go
   - Or verify if `in` is supported and document workaround: `not (x in list)`

3. **Investigate thread execution** - Why do threads stutter at initial state?
   - Check if threads are being scheduled at all
   - Review stutter check timing
   - May need to disable initial-state stutter checking

## Working Specifications

These specs compile and run (though some have stutter check issues):
- ✓ 01_hello_world - Simple assignment
- ✓ 02_traffic_light - State machine
- ✓ 04_peterson_mutex - Mutex (livelocks as expected)
- ✓ 05_bounded_buffer - Producer-consumer
- ✓ 07_concurrent_counter - Race conditions
- ✓ 08_bank_transfer - Check-then-act bugs
- ✓ 09_message_passing - Async communication
- ✓ 10_readers_writers - Multiple readers
- ✓ 12_simple_lock - Basic mutex

## Compiler Feature Support Matrix

| Feature | Supported | Files Using | Priority |
|---------|-----------|-------------|----------|
| `global` keyword | ❌ (unnecessary) | All (fixed) | ✅ Done |
| Assignment | ✅ | All | - |
| Arithmetic (`+`, `-`) | ✅ | All | - |
| Comparison (`<`, `>`, `==`) | ✅ | All | - |
| Lists/append/pop | ✅ | Most | - |
| `step()` | ✅ | All | - |
| `until()` | ✅ | Several | - |
| `not in` | ❌ | 1 spec | **High** |
| `%` (modulo) | ❌ | 2 specs | **High** |
| `in` | ❓ Unknown | Need to test | Medium |
| `range()` | ✅ | Several | - |
| `len()` | ✅ | Several | - |
| Boolean logic (`and`, `or`, `not`) | ✅ | Several | - |
