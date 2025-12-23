# Dining Philosophers in Timewinder (Resource Ordering Solution)
# Converted from PlusCal example
# Source: https://github.com/muratdem/PlusCal-examples/blob/master/DiningPhil/diningRound0.tla
# Original: Classic concurrency problem modeling philosophers sharing forks around a table
# Simplified to 3 philosophers
# Fixed: Use resource ordering - always pick up lower-numbered fork first

N = 3  # Number of philosophers
fork = [None, None, None]  # None means available, otherwise philosopher ID
state = ["Thinking", "Thinking", "Thinking"]

def philosopher(pid):
    left_fork = pid
    right_fork = (pid + 1) % N

    # Resource ordering: always pick up lower-numbered fork first
    if left_fork < right_fork:
        first_fork = left_fork
        second_fork = right_fork
    else:
        first_fork = right_fork
        second_fork = left_fork

    for iteration in range(2):  # Limited iterations to avoid infinite loops
        # Become hungry
        step("become_hungry")
        if state[pid] == "Thinking":
            state[pid] = "Hungry"

        # Try to pick up forks (in order: lower-numbered first)
        step("try_eat")
        if state[pid] == "Hungry":
            # Wait for first fork (lower-numbered)
            until(fork[first_fork] == None)
            fork[first_fork] = pid

            # Wait for second fork (higher-numbered)
            until(fork[second_fork] == None)
            fork[second_fork] = pid

            state[pid] = "Eating"

        # Put down forks and think
        step("put_down_forks")
        if state[pid] == "Eating":
            state[pid] = "Thinking"
            fork[left_fork] = None
            fork[right_fork] = None
