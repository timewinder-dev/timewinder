# Test dict iteration with single variable (keys only)
acc = {"alice": 10, "bob": 20, "charlie": 15}
names = []

def test():
  for name in acc:
    names = names + [name]
  # Keys should be sorted alphabetically
  return names == ["alice", "bob", "charlie"]

result = test()
