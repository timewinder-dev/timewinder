# Simple infinite loop test
# Don't modify any state, so we revisit the same state and detect the cycle

def infinite_loop():
    while True:
        step("Loop")
        # Do nothing - just loop forever
