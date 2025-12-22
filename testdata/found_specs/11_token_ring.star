# Token Ring Protocol in Timewinder
# Models a simple token passing protocol for mutual exclusion
# Only the process holding the token can enter its critical section
# Source: Classic distributed algorithm pattern

token = 0  # Process ID that currently has the token
in_cs = [False, False, False]

def process(pid):
    for iteration in range(2):  # Limited iterations
        # Wait for token
        step("wait_token")
        until(token == pid)

        # Enter critical section
        step("enter_cs")
        in_cs[pid] = True

        # Exit critical section
        step("exit_cs")
        in_cs[pid] = False

        # Pass token to next process
        step("pass_token")
        token = (pid + 1) % 3
