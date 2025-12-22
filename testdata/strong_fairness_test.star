# Test strong fairness
counter = {"value": 0}

def incrementer():
    sfstep("increment")  # Strongly fair step
    counter["value"] = counter["value"] + 1

def observer():
    step("observe")  # Normal step
    # Just observe, don't modify
