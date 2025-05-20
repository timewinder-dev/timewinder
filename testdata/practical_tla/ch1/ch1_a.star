people = ["alice", "bob"]
acc = {"alice": 5, "bob": 5}
sender = "alice"
receiver = "bob"
amount = 3

def no_overdrafts():
  for name in acc:
    if acc[name] < 0:
      return False 
  return True

def algorithm():
  yield("Withdraw")
  acc[sender] -= amount
  yield("Deposit") 
  acc[receiver] += amount
