# Peterson's Algorithm for Mutual Exclusion in Timewinder
# Converted from PlusCal example
# Source: https://lamport.azurewebsites.net/tla/peterson.html
# Original: Classic two-process mutual exclusion algorithm using flags and turn variable
#
# KEY INSIGHT: Peterson's algorithm (1981) simplified Dekker's algorithm (1965) by using
# PROACTIVE YIELDING. Instead of detecting conflicts and backing off, each process
# immediately sets "turn = other" to yield priority. This single insight eliminates
# all the complex backoff logic needed in Dekker's algorithm.
#
# The genius: By giving away priority immediately, each process says "I want in, but
# YOU go first if there's a conflict." This makes the algorithm much simpler to
# understand and prove correct than Dekker's reactive backoff approach.

flag = [False, False]
turn = 0
in_critical = [False, False]

def process(pid):
    other = 1 - pid

    # Loop a few times
    for iteration in range(3):
        # Entry protocol
        step("set_flag")
        flag[pid] = True

        step("set_turn")
        turn = other

        # Wait until we can enter
        step("await_entry")
        until((not flag[other]) or (turn == pid))

        # Critical section
        step("enter_critical")
        in_critical[pid] = True

        sfstep("exit_critical")
        in_critical[pid] = False

        # Exit protocol
        step("clear_flag")
        flag[pid] = False
