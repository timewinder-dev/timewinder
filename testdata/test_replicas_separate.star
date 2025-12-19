"""
Example with 3 separate (non-symmetric) worker threads.
This should explore MORE states than the replicas version
because the model checker cannot use symmetry reduction.
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
