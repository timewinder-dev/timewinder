# Timewinder - Project Documentation for AI Assistants

## Project Overview

**Timewinder** is a temporal logic model checker for programs written in Starlark (a Python-like scripting language). It's inspired by TLA+ and aims to verify behavioral properties of concurrent/multi-threaded programs by systematically exploring different execution interleavings.

### What Problem Does This Solve?

In concurrent systems, bugs can be extremely difficult to find because they depend on specific thread interleavings. Timewinder allows you to:
- Write programs with concurrent behavior
- Specify temporal logic properties (e.g., "bank accounts never go negative")
- Systematically explore all possible execution orderings
- Verify that properties hold across all possible executions

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────┐
│                    CLI (cmd/timewinder/)                │
│                   - run command                         │
│                   - version command                     │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│              Model Checker (model/)                     │
│  - Executor: Orchestrates model checking                │
│  - SingleThreadEngine: BFS state space exploration      │
│  - Spec: Parses TOML specifications                     │
│  - Evaluator: Evaluates temporal properties             │
└────────────┬───────────────────┬───────────────────────┘
             │                   │
    ┌────────▼────────┐  ┌──────▼──────────────┐
    │  VM (vm/)       │  │  Interpreter       │
    │  - Compiler     │◄─┤  (interp/)         │
    │  - Bytecode     │  │  - Step executor   │
    │  - Program      │  │  - State mgmt      │
    │  - 43 opcodes   │  │  - Call handling   │
    └─────────────────┘  └────────────────────┘
```

### Data Flow

1. **Spec File (.toml)** defines:
   - Program file path (.star)
   - Thread entry points
   - Temporal properties to check

2. **Compilation Pipeline**:
   ```
   .star file → Starlark Parser → VM Bytecode → Program
   ```

3. **Execution Pipeline**:
   ```
   Executor → SingleThreadEngine → Thunk Queue
        ↓
   For each Thunk:
     - Select thread to run
     - Execute until yield/completion
     - Generate successor states
     - Check properties
     - Enqueue new Thunks
   ```

4. **State Management**:
   - Each State contains: globals, per-thread stacks, pause reasons
   - Thunks represent (state, thread_to_run, execution_trace) tuples
   - Canonicalization expands non-deterministic values

## Key Concepts

### The `step()` Function

The special `step(name)` function marks atomic boundaries in programs:
```python
def withdraw(amount):
    step("read_balance")
    balance = account["balance"]
    step("write_balance")
    account["balance"] = balance - amount
```

Between steps, other threads can interleave, allowing the model checker to explore race conditions.

### Scoping Rules (Global-First)

**IMPORTANT**: Timewinder uses **global-first scoping**, which differs from standard Python/Starlark:

- When assigning to a variable (`x = value`), Timewinder first checks if `x` exists in the global scope
- If it exists globally, the global variable is modified
- If not, a new local variable is created
- **No `global` keyword is needed or supported** - it will cause a syntax error
- This makes it easy to write concurrent programs that modify shared state

**Example:**
```python
counter = 0  # Global variable

def increment():
    counter = counter + 1  # Modifies the global counter (no 'global' needed!)

def process():
    local_var = 5  # Creates a local variable
    counter = counter + 1  # Still modifies the global counter
