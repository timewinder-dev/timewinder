![Timewinder Logo](docs/logo-1-textright.png)

**A temporal logic model checker for concurrent programs written in Starlark (Python-like syntax)**

Timewinder helps you find bugs in concurrent systems by systematically exploring all possible execution orderings. Think of it as [TLA+](https://lamport.azurewebsites.net/tla/tla.html) or [PlusCal](https://lamport.azurewebsites.net/tla/pluscal.html), but with Python's familiar syntax instead of mathematical notation.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)

---

## Why Timewinder?

Concurrent systems are notoriously difficult to test. Race conditions, deadlocks, and other concurrency bugs often only appear under specific thread interleavings that are nearly impossible to reproduce in traditional testing.

**Example:** Two bank transfers executing simultaneously:
```python
# Thread 1: Transfer $5 from Alice to Bob
balance = accounts["alice"]  # Reads $10
accounts["alice"] = balance - 5  # Writes $5

# Thread 2: Transfer $8 from Alice to Charlie
balance = accounts["alice"]  # Also reads $10!
accounts["alice"] = balance - 8  # Writes $2, losing Thread 1's update
```

With normal testing, this race condition might never appear. With Timewinder, you **systematically explore all possible interleavings** and verify that your invariants (like "accounts never go negative") hold in **every possible execution**.

---

## What is Model Checking?

[Model checking](https://en.wikipedia.org/wiki/Model_checking) is a formal verification technique that exhaustively explores all possible states of a system. Instead of running your program once and hoping for the best, model checking:

1. **Explores all possible execution paths** by trying different thread interleavings
2. **Checks your properties** (invariants, safety, liveness) at each state
3. **Produces counterexamples** when properties are violated, showing you exactly how to reproduce the bug

Timewinder brings model checking to everyday programming with:
- **Python-like syntax** via [Starlark](https://github.com/google/starlark-go) (Google's deterministic Python dialect)
- **Automatic state exploration** with BFS (breadth-first search)
- **Temporal logic properties** to express safety and liveness requirements
- **Fairness assumptions** to avoid false alarms from unrealistic schedules

---

## How It Compares

| Feature | Timewinder | TLA+ | PlusCal | Traditional Testing |
|---------|------------|------|---------|-------------------|
| **Syntax** | Python-like (Starlark) | Mathematical | Algorithmic | Language-specific |
| **Learning Curve** | Low (if you know Python) | High | Medium | Low |
| **State Exploration** | ✅ Exhaustive | ✅ Exhaustive | ✅ Exhaustive (via TLA+) | ❌ Single execution |
| **Finds Race Conditions** | ✅ Yes | ✅ Yes | ✅ Yes | 🎲 Maybe |
| **Counterexamples** | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No (just failure) |
| **IDE Support** | Good (Python) | TLA+ Toolbox | TLA+ Toolbox | Excellent |

**When to use Timewinder:**
- You know Python and want to verify concurrent algorithms
- You're designing a new distributed protocol or concurrent data structure
- You want to understand TLA+/PlusCal concepts with familiar syntax
- You need to find rare concurrency bugs before they hit production

**When to use TLA+:**
- You need industry-standard formal verification
- You're working on critical systems (aerospace, medical, financial)
- You want to verify complex temporal properties with TLAPS (TLA+ proof system)

---

## Getting Started

### Installation

**Prerequisites:**
- [Go 1.21+](https://go.dev/doc/install)

**Install from source:**
```bash
git clone https://github.com/timewinder-dev/timewinder.git
cd timewinder
go install ./cmd/timewinder
```

**Verify installation:**
```bash
timewinder version
```

### Your First Model

Let's verify a simple bank transfer system with two concurrent threads.

**1. Create a model file (`bank.star`):**

```python
# Global state
accounts = {"alice": 10, "bob": 10}

# Property: accounts should never go negative
def no_overdraft():
    for name, balance in accounts:
        if balance < 0:
            return False
    return True

# Transfer money from alice to bob
def transfer():
    step("read_balance")
    balance = accounts["alice"]

    step("withdraw")
    accounts["alice"] = balance - 5

    step("deposit")
    accounts["bob"] += 5
```

**2. Create a spec file (`bank.toml`):**

```toml
[spec]
program = "bank.star"

# Run two transfers concurrently
[threads.transfer1]
entrypoint = "transfer()"

[threads.transfer2]
entrypoint = "transfer()"

# Check that accounts never go negative
[properties]
NoOverdraft = {Always = "no_overdraft()"}
```

**3. Run the model checker:**

```bash
timewinder run bank.toml
```

**Output:**
```
✗ Property violation found!
Property: NoOverdraft
State #42: {"alice": 5, "bob": 20}

Trace:
  1. Thread transfer1: read_balance (alice=10)
  2. Thread transfer2: read_balance (alice=10)
  3. Thread transfer1: withdraw (alice=5)
  4. Thread transfer2: withdraw (alice=0)  ← Lost update!
```

Timewinder found the race condition! Both threads read the balance before either writes, causing one update to be lost.

**4. Fix the bug with atomic operations:**

```python
def transfer():
    step("atomic_transfer")  # Single atomic step
    accounts["alice"] -= 5
    accounts["bob"] += 5
```

Now Timewinder reports: `✓ All properties satisfied`

---

## Core Concepts

### The `step()` Function

The `step()` function marks **atomic boundaries** in your program. Between steps, other threads can execute, allowing the model checker to explore different interleavings.

```python
def withdraw(amount):
    step("read_balance")
    balance = account["balance"]

    step("check_sufficient")
    if balance < amount:
        return False

    step("write_balance")
    account["balance"] = balance - amount
    return True
```

**Key insight:** Timewinder explores what happens when other threads run between your steps. This is where race conditions hide!

### Global-First Scoping

Unlike Python, Timewinder uses **global-first scoping** to make concurrent code simpler:

```python
counter = 0  # Global variable

def increment():
    counter = counter + 1  # Modifies global (no 'global' keyword needed!)
```

**Why?** In concurrent systems, threads naturally share state. This matches TLA+/PlusCal semantics where variables are inherently shared.

**Caveat:** If you want a local variable, use a name that doesn't conflict with globals.

### Properties: What Can Go Wrong?

Properties are assertions about your system that should always (or eventually) be true:

```toml
[properties]
# Safety: bad things never happen
NoOverdraft = {Always = "balance >= 0"}
MutualExclusion = {Always = "critical_section_count <= 1"}

# Liveness: good things eventually happen
EventuallyDone = {Eventually = "finished"}
NoStarvation = {AlwaysEventually = "thread_enters_critical_section()"}
```

**Temporal operators:**
- `Always`: Property holds in **every state** (safety)
- `Eventually`: Property holds in **at least one state** (reachability)
- `EventuallyAlways`: Property **eventually becomes permanently true** (stability)
- `AlwaysEventually`: Property **repeatedly becomes true** (fairness/liveness)

### Fairness: Avoiding False Alarms

Without fairness assumptions, the model checker might find "bugs" where a thread never runs:

```python
def worker():
    while True:
        step("work")
        do_work()
```

**Weak fairness** (`fair=true`): If a thread stays continuously enabled, it **must eventually run**
```toml
[threads.worker]
entrypoint = "worker()"
fair = true  # Won't starve if always runnable
```

**Strong fairness** (`strong_fair=true`): If a thread is **repeatedly enabled**, it must eventually run
```toml
[threads.waiter]
entrypoint = "wait_for_event()"
strong_fair = true  # Won't starve even if event toggles
```

---

## Special Functions Reference

Timewinder provides special functions to control thread scheduling and fairness:

### Basic Step Functions

| Function | Description | Fairness | Use Case |
|----------|-------------|----------|----------|
| `step(label)` | Yield to other threads | None | Default atomic boundary |
| `fstep(label)` | Yield with weak fairness | Weak | Must run if continuously enabled |
| `sfstep(label)` | Yield with strong fairness | Strong | Must run if repeatedly enabled |

**Example:**
```python
step("acquire_lock")    # Other threads can run here
lock.held = True
```

### Wait Functions

| Function | Description | Fairness | Use Case |
|----------|-------------|----------|----------|
| `wait(condition)` | Wait until condition is true | None | Basic blocking |
| `fwait(condition)` | Wait with weak fairness | Weak | Must wake when condition stays true |
| `sfwait(condition)` | Wait with strong fairness | Strong | Must wake when condition toggles |

**Example:**
```python
# Wait for buffer to be non-empty
fwait(len(buffer) > 0)  # Fair: won't starve if buffer stays full
item = buffer.pop(0)
```

**Deprecated aliases:** `until()`, `funtil()`, `sfuntil()` still work but prefer `wait()`, `fwait()`, `sfwait()`

### Loop-Step Combinations

| Function | Description | Equivalent To |
|----------|-------------|---------------|
| `step_until(label, cond)` | Step repeatedly while condition is true | `while cond: step(label)` |
| `fstep_until(label, cond)` | Fair version | `while cond: fstep(label)` |
| `sfstep_until(label, cond)` | Strongly fair version | `while cond: sfstep(label)` |

**Example:**
```python
# Process all items in queue
step_until("process", len(queue) > 0)
```

---

## Spec File Format

Specs are written in [TOML](https://toml.io/) and define:
- Which program to check
- What threads to run
- What properties to verify
- Fairness assumptions

**Complete example:**

```toml
[spec]
program = "dining_philosophers.star"
termination = true  # Check that all threads eventually finish (optional)

# Run 5 philosopher threads
[threads.phil0]
entrypoint = "philosopher(0)"
strong_fair = true

[threads.phil1]
entrypoint = "philosopher(1)"
strong_fair = true

# ... phil2, phil3, phil4 ...

# Safety: No two philosophers hold the same fork
[properties]
MutualExclusion = {Always = "check_mutex()"}
NoStarvation = {AlwaysEventually = "eating[0]"}  # Phil 0 eats infinitely often
```

**Options:**
- `program`: Path to `.star` file (required)
- `termination`: Check that all threads finish (default: false)
- `max_depth`: Limit exploration depth (optional)

**Thread options:**
- `entrypoint`: Function to call (e.g., `"main()"`, `"worker(5)"`)
- `fair`: Enable weak fairness (default: false)
- `strong_fair`: Enable strong fairness (default: false)

---

## Language Features

Timewinder uses [Starlark](https://github.com/google/starlark-go), a deterministic Python dialect. Supported features:

**✅ Supported:**
- Variables, arithmetic, comparisons
- `if`/`elif`/`else`, `for`, `while`
- Functions with parameters and returns
- Lists: `[1, 2, 3]`, `list.append()`, `list.pop()`
- Dicts: `{"key": "value"}`, `dict["key"]`, `dict.items()`
- Tuples: `(1, 2, 3)`
- List comprehensions: `[x*2 for x in range(10)]`
- `oneof()`: Non-deterministic choice for model checking

**❌ Not Supported:**
- I/O (file operations, network)
- Modules/imports (use single-file models)
- Classes (use dicts and functions)
- `print()` (use properties to observe state)
- Random numbers (use `oneof()` for non-determinism)

**Global-first scoping:** Assignments to globals don't need `global` keyword (see above).

---

## Examples

The `testdata/` directory contains many examples:

**Basic concurrency:**
- [`practical_tla/ch1/`](testdata/practical_tla/ch1/) - Bank transfers, wire transfers ([Practical TLA+ book](https://link.springer.com/book/10.1007/978-1-4842-3829-5))
- [`found_specs/04_peterson_mutex.toml`](testdata/found_specs/04_peterson_mutex.toml) - Peterson's mutual exclusion algorithm
- [`found_specs/05_bounded_buffer.toml`](testdata/found_specs/05_bounded_buffer.toml) - Producer-consumer with bounded buffer

**Classic problems:**
- [`found_specs/06_dining_philosophers.toml`](testdata/found_specs/06_dining_philosophers.toml) - Dining philosophers with deadlock detection
- [`found_specs/07_dekker_mutex.toml`](testdata/found_specs/07_dekker_mutex.toml) - Dekker's mutual exclusion
- [`found_specs/08_bakery_mutex.toml`](testdata/found_specs/08_bakery_mutex.toml) - Lamport's bakery algorithm

**Advanced:**
- [`practical_tla/ch5/`](testdata/practical_tla/ch5/) - Process sets, await statements
- [`practical_tla/ch6/`](testdata/practical_tla/ch6/) - Termination checking

**Run any example:**
```bash
timewinder run testdata/found_specs/04_peterson_mutex.toml
```

---

## Command-Line Options

```bash
# Basic usage
timewinder run spec.toml

# Options
timewinder run spec.toml --debug          # Show detailed execution trace
timewinder run spec.toml --keep-going     # Find all violations, don't stop at first
timewinder run spec.toml --no-deadlocks   # Disable deadlock detection

# Performance profiling
timewinder run spec.toml --cpu-profile=cpu.prof
go tool pprof cpu.prof
```

---

## Learning Resources

### TLA+ and Formal Methods
- **[Learn TLA+](https://learntla.com/)** - Hillel Wayne's excellent interactive tutorial
- **[Practical TLA+ book](https://link.springer.com/book/10.1007/978-1-4842-3829-5)** - By Hillel Wayne, examples included in this repo
- **[TLA+ Video Course](https://lamport.azurewebsites.net/video/videos.html)** - By Leslie Lamport (creator of TLA+)
- **[TLA+ Examples](https://github.com/tlaplus/Examples)** - Official TLA+ example specifications

### Model Checking
- **[Wikipedia: Model Checking](https://en.wikipedia.org/wiki/Model_checking)** - Overview of the technique
- **[The TLA+ Hyperbook](https://lamport.azurewebsites.net/tla/hyperbook.html)** - Deep dive into TLA+ and temporal logic
- **[Principles of Model Checking](https://mitpress.mit.edu/9780262026499/)** - Textbook (advanced)

### Starlark Language
- **[Starlark Language Spec](https://github.com/bazelbuild/starlark/blob/master/spec.md)** - Complete language reference
- **[Starlark Go Implementation](https://github.com/google/starlark-go)** - What Timewinder uses
- **[Bazel (uses Starlark)](https://bazel.build/)** - Build tool that popularized Starlark

### Concurrency
- **[The Little Book of Semaphores](https://greenteapress.com/wp/semaphores/)** - Classic concurrency problems
- **[Concurrent Programming (Maurice Herlihy)](https://www.elsevier.com/books/the-art-of-multiprocessor-programming/herlihy/978-0-12-415950-1)** - Textbook
- **[Concurrency Bugs](https://blog.acolyer.org/2019/05/17/understanding-real-world-concurrency-bugs-in-go/)** - Study of real-world bugs

---

## Architecture

Timewinder has a clean layered architecture:

```
┌─────────────────────────────────────────┐
│  CLI (cmd/timewinder/)                  │
│  - Run command                          │
└────────────┬────────────────────────────┘
             │
┌────────────▼────────────────────────────┐
│  Model Checker (model/)                 │
│  - SingleThreadEngine (BFS)             │
│  - MultiThreadEngine (parallel BFS)     │
│  - Property evaluation                  │
│  - Fairness checking                    │
└────────┬───────────────┬────────────────┘
         │               │
┌────────▼────────┐  ┌──▼───────────────┐
│  VM (vm/)       │  │  Interpreter     │
│  - Compiler     │◄─┤  (interp/)       │
│  - Bytecode     │  │  - Execution     │
│  - 43 opcodes   │  │  - State mgmt    │
└─────────────────┘  └──────────────────┘
```

**Key components:**
- **VM**: Compiles Starlark to bytecode and executes it
- **Interpreter**: Manages state, handles yields, tracks threads
- **Model Checker**: Explores state space with BFS, checks properties
- **CAS (Content-Addressable Storage)**: Deduplicates states by hash

**Multi-threaded checking:**
- Parallel BFS with work-stealing
- Separate exec and check worker pools
- Lock-free state exploration where possible

---

## Contributing

Contributions welcome! Areas that need help:

- **Partial order reduction** - Reduce equivalent interleavings
- **Symmetry reduction** - Detect symmetric states
- **Better error messages** - Make violations easier to understand
- **IDE integration** - LSP server, syntax highlighting
- **More examples** - Port classic TLA+ specs

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

GNU General Public License v3.0 - see [LICENSE](LICENSE) for details

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

---

## Acknowledgments

Timewinder is inspired by:
- **[TLA+](https://lamport.azurewebsites.net/tla/tla.html)** and **[PlusCal](https://lamport.azurewebsites.net/tla/pluscal.html)** by Leslie Lamport
- **[Starlark](https://github.com/google/starlark-go)** by the Bazel team
- **[Practical TLA+](https://link.springer.com/book/10.1007/978-1-4842-3829-5)** by Hillel Wayne

Special thanks to the formal methods community for making these powerful techniques accessible.

---

## FAQ

**Q: Why not just use TLA+ or PlusCal?**
A: TLA+ is incredibly powerful but has a steep learning curve. Timewinder makes model checking accessible to Python programmers who want to verify concurrent algorithms without learning new mathematical notation.

**Q: Can I verify real production code?**
A: Timewinder is for **verifying algorithms and protocols**, not full applications. Model your concurrent logic in Starlark, verify it with Timewinder, then implement it in your production language with confidence.

**Q: How does this compare to fuzzing?**
A: Fuzzing (like Go's race detector) finds bugs through randomized testing. Model checking **exhaustively explores all states** and **proves correctness** (within the model). They're complementary techniques.

**Q: What about performance?**
A: Model checking has exponential complexity (state explosion). Timewinder can handle small to medium concurrent systems (typically <100,000 states). Use fairness and good abstractions to keep state space manageable.

**Q: Can I model distributed systems?**
A: Yes! Model network messages as shared state, use `oneof()` for message reordering, and verify distributed protocols. See the examples for patterns.

---

**Ready to find bugs before they find you?**

```bash
timewinder run your_first_spec.toml
```
