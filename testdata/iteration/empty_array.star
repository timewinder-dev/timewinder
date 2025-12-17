# Test empty array iteration
items = []
executed = False

def test():
  for x in items:
    executed = True  # Should never execute
  return executed == False

result = test()