```

**Why this design?**
- Simplifies writing concurrent programs with shared state
- Matches the mental model of TLA+ and PlusCal where variables are inherently global
- Reduces boilerplate in specifications

**Caveat:**
- If you accidentally use the same name as a global variable, you'll modify the global instead of creating a local
- Use distinct names for local variables to avoid confusion

### Properties

Properties are expressions evaluated on each state:
```toml
[properties]
NoOverdraft = {Always = "balance >= 0"}
EventuallyPositive = {Eventually = "balance > 0"}
```

Supported temporal operators:
- `Always`: Property must hold in all states
- `Eventually`: Property must hold in at least one state
- `AlwaysEventually`: Property repeatedly becomes true
- `EventuallyAlways`: Property eventually becomes permanently true

## Current Implementation Status

### What Works ✓

1. **Starlark Compilation**:
   - Variables (local/global)
   - Arithmetic, comparison, logical operations
   - Control flow (if/elif/else, for loops)
   - Functions with parameters
   - Lists, dicts, basic data structures
   - `step()` function compilation

2. **VM/Bytecode**:
   - 43 opcodes implemented
   - Stack-based execution model
   - Function calls and returns
   - Iterator support (partial)

3. **Interpreter**:
   - Single-step bytecode execution
   - State serialization/cloning
   - Thread pause/resume
   - Call stack management

4. **Model Checking Infrastructure**:
   - Spec file parsing (TOML)
   - Executor initialization
   - State queue management
   - Basic BFS exploration framework

5. **CLI**:
   - `timewinder run <spec>` command
   - Version reporting
   - Debug printing

### What's Broken or Incomplete ✗

1. **CRITICAL - YIELD Opcode Handler** (interp/step.go):
   - YIELD is defined but not implemented in Step()
   - When `step()` is called, it generates a YIELD instruction
   - This instruction is not handled, likely causing panics
   - **Impact**: Core functionality is broken

2. **CRITICAL - BuildRunnable Bug** (model/run.go:38):
   ```go
   out = append(out)  // Missing argument!
   ```
   - Should be `out = append(out, something)`
   - Causes runnable successors to be empty
   - **Impact**: No state transitions are generated

3. **CRITICAL - CheckProperties is Stubbed** (model/run.go:45-47):
   ```go
   func CheckProperties(s *interp.State, props Properties) error {
       return nil  // TODO: Actually check properties
   }
   ```
   - Properties are never actually evaluated
   - **Impact**: No verification happens

4. **Iterator Implementation Incomplete**:
   - `SliceIterator` exists but has no methods
   - ITER_START, ITER_NEXT, ITER_END may not work properly
   - For loops might be broken

5. **State Deserialization** (interp/state.go):
   - Returns "unimplemented" error
   - Affects state persistence and caching

6. **No Cycle Detection**:
   - Model checker doesn't track visited states
   - Can run infinitely on loops
   - No CAS (content-addressable storage) integration

7. **Missing Counterexample Generation**:
   - When properties fail, no trace is reported
   - Hard to debug why a property doesn't hold

### Recent Development History

**Last Work (Sept 6, 2025)**:
- Commit 7987dfa: Major implementation work
  - Added 205 lines to interp/step.go (YIELD implementation?)
  - Added 82 lines to interp/types.go
  - Modified executor, evaluator, run.go
  - Created debug_test.star/toml (now deleted)
  - **Re-enabled RunModel() call in CLI**

**Earlier Work (May-Jul 2025)**:
- Refactored from bottom-up (VM first) to top-down (CLI first)
- Created cobra-based CLI
- Extracted run functionality
- Added evaluation stubs
- Implemented non-determinism support

**Foundational Work (Earlier)**:
- Implemented VM, bytecode, compiler
- Built interpreter
- Added function call support
- Created spec file format

## Dependencies

Key libraries:
- `go.starlark.net`: Starlark parser
- `github.com/spf13/cobra`: CLI framework
- `github.com/BurntSushi/toml`: TOML parsing
- `github.com/shamaton/msgpack/v2`: Serialization
- `github.com/dgryski/go-farm`: Hashing

## Testing

Test files in `testdata/`:
- `practical_tla/ch1/ch1_a.star` - Banking example from Practical TLA+ book
- `small/*.star` - Simple test cases

## Known Issues Summary

| Issue | File | Line | Severity |
|-------|------|------|----------|
| YIELD not handled | interp/step.go | - | CRITICAL |
| BuildRunnable bug | model/run.go | 38 | CRITICAL |
| CheckProperties stub | model/run.go | 45 | CRITICAL |
| Iterator incomplete | interp/types.go | - | HIGH |
| No cycle detection | model/single_thread.go | - | HIGH |
| State.Deserialize stub | interp/state.go | - | MEDIUM |

## Files to Understand

Priority order for understanding the codebase:

1. **testdata/practical_tla/ch1/ch1_a.star** - Example program
2. **testdata/practical_tla/ch1/ch1_a.toml** - Example spec
3. **cmd/timewinder/run.go** - Entry point
4. **model/executor.go** - Orchestration logic
5. **model/single_thread.go** - BFS exploration
6. **interp/step.go** - Bytecode interpreter
7. **vm/compile.go** - Compiler
8. **vm/opcodes.go** - Instruction set

## Development Notes

### The compile/ Directory

There's a large alternate compiler in `compile/` (~2000 lines). This appears to be legacy code. The active compiler is in `vm/compile.go`. Consider removing `compile/` once stable.

### State Space Explosion

Model checking suffers from state space explosion. Future optimizations:
- State hashing and deduplication
- Symmetry reduction
- Partial order reduction
- State compression

### Testing Strategy

Current testing is minimal. Need:
- Unit tests for each opcode
- Integration tests for common patterns
- Property violation tests
- Performance benchmarks

## Experimental Branches

Note: The repository currently has several experimental commits branching from b1665f1:
- `7987dfa` - Implements YIELD handling and re-enables RunModel()
- `66731e5`, `dbc0990`, `6806ef6` - Other experimental work
- Current HEAD: `b1665f1` (RunModel still commented out)
- Main branch: `f2d204e`

These branches contain partial fixes but aren't fully working yet.

---

# Suggestions for Next Work

## Priority 1: Get Basic Model Checking Working (Critical Path)

These three bugs must be fixed to get even the simplest example working:

### 1. Fix BuildRunnable Bug (CRITICAL)
**File**: `model/run.go:38`
**Issue**:
```go
out = append(out)  // Missing argument!
```
**Fix**: Should be `out = append(out, x)`

**Why Critical**: Without this fix, the model checker immediately exits after initialization because no successor states are ever generated. The queue stays empty.

**Effort**: 5 minutes

---

### 2. Implement YIELD Opcode Handler (CRITICAL)
**File**: `interp/step.go`
**Issue**: The YIELD opcode exists but has no case in the Step() function
**What to implement**:
```go
case vm.YIELD:
    return YieldStep, 0, nil
```

**Why Critical**: Every `step()` call in user programs generates a YIELD instruction. Without handling it, programs using `step()` will fail (and all interesting concurrent programs need `step()`).

**Effort**: 10 minutes

**Note**: Check if commit `7987dfa` has a working implementation you can reference.

---

### 3. Implement CheckProperties (CRITICAL)
**File**: `model/run.go:45-47`
**Current**:
```go
func CheckProperties(state *interp.State, props []Property) error {
    return nil
}
```

**What to implement**:
- For each property in `props`, call `property.Check(state)`
- If any property returns false, return an error with:
  - Which property failed
  - The current state (or trace)
  - Useful debugging info

**Why Critical**: This is the whole point of the tool - verifying properties! Without it, you can't detect violations.

**Effort**: 30 minutes

---

### 4. Re-enable RunModel() (TRIVIAL)
**File**: `cmd/timewinder/run.go:40-43`
**Issue**: RunModel() is commented out
**Fix**: See commit `7987dfa` for the uncommented version

**Effort**: 2 minutes

**Note**: You may be on an older commit. Consider whether to merge the experimental branches or start fresh.

---

## Priority 2: Complete Core Functionality

After the critical fixes, these are needed for real-world usage:

### 5. Implement Iterator Support (HIGH)
**Files**:
- `interp/step.go` - Handle ITER_START, ITER_NEXT, ITER_END
- `interp/types.go` - Implement SliceIterator methods

**Why Important**: The test case uses `for name in acc` which requires iterators. Many programs need loops.

**Effort**: 2-3 hours

**Test**: The `no_overdrafts()` function in ch1_a.star uses a for loop

---

### 6. Add State Cycle Detection (HIGH)
**File**: `model/single_thread.go`
**What to implement**:
- Hash each state (use dgryski/go-farm)
- Track seen states in a map
- Skip states we've already explored
- Detect infinite loops

**Why Important**: Without this, model checking can run forever on programs with loops.

**Effort**: 1-2 hours

---

### 7. Implement Temporal Logic Evaluation (MEDIUM)
**File**: `model/evaluator.go`
**What to implement**:
- `Always`: Track that property holds in every state
- `Eventually`: Track that property holds in at least one state
- `AlwaysEventually`: Property becomes true infinitely often
- `EventuallyAlways`: Property eventually stays true forever

**Why Important**: These are more sophisticated than simple invariants and unlock richer specifications.

**Effort**: 3-4 hours

**Current Status**: Only "Always" is really meaningful with current single-state checking. Others need trace-level evaluation.

---

### 8. Better Error Messages and Counterexamples (HIGH)
**What to implement**:
- When CheckProperties fails, print:
  - The execution trace (sequence of steps)
  - State at each step
  - Which thread ran at each step
  - Clear description of property violation

**Why Important**: Without good errors, debugging failing properties is nearly impossible.

**Effort**: 2-3 hours

---

## Priority 3: Robustness and Polish

### 9. Add Comprehensive Tests
**What to add**:
- Unit tests for each opcode
- Tests for compiler edge cases
- Integration tests for small programs
- Property violation tests
- Performance benchmarks

**Why Important**: Prevent regressions, document expected behavior

**Effort**: Ongoing

---

### 10. Implement State Serialization
**File**: `interp/state.go`
**Issue**: `Deserialize()` is unimplemented
**Why Useful**:
- Persist explored states to disk
- Resume interrupted model checking
- Analyze states offline

**Effort**: 2-3 hours

---

### 11. Clean Up Legacy Code
**What to do**:
- Evaluate if `compile/` directory is needed
- If not, delete it (it's ~2000 lines of duplicate code)
- Update imports if necessary

**Why**: Reduces confusion, improves maintainability

**Effort**: 30 minutes

---

### 12. Add More Examples
**What to add**:
- Mutex example
- Producer-consumer
- Leader election
- Distributed transactions

**Why**: Demonstrate capabilities, test edge cases

**Effort**: Ongoing

---

## Priority 4: Advanced Features

### 13. Partial Order Reduction
Reduce state space by recognizing that some thread interleavings are equivalent.

**Effort**: Weeks

---

### 14. Symmetry Reduction
Detect symmetric states (e.g., if thread A and B are identical) to reduce redundant exploration.

**Effort**: Weeks

---

### 15. State Compression
Compress states for storage efficiency.

**Effort**: Days

---

### 16. Parallel Model Checking
Use multiple cores to explore state space faster.

**Effort**: Weeks

---

## Immediate Recommended Action Plan

**Phase 1: Get it working** (1-2 hours)
1. Fix BuildRunnable bug (5 min)
2. Implement YIELD handler (10 min)
3. Re-enable RunModel() (2 min)
4. Implement basic CheckProperties (30 min)
5. Test on ch1_a.toml example

**Phase 2: Make it useful** (1 day)
1. Complete iterator support (2-3 hours)
2. Add cycle detection (1-2 hours)
3. Improve error messages (2-3 hours)

**Phase 3: Make it robust** (1 week)
1. Add comprehensive tests
2. Implement temporal logic operators
3. Add more examples
4. Write documentation

---

## Quick Start for Next Session

1. **Decide on branch strategy**: Do you want to merge the experimental commits (like 7987dfa) or fix bugs on current HEAD?

2. **Run the test case** to see current behavior:
   ```bash
   go build -o timewinder ./cmd/timewinder
   ./timewinder run testdata/practical_tla/ch1/ch1_a.toml
   ```

3. **Fix the three critical bugs** in order:
   - BuildRunnable append bug
   - YIELD opcode handler
   - CheckProperties implementation

4. **Test again** - you should see either:
   - "Model checking completed successfully" (if no violations)
   - Or a clear error message showing the property violation

5. **Celebrate** - you'll have a working temporal logic model checker!
