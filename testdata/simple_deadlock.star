# Simple deadlock test - two threads waiting for a condition that never occurs

l = ["foo"]

def waiter_a():
  until(len(l) == 0)
  step("WaiterA_Done")

def waiter_b():
  until(len(l) == 0)
  step("WaiterB_Done")
