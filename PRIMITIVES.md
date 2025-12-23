# Timewinder Synchronization Primitives

## Overview

Timewinder provides several primitives for controlling thread execution and synchronization. These primitives allow you to model concurrent algorithms with precise control over interleaving.

## Basic Primitives

### `step(label)`
Yields control to allow other threads to execute. Always yields regardless of fairness.

```python
step("critical_section")
# Other threads can run here
```

### `wait(condition)` (formerly `until`)
Waits for a condition to become true. If the condition is false, the thread yields and will retry when resumed. **Checks condition once** - does not loop.

```python
wait(flag == True)  # Waits until flag becomes True
```

**Renamed from:** `until()` (deprecated but still supported for backward compatibility)

## Fairness Variants

Each primitive has fairness variants that affect scheduling guarantees:

| Primitive | Normal | Weak Fair | Strong Fair |
|-----------|---------|-----------|-------------|
| Step | `step()` | `fstep()` | `sfstep()` |
| Wait | `wait()` | `fwait()` | `sfwait()` |
| Step Until | `step_until()` | `fstep_until()` | `sfstep_until()` |

- **Normal**: No fairness guarantees
- **Weak Fair**: If continuously enabled, must eventually execute
- **Strong Fair**: If repeatedly enabled, must eventually execute

## New: Loop-Based Primitives

### `step_until(label, condition)`
**NEW**: Combines `while` loop + `step()` into a single primitive. Keeps yielding while condition is true.

```python
# Old style - manual loop
while flag[other]:
    step("wait_loop")

# New style - step_until primitive
step_until("wait_loop", flag[other])
```

**Compilation:**
```
loop_start:
  <evaluate condition>
  JFALSE loop_end
  PUSH label
  YIELD label
  JMP loop_start
loop_end:
  PUSH None
```

**Use case:** When you need to continuously check a condition that can change, such as waiting for another thread to complete an action.

## Comparison: `wait()` vs `step_until()`

### `wait(condition)` - Single Check
- Evaluates condition **once**
- If false, yields and marks thread as waiting
- Thread becomes runnable when condition becomes true
- Does NOT loop - checks only when thread is scheduled

```python
flag = False
wait(flag)  # Waits once, wakes up when flag becomes True
```

### `step_until(label, condition)` - Continuous Loop
- **Loops** while condition is true
- Yields on each iteration with the given label
- Keeps re-checking condition
- Exits loop when condition becomes false

```python
flag = True
step_until("busy_wait", flag)  # Loops until flag becomes False
```

## Real-World Examples

### Dekker's Algorithm - Using `step_until()`
```python
def process(pid):
    flag[pid] = True

    # Keep checking while other wants to enter
    while flag[other]:
        if turn == other:
            flag[pid] = False
            wait(turn == pid)  # Wait once for our turn
            flag[pid] = True
        step("recheck")  # Manual step in while loop

    # Critical section
```

Can be simplified to:
```python
def process(pid):
    flag[pid] = True

    # Keep checking while other wants to enter
    while flag[other]:
        if turn == other:
            flag[pid] = False
            wait(turn == pid)
            flag[pid] = True
        step("recheck")

    # Note: step_until() works when you're checking a single condition
    # For complex backoff logic like Dekker's, manual while+step is clearer
```

### Simple Producer-Consumer - Using `wait()`
```python
def producer():
    item = produce_item()
    wait(len(buffer) < MAX_SIZE)  # Wait for space
    buffer.append(item)

def consumer():
    wait(len(buffer) > 0)  # Wait for item
    item = buffer.pop(0)
    consume(item)
```

### Bakery Algorithm - Using `step_until()`
```python
for j in range(N):
    if j == pid:
        continue

    # Keep checking while j is choosing
    step_until("wait_choosing", choosing[j])

    # Keep checking while j has priority
    step_until("wait_priority",
               number[j] != 0 and (number[j] < number[pid] or
                                   (number[j] == number[pid] and j < pid)))
```

## Deprecated Aliases

For backward compatibility, the old names still work:
- `until()` → `wait()`
- `funtil()` → `fwait()`
- `sfuntil()` → `sfwait()`

## Summary

- **`step(label)`**: Always yields
- **`wait(condition)`**: Yield once until condition is true (renamed from `until`)
- **`step_until(label, condition)`**: **NEW** - Loop and yield while condition is true

Choose based on your needs:
- Use `wait()` when you're waiting for a one-time event
- Use `step_until()` when you need to continuously monitor a changing condition
- Use manual `while` + `step()` when you have complex logic inside the loop
