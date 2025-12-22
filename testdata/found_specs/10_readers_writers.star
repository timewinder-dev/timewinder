# Readers-Writers Problem in Timewinder
# Classic synchronization problem where multiple readers can read simultaneously
# but writers need exclusive access
# Source: Classic OS/concurrency textbook problem

readers = 0
writing = False
data = 0

def reader():
    # Enter read section
    step("enter_read")
    until(not writing)
    readers = readers + 1

    # Read data
    step("reading")
    _ = data  # Simulate reading

    # Exit read section
    step("exit_read")
    readers = readers - 1

def writer():
    # Enter write section
    step("enter_write")
    until(not writing and readers == 0)
    writing = True

    # Write data
    step("writing")
    data = data + 1

    # Exit write section
    step("exit_write")
    writing = False
