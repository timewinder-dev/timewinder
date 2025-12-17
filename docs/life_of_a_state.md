# Life of a State: Timewinder Execution Flow

This document traces the complete lifecycle of a state as it flows through Timewinder's model checker, from initialization through execution to successor generation.

## Overview: Two Levels of Execution

Timewinder operates at two distinct levels:

1. **Model Checker Level** (`model/`): Orchestrates state space exploration via breadth-first search (BFS)
2. **Interpreter Level** (`interp/`, `vm/`): Executes individual threads within a state until they pause

The model checker generates states to explore, while the interpreter runs each state's code.

---

## Part 1: Model Checker Level

### Architecture Components

- **Executor**: Orchestrates the entire model checking process
- **SingleThreadEngine**: Implements BFS exploration with depth tracking
- **Thunk**: Represents a work item: `(state, thread_to_run, execution_trace)`
- **Queue/NextQueue**: Two queues for current depth and next depth exploration

### Initialization Phase

```
1. User provides spec.toml
   ├─ Program file path (.star)
   ├─ Thread entry points
   └─ Properties to check

2. Executor.Initialize()
   ├─ Compile program (vm.CompileLiteral)
   ├─ initializeGlobal()
   │  ├─ Inject builtins (range, oneof) into StackFrame
   │  ├─ RunToEnd() on global code
   │  └─ Store as InitialState.Globals
   ├─ initializeProperties()
   │  └─ Compile property expressions
   └─ initEngine()
      └─ InitSingleThread()
         ├─ Canonicalize(InitialState)
         │  └─ Expands NonDetValues in global scope
         ├─ CheckProperties(each expanded state)
         └─ Create initial Thunks (one per thread per state)
```

**Key Insight**: If global code contains `amount = oneof(range(8))`, Canonicalize expands this into 8 initial states during initialization.

### Main Exploration Loop

```
SingleThreadEngine.RunModel() - BFS with explicit depth tracking:

depth = 0
Queue = [initial thunks]

LOOP:
  FOR each thunk in Queue:
    ┌─────────────────────────────────────┐
    │ 1. RunTrace(thunk)                  │
    │    ├─ Clone state                   │
    │    ├─ RunToPause(state, thread)     │
    │    └─ Returns (newState, choices)   │
    └─────────────────────────────────────┘

    IF choices != nil:  // NonDet from oneof()
      ┌─────────────────────────────────────┐
      │ IMMEDIATE EXPANSION                 │
      │ FOR each choice:                    │
      │   ├─ Clone thunk + state            │
      │   ├─ Push choice to stack           │
      │   └─ Add to Queue (same depth)      │
      └─────────────────────────────────────┘
      CONTINUE  // Skip BuildRunnable

    ┌─────────────────────────────────────┐
    │ 2. CheckProperties(newState)        │
    │    └─ Evaluate each property        │
    │       └─ ABORT if violation         │
    └─────────────────────────────────────┘

    ┌─────────────────────────────────────┐
    │ 3. BuildRunnable(thunk, newState)   │
    │    ├─ Hash state (via CAS)          │
    │    ├─ Check VisitedStates[hash]     │
    │    │  └─ If visited: return nil     │
    │    ├─ Mark as visited               │
    │    ├─ Canonicalize(newState)        │
    │    │  └─ Expands NonDetValues        │
    │    └─ FOR each runnable thread:     │
    │       └─ Create successor Thunk     │
    └─────────────────────────────────────┘

    Add successors to NextQueue

  IF NextQueue empty: DONE
  Queue = NextQueue
  NextQueue = nil
  depth++
```

### State Hashing and Cycle Detection

```
BuildRunnable(thunk, state):
  1. CAS.Put(state)
     ├─ Decomposes state into content-addressable components
     ├─ Serializes via msgpack
     └─ Returns Hash (via FarmHash)

  2. Check VisitedStates[hash]
     ├─ If true: return nil (prune cycle)
     └─ Else: mark visited, continue

  3. Generate successors for next depth
```

**Cycle Detection**: States are identified by cryptographic hash. If we encounter a state hash we've seen before, we prune that branch (no successors generated).

### Successor Generation

```
BuildRunnable creates successors for the NEXT depth:

FOR each thread in state:
  IF thread.PauseReason == Yield or Start:
    FOR each canonicalized state:
      Create Thunk:
        ├─ ToRun: thread index
        ├─ State: canonicalized state clone
        └─ Trace: append (ThreadRan, StateHash)
```

