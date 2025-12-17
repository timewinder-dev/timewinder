# Each process independently transfers money from alice to bob

people = ["alice", "bob"]
acc = {"alice": 5, "bob": 5}
sender = "alice"
receiver = "bob"

def no_overdrafts():
  for name, amt in acc:
    if amt < 0:
      return False
  return True

def main():
  amount = oneof(range(1, acc[sender] -1))
  if amount == None:
    return
  step("CheckFunds")
  if amount <= acc[sender]:
    acc[sender] -= amount
    step("Deposit")
    acc[receiver] += amount
