# Test file for new wait() and step_until() primitives
# Demonstrates both wait() and step_until() in a simple turn-taking scenario

turn = 0
done = [False, False]

def thread_with_wait(tid):
    """Demonstrates wait() - waits once for condition"""
    # Wait for our turn using wait() (formerly until())
    step("wait_turn")
    wait(turn == tid)

    # Do work
    step("work")
    done[tid] = True

    # Pass turn to next thread
    step("pass_turn")
    turn = (tid + 1) % 2

def thread_with_step_until(tid):
    """Demonstrates step_until() - loops while condition is true"""
    # Wait for our turn using step_until() - loops while NOT our turn
    step_until("wait_turn", turn != tid)

    # Do work
    step("work")
    done[tid] = True

    # Pass turn to next thread
    step("pass_turn")
    turn = (tid + 1) % 2
