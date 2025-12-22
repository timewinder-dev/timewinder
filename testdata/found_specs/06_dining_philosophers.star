# Dining Philosophers in Timewinder
# Converted from PlusCal example
# Source: https://github.com/muratdem/PlusCal-examples/blob/master/DiningPhil/diningRound0.tla
# Original: Classic concurrency problem modeling philosophers sharing forks around a table
# Simplified to 3 philosophers

N = 3  # Number of philosophers
fork = [None, None, None]  # None means available, otherwise philosopher ID
state = ["Thinking", "Thinking", "Thinking"]

def philosopher(pid):
    left_fork = pid
    right_fork = (pid - 1) % N

    for iteration in range(2):  # Limited iterations to avoid infinite loops
        # Become hungry
        step("become_hungry")
        if state[pid] == "Thinking":
            state[pid] = "Hungry"

        # Try to pick up forks
        step("try_eat")
        if state[pid] == "Hungry":
            # Wait for right fork
            until(fork[right_fork] == None)
            fork[right_fork] = pid

            # Wait for left fork
            until(fork[left_fork] == None)
            fork[left_fork] = pid

            state[pid] = "Eating"

        # Put down forks and think
        step("put_down_forks")
        if state[pid] == "Eating":
            state[pid] = "Thinking"
            fork[left_fork] = None
            fork[right_fork] = None
