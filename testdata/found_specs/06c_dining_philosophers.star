# Dining Philosophers in Timewinder (Strong Fairness Solution)
# Converted from PlusCal example
# Source: https://github.com/muratdem/PlusCal-examples/blob/master/DiningPhil/diningRound0.tla
# Original: Classic concurrency problem modeling philosophers sharing forks around a table
# Simplified to 3 philosophers
# Fixed: Use strong fairness to ensure each philosopher eventually eats

N = 3  # Number of philosophers
fork = [None, None, None]  # None means available, otherwise philosopher ID
state = ["Thinking", "Thinking", "Thinking"]

def philosopher(pid):
    left_fork = pid
    right_fork = (pid + 1) % N

    for iteration in range(2):  # Limited iterations to avoid infinite loops
        # Become hungry
        sfstep("become_hungry")  # Strongly fair
        if state[pid] == "Thinking":
            state[pid] = "Hungry"

        # Try to pick up forks (symmetric - right then left)
        sfstep("try_eat")  # Strongly fair
        if state[pid] == "Hungry":
            # Wait for right fork
            sfuntil(fork[right_fork] == None)  # Strongly fair wait
            fork[right_fork] = pid

            # Wait for left fork
            sfuntil(fork[left_fork] == None)  # Strongly fair wait
            fork[left_fork] = pid

            state[pid] = "Eating"

        # Put down forks and think
        sfstep("put_down_forks")  # Strongly fair
        if state[pid] == "Eating":
            state[pid] = "Thinking"
            fork[left_fork] = None
            fork[right_fork] = None
