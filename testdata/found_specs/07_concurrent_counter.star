# Concurrent Counter in Timewinder
# Simple example showing race conditions with concurrent increments
# Common teaching example in concurrency courses
# Demonstrates lost update problem when reads and writes are separated

counter = 0

def incrementer():
    for i in range(5):
        # Read counter
        step("read_counter")
        temp = counter

        # Write counter (race condition possible here!)
        step("write_counter")
        counter = temp + 1
