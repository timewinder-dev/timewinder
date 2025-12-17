# Test array iteration with single variable
items = [1, 2, 3, 4, 5]
sum = 0

def test():
  for x in items:
    sum = sum + x
  return sum

result = test()
