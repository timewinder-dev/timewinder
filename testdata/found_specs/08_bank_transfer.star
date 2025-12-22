# Bank Transfer with Overdraft Protection in Timewinder
# Based on examples from Hillel Wayne's Learn TLA+ tutorial
# Source: https://www.hillelwayne.com/post/list-of-tla-examples/
#         https://learntla.com
# Original: Models concurrent withdrawals from a bank account with overdraft checking
# Demonstrates importance of atomic operations in financial transactions

account = 100

def withdraw(amount):
    # Check balance (separate step - race condition possible!)
    step("check_balance")
    if account >= amount:
        # Withdraw (another thread could withdraw here first!)
        step("deduct_amount")
        account = account - amount