**Key Insight**: Successor thunks represent the next step of execution. They're enqueued for the next depth level, ensuring BFS ordering.

---

## Part 2: Interpreter Level

The interpreter runs a single thread within a state until it pauses (yield, finish, or non-determinism).

### Core Components

- **State**: Contains globals, per-thread stacks, pause reasons
- **StackFrame**: Execution frame with PC, stack, variables, iterator state
- **Program**: Bytecode with 43 opcodes
- **Step()**: Executes one bytecode instruction

### Execution Flow

```
RunToPause(program, state, threadIndex):

  LOOP:
    ┌────────────────────────────────────┐
    │ Step(program, globals, stacks[i])  │
    │                                    │
    │ Returns: (StepResult, n, error)    │
    │   ├─ ContinueStep: keep going      │
    │   ├─ CallStep: function call       │
    │   ├─ ReturnStep: function return   │
    │   ├─ EndStep: no more instructions │
    │   └─ YieldStep: yield to scheduler │
    └────────────────────────────────────┘

    SWITCH StepResult:

    CASE ContinueStep:
      continue  // Most common case

    CASE YieldStep:
      state.PauseReason[i] = Yield
      return (nil, nil)  // Pause execution

    CASE CallStep:
      ┌────────────────────────────────────┐
      │ BuildCallFrame(program, frame, n)  │
      │                                    │
      │ IF BuiltinValue:                   │
      │   ├─ Look up from BuiltinRegistry  │
      │   ├─ Pop arguments                 │
      │   ├─ Call implementation           │
      │   ├─ Push result (even NonDetVal)  │
      │   ├─ Increment PC                  │
      │   └─ Return nil (no new frame)     │
      │                                    │
      │ ELSE (FnPtrValue):                 │
      │   ├─ Create new StackFrame         │
      │   ├─ Bind arguments to parameters  │
      │   └─ Return new frame              │
      └────────────────────────────────────┘

      // Check for NonDetValue after builtin call
      IF f == nil AND top of stack is NonDetValue:
        ├─ Pop NonDetValue
        ├─ Set PauseReason[i] = NonDet
        └─ Return choices  // Signal to model checker

      IF f != nil:
        Append f to call stack

    CASE ReturnStep:
      IF only one frame (top-level return):
        PauseReason[i] = Finished
        return (nil, nil)
      ELSE:
        ├─ Pop call frame
        ├─ Push return value to caller
        └─ Increment caller's PC

    CASE EndStep:
      PauseReason[i] = Finished
      return (nil, nil)
```

### The Step Function: Bytecode Execution

```
Step(program, globals, stack):
  frame = stack[top]
  instruction = program.GetInstruction(frame.PC)

  SWITCH instruction.Code:

    PUSH: frame.Push(instruction.Arg.Clone())
    POP: frame.Pop()

    GETVAL:
      name = frame.Pop()
      value = resolveVar(name, program, globals, stack)
      frame.Push(value)

    SETVAL:
      name = frame.Pop()
      val = frame.Pop()
      StoreVar(name, val)  // In appropriate scope

    ADD/SUBTRACT/MULTIPLY/DIVIDE:
      b = frame.Pop()
      a = frame.Pop()
      result = numericOp(op, a, b)
      frame.Push(result)

    EQ/LT/LTE:
      b = frame.Pop()
      a = frame.Pop()
      result = compare(a, b)
      frame.Push(BoolValue(result))

    JMP:
      frame.PC = frame.PC.SetOffset(label)
      return ContinueStep

    JFALSE:
      cond = frame.Pop()
      IF !cond.AsBool():
        frame.PC = frame.PC.SetOffset(label)
        return ContinueStep

    CALL:
      return (CallStep, argCount, nil)

    RETURN:
      return (ReturnStep, 0, nil)

    YIELD:
      frame.Push(instruction.Arg)  // Step name
      frame.PC = frame.PC.Inc()
      return (YieldStep, 0, nil)

    BUILD_LIST:
      n = instruction.Arg
      elements = [frame.Pop() for _ in range(n)]
      frame.Push(ArrayValue(elements))

    BUILD_DICT:
      n = instruction.Arg
      pairs = [frame.Pop() for _ in range(n)]
      frame.Push(StructValue(dict(pairs)))

    // ... 43 opcodes total

  frame.PC = frame.PC.Inc()  // Default: increment PC
  return (ContinueStep, 0, nil)
```

### Call Stack Management

