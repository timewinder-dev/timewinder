# Test calling user-defined functions

counter = 0

def increment():
  counter = counter + 1

def test():
  increment()
  increment()
