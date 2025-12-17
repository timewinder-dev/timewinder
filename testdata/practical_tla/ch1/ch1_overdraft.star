people = ["alice", "bob"]
acc = {"alice": 5, "bob": 5}
sender = "alice"
receiver = "bob"
amount = oneof(range(8))

def no_overdrafts():
  for name, amt in acc:
    if amt < 0:
      return False
  return True

def algorithm():
  step("Withdraw")
  acc[sender] -= amount
  step("Deposit")
  acc[receiver] += amount
