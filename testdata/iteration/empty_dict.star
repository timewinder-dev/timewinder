# Test empty dict iteration
acc = {}
executed = False

def test():
  for k, v in acc:
    executed = True  # Should never execute
  return executed == False

result = test()