Function calls use a stack of StackFrames:

```
Before call to foo(x, y):
  Stack: [MainFrame]
  MainFrame.Stack: [ArgValue(x), ArgValue(y), FnPtrValue(foo)]

CALL 2 instruction:
  1. Step() returns (CallStep, 2, nil)
  2. RunToPause calls BuildCallFrame(prog, MainFrame, 2)
  3. BuildCallFrame:
     ├─ Pop FnPtrValue(foo)
     ├─ Pop 2 ArgValues
     ├─ Create FooFrame with PC at foo's entry point
     ├─ Bind arguments to parameters
     └─ Return FooFrame
  4. RunToPause appends FooFrame to stack

Inside foo:
  Stack: [MainFrame, FooFrame]
  Execution continues in FooFrame

RETURN instruction:
  1. Step() returns (ReturnStep, 0, nil)
  2. RunToPause:
     ├─ Pop FooFrame
     ├─ val = FooFrame.Pop()  // Return value
     ├─ MainFrame.Push(val)
     ├─ MainFrame.PC = MainFrame.PC.Inc()  // Move past CALL
     └─ Continue in MainFrame
```

### Builtin Function Execution

Builtins are special: they execute immediately without creating a new frame.

```
Example: oneof([1, 2, 3])

Bytecode:
  GETVAL "oneof"              // Push BuiltinValue{Name: "oneof"}
  BUILD_ARG                   // Wrap None as positional arg marker
  PUSH [1, 2, 3]              // Push array
  BUILD_ARG                   // Wrap as ArgValue
  CALL 1                      // Call with 1 argument

CALL 1 execution:
  1. Step() returns (CallStep, 1, nil)

  2. BuildCallFrame(prog, frame, 1):
     ├─ Pop BuiltinValue{Name: "oneof"}
     ├─ Look up BuiltinRegistry["oneof"]
     ├─ Pop ArgValue([1, 2, 3])
     ├─ Call builtinOneof([1, 2, 3])
     │  └─ Returns NonDetValue{Choices: [1, 2, 3]}
     ├─ Push NonDetValue to stack
     ├─ frame.PC = frame.PC.Inc()
     └─ Return nil (no new frame)

  3. RunToPause sees f == nil:
     ├─ Check top of stack
     ├─ Is NonDetValue? YES
     ├─ Pop NonDetValue
     ├─ state.PauseReason[thread] = NonDet
     └─ Return ([1, 2, 3], nil)
```

The model checker receives the choices array and immediately expands into 3 successor states.

### Variable Resolution

```
resolveVar(name, program, globals, stack):
  // Search local scopes (stack frames) first
  FOR frame in reverse(stack):
    IF name in frame.Variables:
      return frame.Variables[name]

  // Check global scope
  IF name in globals.Variables:
    return globals.Variables[name]

  // Check program symbols (function names)
  IF ptr = program.Resolve(name):
    return FnPtrValue(ptr)

  ERROR: No such variable
```

Variables are resolved using lexical scoping: local → global → program symbols.

---

## Part 3: Non-Determinism Flow

### Two Contexts for oneof()

#### Context 1: Global Initialization

```
Global code: amount = oneof(range(8))

1. Executor.initializeGlobal()
   ├─ Inject builtins into StackFrame
   ├─ RunToEnd(program, nil, frame)
   │  └─ Executes: amount = oneof(range(8))
   │     ├─ range(8) returns ArrayValue([0,1,2,3,4,5,6,7])
   │     ├─ oneof(...) returns NonDetValue{Choices: [0..7]}
   │     ├─ NonDetValue pushed to stack
   │     ├─ SETVAL pops it and assigns to "amount"
   │     └─ frame.Variables["amount"] = NonDetValue{...}
   └─ InitialState.Globals = frame

2. InitSingleThread()
   ├─ Canonicalize(InitialState)
   │  └─ Walks Variables, calls Expand() on each
   │     └─ NonDetValue.Expand() returns [0,1,2,3,4,5,6,7]
   │     → Produces 8 initial states (amount=0, amount=1, ...)
   │
   └─ FOR each expanded state:
      └─ Create initial Thunks (one per thread)
```

**Result**: 8 initial states enter the exploration queue at depth 0.

#### Context 2: During Thread Execution

