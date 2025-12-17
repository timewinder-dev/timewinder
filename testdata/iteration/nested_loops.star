# Test nested loop iteration
outer = [1, 2, 3]
inner = [10, 20]
total = 0

def test():
  for x in outer:
    for y in inner:
      total = total + (x * y)
  # (1*10 + 1*20) + (2*10 + 2*20) + (3*10 + 3*20)
  # = 30 + 60 + 90 = 180
  return total == 180

result = test()
