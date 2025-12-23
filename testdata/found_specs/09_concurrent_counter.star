# Concurrent Counter with Race Condition
# Source: https://learntla.com/core/concurrency.html
# Demonstrates classic race condition when incrementing a shared counter
# Two threads read counter, increment locally, then write back
# Without proper synchronization, increments can be lost

counter = {"value": 0}  # Shared counter (using dict to allow mutation)

def incrementer(tid):
    """Thread that increments the counter - with a race condition!"""
    for iteration in range(2):  # Each thread increments twice
        # Read counter (non-atomic)
        step("read_counter")
        tmp = counter["value"]

        # Increment local copy (non-atomic)
        step("increment_local")
        tmp = tmp + 1

        # Write back (non-atomic)
        step("write_counter")
        counter["value"] = tmp
