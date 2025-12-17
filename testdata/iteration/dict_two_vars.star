# Test dict iteration with key and value
acc = {"alice": 10, "bob": 20, "charlie": 15}
total = 0
count = 0

def test():
  for name, amount in acc:
    total = total + amount
    count = count + 1
  return total == 45 and count == 3

result = test()
