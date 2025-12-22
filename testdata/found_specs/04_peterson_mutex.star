# Peterson's Algorithm for Mutual Exclusion in Timewinder
# Converted from PlusCal example
# Source: https://lamport.azurewebsites.net/tla/peterson.html
# Original: Classic two-process mutual exclusion algorithm using flags and turn variable

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

        step("exit_critical")
        in_critical[pid] = False

        # Exit protocol
        step("clear_flag")
        flag[pid] = False
