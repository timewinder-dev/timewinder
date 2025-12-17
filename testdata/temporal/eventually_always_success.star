# Test EventuallyAlways - Success (Terminating)
# Counter increases and eventually reaches target, then stays >= target

counter = 0
target = 3

def reaches_target():
    return counter >= target

def increment_counter():
    step("increment")
    counter = counter + 1

def main():
    # Keep incrementing while counter < 5
    increment_counter()
    if counter < 5:
        increment_counter()
    if counter < 5:
        increment_counter()
    if counter < 5:
        increment_counter()
    if counter < 5:
        increment_counter()
    # After this, counter will be 5, which is >= target (3)
