people = ["alice", "bob"]
acc = {"alice": 5, "bob": 5}
sender = "alice"
receiver = "bob"
amount = 8

def no_overdrafts():
  # Simple check without loop (since iterators not implemented yet)
  if acc["alice"] < 0:
    return False
  if acc["bob"] < 0:
    return False
  return True

def algorithm():
  step("Withdraw")
  acc[sender] -= amount
  step("Deposit")
  acc[receiver] += amount
