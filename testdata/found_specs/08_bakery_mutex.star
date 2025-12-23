# Lamport's Bakery Algorithm for Mutual Exclusion
# Source: https://www.geeksforgeeks.org/dsa/bakery-algorithm-in-process-synchronization/
# N-process mutual exclusion using ticket numbers (like a bakery queue)
# Processes take tickets and wait for their turn in lexicographical order
# Guarantees: mutual exclusion, fairness, starvation-freedom

N = 3  # Number of processes

# Shared variables
choosing = [False, False, False]  # choosing[i] = True means process i is picking a ticket
number = [0, 0, 0]                # number[i] = process i's ticket number (0 = not waiting)
in_critical = [False, False, False]  # Track who's in critical section (for verification)

def max_number():
    """Helper to find maximum ticket number"""
    max_num = 0
    for i in range(N):
        if number[i] > max_num:
            max_num = number[i]
    return max_num

def process(pid):
    for iteration in range(2):  # Run twice to demonstrate fairness
        # Entry protocol: Get a ticket
        step("start_choosing")
        choosing[pid] = True

        step("take_ticket")
        # Take a ticket with number higher than all others
        number[pid] = max_number() + 1

        step("done_choosing")
        choosing[pid] = False

        # Wait for turn: check all other processes
        for j in range(N):
            if j == pid:
                step("skip_self")
                continue  # Skip self

            # Keep checking while process j is choosing
            # Use while loop because choosing status can change
            step("check_choosing")
            while choosing[j]:
                step("wait_choosing")

            # Keep checking while process j has priority over us
            # j has priority if:
            #   - j has a ticket (number[j] != 0) AND
            #   - j's ticket is lower than ours OR
            #   - tickets are equal but j's pid is lower
            step("check_priority")
            while number[j] != 0 and (number[j] < number[pid] or (number[j] == number[pid] and j < pid)):
                step("wait_priority")

        # Enter critical section
        step("enter_critical")
        in_critical[pid] = True

        # Critical section work
        step("in_critical")
        # (nothing to do, just verifying mutual exclusion)

        # Exit protocol
        step("exit_critical")
        in_critical[pid] = False
        number[pid] = 0  # Release ticket