```
Thread code:
  step("before")
  x = oneof([10, 20])
  step("after")

1. Model checker calls RunTrace(thunk)
   └─ RunToPause(prog, state, threadIndex)

2. Execute until oneof() call:
   ├─ step("before") → Yields, pauses
   ├─ Resume: GETVAL "oneof", PUSH [10,20], CALL 1
   └─ BuildCallFrame:
      ├─ Call builtinOneof([10, 20])
      ├─ Push NonDetValue{Choices: [10, 20]}
      └─ Return nil

3. RunToPause detects NonDetValue:
   ├─ Pop NonDetValue
   ├─ PauseReason = NonDet
   └─ Return (state, [10, 20], nil)

4. Model checker sees choices != nil:
   FOR each choice in [10, 20]:
     ├─ Clone thunk and state
     ├─ Push choice (10 or 20) to state.Stack
     ├─ Add to Queue (same depth)
     └─ Next iteration will continue at SETVAL

5. Two successor states:
   State A: x = 10, continues to step("after")
   State B: x = 20, continues to step("after")
```

**Key Difference**:
- Initialization: NonDetValue stored in variables, Canonicalize expands
- Execution: NonDetValue triggers immediate expansion, never stored

---

## Part 4: State Representation and Serialization

### State Structure

```go
type State struct {
    Globals      *StackFrame      // Global variables and functions
    Stacks       [][]*StackFrame  // Per-thread call stacks
    PauseReason  []Pause          // Why each thread paused
}

type StackFrame struct {
    Stack         []vm.Value           // Operand stack
    PC            vm.ExecPtr           // Program counter
    Variables     map[string]vm.Value  // Local variables
    IteratorStack []*IteratorState     // Iterator state
}

type Pause int
const (
    Start    Pause = iota  // Thread not yet started
    Finished               // Thread completed
    Yield                  // Thread called step()
    NonDet                 // Non-deterministic choice
)
```

### Content-Addressable Storage (CAS)

States are hashed for cycle detection:

```
CAS.Put(state):
  1. Decompose state into components:
     ├─ DecomposeStackFrame(Globals)
     │  ├─ Variables map → sorted keys, hash each value
     │  ├─ Stack → hash each value
     │  └─ PC, IteratorStack → serialize directly
     │
     ├─ FOR each thread's call stack:
     │  └─ DecomposeStackFrame(each frame)
     │
     └─ PauseReason array → serialize directly

  2. Serialize each component with msgpack

  3. Compute FarmHash of serialized data
     └─ Returns Hash (uint64)
```

**Value Decomposition**:
- Simple values (Int, Bool, Str, FnPtr, Builtin): Store directly
- Complex values (Array, Struct, NonDet): Store as Refs with recursive hashes
- BuiltinValue: Stores only name string (looked up from registry on recompose)

---

## Part 5: Example Trace

Let's trace `amount = oneof(range(3))` with a single thread:

### Initial Setup

```
Spec:
  [threads.main]
  entrypoint = "transfer()"

Program:
  amount = oneof(range(3))

  def transfer():
      step("withdraw")
      balance -= amount
      step("deposit")
```

### Execution Trace

