\* Bank Transfer with Overdraft Protection
\* Source: https://www.hillelwayne.com/post/list-of-tla-examples/
\*         https://learntla.com (Hillel Wayne's Learn TLA+ tutorial)
\* Models concurrent withdrawals from a bank account with overdraft checking
\* Demonstrates importance of atomic operations in financial transactions

--algorithm BankTransfer {
  variables account = 100;

  process (withdraw \in 1..2) {
    w1:
      with (amount \in 10..50) {
        when account >= amount;
        account := account - amount;
      }
  }
}
