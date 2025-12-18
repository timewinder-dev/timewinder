# Test that should complete without deadlock - threads finish normally

counter = 0

def incrementer():
  step("Increment")
  counter = counter + 1
  step("Done")
