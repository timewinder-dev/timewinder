# Each process independently transfers money from alice to bob

people = ["alice", "bob"]
acc = {"alice": 5, "bob": 5}
sender = "alice"
receiver = "bob"
amount = oneof(range(1, acc[sender]))

def no_overdrafts():
  for name, amt in acc:
    if amt < 0:
      return False
  return True

def main():
  step("Withdraw")
  acc[sender] -= amount
  step("Deposit")
  acc[receiver] += amount
