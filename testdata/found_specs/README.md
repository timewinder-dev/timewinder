# Found TLA+/PlusCal Specifications

This directory contains TLA+/PlusCal specifications found from blogs, tutorials, and educational resources, converted to Timewinder format for testing and demonstration purposes.

**Note**: Not all of these specifications are expected to work perfectly with Timewinder. They serve as a test suite and showcase of different concurrency patterns.

## Examples

### 1. Hello World
**Files**: `01_hello_world.*`
**Source**: [Writing Lines Of Code - PlusCal Tutorial](https://writinglinesofcode.com/blog/pluscal-part-01)
**Description**: Minimal PlusCal algorithm demonstrating basic structure
**Pattern**: Simple sequential execution

### 2. Traffic Light
**Files**: `02_traffic_light.*`
**Source**: [Writing Lines Of Code - PlusCal Tutorial](https://writinglinesofcode.com/blog/pluscal-part-01)
**Description**: Simple state machine cycling through red, yellow, and green states
**Pattern**: State machine, sequential transitions

### 3. Duplicate Checker
**Files**: `03_duplicate_checker.*`
**Source**: [Learn TLA+ - PlusCal Tutorial](https://learntla.com/core/pluscal.html)
**Description**: Checks if a sequence contains duplicate elements using set operations
**Pattern**: Iteration, set membership, sequential algorithm

### 4. Peterson's Algorithm
**Files**: `04_peterson_mutex.*`
**Source**: [Leslie Lamport - Peterson's Algorithm](https://lamport.azurewebsites.net/tla/peterson.html)
**Description**: Classic two-process mutual exclusion algorithm
**Pattern**: Mutual exclusion, flags and turn variable, busy waiting

### 5. Bounded Buffer (Producer-Consumer)
**Files**: `05_bounded_buffer.*`
**Source**: [GitHub - belaban/pluscal](https://github.com/belaban/pluscal/blob/master/BoundedBuffer.tla)
**Description**: Producers and consumers sharing a bounded queue
**Pattern**: Producer-consumer, bounded buffer, synchronization

### 6. Dining Philosophers
**Files**: `06_dining_philosophers.*`
**Source**: [GitHub - muratdem/PlusCal-examples](https://github.com/muratdem/PlusCal-examples/blob/master/DiningPhil/diningRound0.tla)
**Description**: Classic concurrency problem with philosophers sharing forks
**Pattern**: Resource contention, deadlock potential, circular wait

### 7. Concurrent Counter
**Files**: `07_concurrent_counter.*`
**Source**: Common concurrency teaching example
**Description**: Demonstrates race conditions with concurrent increments
**Pattern**: Lost update problem, race condition, read-modify-write

## File Structure

For each example, you'll find three files:
- `.tla` - Original PlusCal/TLA+ specification
- `.star` - Timewinder (Starlark) conversion
- `.toml` - Timewinder specification file (threads and properties)

## Key Learning Resources

- [Learn TLA+](https://learntla.com) - Comprehensive free tutorial by Hillel Wayne
- [Hillel Wayne's Blog](https://www.hillelwayne.com) - List of TLA+ examples and explanations
- [Leslie Lamport's TLA+ Site](https://lamport.azurewebsites.net/tla/tla.html) - Official TLA+ resources
- [PlusCal Tutorial](https://lamport.azurewebsites.net/tla/tutorial/intro.html) - Official PlusCal tutorial
- [GitHub: tlaplus/Examples](https://github.com/tlaplus/Examples) - Official TLA+ example repository

## Conversion Notes

When converting from PlusCal to Timewinder:

1. **step()** replaces label boundaries - marks atomic operation boundaries
2. **until()** replaces await - busy-wait condition checking
3. **Global-first scoping** - Timewinder uses global-first variable lookup. Variables are resolved in global scope first, then local scope. No `global` keyword is needed or supported.
4. **Loops** - Often bounded in Timewinder to ensure termination
5. **Sets** - Converted to Python lists (sets not directly supported)
6. **Sequences** - Converted to Python lists
7. **Process parameters** - Converted to function parameters
8. **Fairness** - Expressed via `fair = true` in TOML spec files

### Scoping Rules

Timewinder uses **global-first scoping**:
- When you assign to a variable, Timewinder first checks if it exists in the global scope
- If it exists globally, the global variable is modified
- If not, a new local variable is created
- This means you can modify global variables directly from functions without any `global` keyword
- Example:
  ```python
  counter = 0

  def increment():
      counter = counter + 1  # Modifies the global counter
  ```

## Running Examples

```bash
./timewinder run testdata/found_specs/01_hello_world.toml
./timewinder run testdata/found_specs/04_peterson_mutex.toml
# ... etc
```

## Sources Summary

Main sources for these specifications:
- [Hillel Wayne - Learn TLA+](https://learntla.com/core/pluscal.html)
- [Hillel Wayne - List of TLA+ Examples](https://www.hillelwayne.com/post/list-of-tla-examples/)
- [Leslie Lamport - Peterson's Algorithm](https://lamport.azurewebsights.net/tla/peterson.html)
- [GitHub - belaban/pluscal](https://github.com/belaban/pluscal)
- [GitHub - muratdem/PlusCal-examples](https://github.com/muratdem/PlusCal-examples)
- [Writing Lines Of Code - PlusCal Tutorial](https://writinglinesofcode.com/blog/pluscal-part-01)
