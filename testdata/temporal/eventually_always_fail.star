# Test EventuallyAlways - Failure (Never Permanently True)
# Counter increases beyond a limit - property is true initially but becomes false

counter = 0

def always_less_than_ten():
    return counter < 10

def increment_counter():
    step("increment")
    counter = counter + 1

def main():
    # Increment counter 15 times - it will exceed 10 and violate EventuallyAlways
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    increment_counter()
    # After this, counter = 15, which is not < 10
