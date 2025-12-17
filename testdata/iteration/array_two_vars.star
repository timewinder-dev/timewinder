# Test array iteration with index and value
items = ["a", "b", "c"]
indices = []
values = []

def test():
  for i, v in items:
    indices = indices + [i]
    values = values + [v]
  return len(indices) == 3 and len(values) == 3

result = test()
