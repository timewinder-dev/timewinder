"""
Example demonstrating thread replicas with symmetry sets.
This creates 3 identical worker threads that increment a shared counter.
"""

counter = {"value": 0}

def final_count():
    """Check that counter equals expected value after all workers finish"""
    return counter["value"] == 3

def worker():
    """Each worker increments the counter once"""
    fstep("read")
    val = counter["value"]
    fstep("write")
    counter["value"] = val + 1
