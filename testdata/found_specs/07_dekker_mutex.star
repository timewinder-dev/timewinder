# Dekker's Algorithm for Mutual Exclusion
# Source: https://www.geeksforgeeks.org/dsa/dekkers-algorithm-in-process-synchronization/
# Classic 2-process mutual exclusion algorithm (predates Peterson's algorithm, 1965)
# Uses flags to indicate intent and turn to break ties with REACTIVE BACKOFF
#
# KEY DIFFERENCE FROM PETERSON: Dekker does NOT proactively yield (no "turn = other").
# Instead, it detects conflicts and backs off when it's not our turn. More complex
# but historically important as the first correct software solution to mutual exclusion.
#
# Guarantees: mutual exclusion, deadlock-freedom, starvation-freedom

# Shared variables
flag = [False, False]  # flag[i] = True means process i wants to enter critical section
turn = 0               # Tie-breaker: whose turn is it?
in_critical = [False, False]  # Track who's in critical section (for verification)

def process(pid):
    other = 1 - pid  # The other process (0 or 1)

    for iteration in range(2):  # Run twice to demonstrate fairness
        # Entry protocol: Raise flag to indicate intent
        step("set_flag")
        flag[pid] = True

        # Dekker's waiting loop: keep checking while other wants to enter
        # This is the KEY structure that makes Dekker work
        step("wait_loop")
        while flag[other]:
            # If it's the other's turn, back off (reactive backoff)
            if turn == other:
                flag[pid] = False

                step("wait_turn")
                wait(turn == pid)

                # Re-raise flag after getting turn
                flag[pid] = True

            # Check again (loop continues to re-check flag[other])
            step("recheck")

        # Enter critical section (mutual exclusion guaranteed)
        # We only reach here when flag[other] is False
        step("enter_critical")
        in_critical[pid] = True

        # Critical section work happens here
        step("in_critical")
        # (nothing to do, just verifying mutual exclusion)

        # Exit protocol
        step("exit_critical")
        in_critical[pid] = False
        turn = other  # Give turn to other process
        flag[pid] = False