```
═══════════════════════════════════════════════════════════
INITIALIZATION
═══════════════════════════════════════════════════════════

1. Executor.initializeGlobal()
   ├─ Inject: Globals["range"] = BuiltinValue{Name: "range"}
   ├─ Inject: Globals["oneof"] = BuiltinValue{Name: "oneof"}
   └─ RunToEnd on global code:

      Bytecode:
        GETVAL "range"     → Push BuiltinValue{Name: "range"}
        PUSH 3             → Push IntValue(3)
        BUILD_ARG          → Wrap as ArgValue
        CALL 1             → Call range(3)
          └─ Returns ArrayValue([0, 1, 2])
          └─ Push to stack

        GETVAL "oneof"     → Push BuiltinValue{Name: "oneof"}
        SWAP               → Swap array and builtin
        BUILD_ARG          → Wrap array as ArgValue
        CALL 1             → Call oneof([0,1,2])
          └─ Returns NonDetValue{Choices: [0,1,2]}
          └─ Push to stack

        PUSH "amount"      → Push StrValue("amount")
        SWAP
        SETVAL             → Globals["amount"] = NonDetValue{...}

2. InitSingleThread()
   ├─ Canonicalize(InitialState):
   │  └─ Globals["amount"].Expand() → [0, 1, 2]
   │  → Produces 3 states:
   │     State₀: amount=0
   │     State₁: amount=1
   │     State₂: amount=2
   │
   ├─ CheckProperties(each state)
   │
   └─ Create initial Thunks:
      Thunk₀: (State₀, thread=0, trace=[])
      Thunk₁: (State₁, thread=0, trace=[])
      Thunk₂: (State₂, thread=0, trace=[])

Queue = [Thunk₀, Thunk₁, Thunk₂]
depth = 0

═══════════════════════════════════════════════════════════
DEPTH 0: EXPLORE 3 STATES
═══════════════════════════════════════════════════════════

─────────────────────────────────────────
Process Thunk₀ (amount=0)
─────────────────────────────────────────

1. RunTrace(Thunk₀):
   └─ RunToPause(State₀, thread=0)

      Thread entry point: transfer()

      Bytecode of transfer():
        GETVAL "transfer"  → Push FnPtrValue(transfer)
        CALL 0             → Call transfer()
          └─ Create new StackFrame at transfer's PC

        Inside transfer:
          YIELD "withdraw" → Push StrValue("withdraw")
                          → PauseReason[0] = Yield
                          → PAUSE AND RETURN

   Returns: (State₀', nil, nil)  // No choices, normal yield

2. CheckProperties(State₀')
   ✓ Properties satisfied

3. BuildRunnable(Thunk₀, State₀'):
   ├─ Hash state: h₀ = CAS.Put(State₀')
   ├─ VisitedStates[h₀] = true
   ├─ Canonicalize(State₀') → [State₀']  // No NonDet values
   ├─ Thread 0 pause reason = Yield (runnable)
   └─ Create successor:
      Thunk₀₁: (State₀', thread=0, trace=[(0, h₀)])

NextQueue = [Thunk₀₁]

─────────────────────────────────────────
Process Thunk₁ (amount=1)
─────────────────────────────────────────

[Same as Thunk₀, creates Thunk₁₁ with amount=1]

NextQueue = [Thunk₀₁, Thunk₁₁]

─────────────────────────────────────────
Process Thunk₂ (amount=2)
─────────────────────────────────────────

[Same as Thunk₀, creates Thunk₂₁ with amount=2]

NextQueue = [Thunk₀₁, Thunk₁₁, Thunk₂₁]

═══════════════════════════════════════════════════════════
DEPTH 1: EXPLORE 3 STATES
═══════════════════════════════════════════════════════════

Queue = [Thunk₀₁, Thunk₁₁, Thunk₂₁]

─────────────────────────────────────────
Process Thunk₀₁ (amount=0, after withdraw)
─────────────────────────────────────────

1. RunTrace(Thunk₀₁):
   └─ RunToPause(State₀', thread=0)

      Resume inside transfer():
        GETVAL "balance"   → Push current balance
        GETVAL "amount"    → Push IntValue(0)
        SUBTRACT           → balance - 0
        PUSH "balance"
        SWAP
        SETVAL             → Update balance

        YIELD "deposit"    → Push StrValue("deposit")
                          → PauseReason[0] = Yield
                          → PAUSE AND RETURN

   Returns: (State₀'', nil, nil)

2. CheckProperties(State₀'')
   ✓ balance still valid (amount=0, no change)

3. BuildRunnable(Thunk₀₁, State₀''):
   ├─ Hash: h₀₁ = CAS.Put(State₀'')
   └─ Create successor: Thunk₀₂

[Similar for Thunk₁₁ and Thunk₂₁]

═══════════════════════════════════════════════════════════
DEPTH 2: EXPLORE 3 STATES (FINAL STEP)
═══════════════════════════════════════════════════════════

[Each thunk finishes transfer(), PauseReason = Finished]
[No more successors generated]

NextQueue = []
DONE

Statistics:
  Total state transitions: 9 (3 states × 3 steps)
  Unique states: 9
  Maximum depth: 2
```

---

## Summary: Key Insights

1. **Two-Level Architecture**: Model checker orchestrates, interpreter executes
2. **BFS with Explicit Depth**: Queue/NextQueue pattern ensures breadth-first ordering
3. **Immediate Expansion**: oneof() during execution expands at the same depth
4. **Canonicalize for Initialization**: oneof() in globals expands during Canonicalize
5. **Cycle Detection**: Content-addressable hashing prevents infinite loops
6. **Unified Call Handling**: PC increment happens in BuildCallFrame for both builtins and regular functions
7. **Builtins are Special**: Execute immediately without creating frames, but can trigger state expansion

The power of this architecture is that the model checker systematically explores all possible executions, while the interpreter provides familiar imperative semantics for each individual execution.
