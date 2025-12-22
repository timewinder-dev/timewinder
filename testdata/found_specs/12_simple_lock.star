# Simple Lock/Mutex Implementation in Timewinder
# Basic mutual exclusion using a single boolean lock variable
# Demonstrates the simplest form of synchronization
# Source: Fundamental concurrency pattern

lock = False
critical_count = 0
in_critical = [False, False]

def process(pid):
    for iteration in range(3):
        # Acquire lock
        step("acquire")
        until(not lock)
        lock = True

        # Enter critical section
        step("critical")
        in_critical[pid] = True
        critical_count = critical_count + 1

        # Leave critical section
        step("leave_critical")
        in_critical[pid] = False

        # Release lock
        step("release")
        lock = False
