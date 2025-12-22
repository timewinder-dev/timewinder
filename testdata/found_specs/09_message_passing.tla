\* Simple Message Passing System
\* Models asynchronous communication between sender and receiver
\* Common pattern in distributed systems and concurrent programming

--algorithm MessagePassing {
  variables
    channel = <<>>,
    received = 0;

  process (sender = "s") {
    send:
      channel := Append(channel, "hello");
      channel := Append(channel, "world");
  }

  process (receiver = "r") {
    loop:
      while (received < 2) {
        recv:
          await Len(channel) > 0;
          channel := Tail(channel);
          received := received + 1;
      }
  }
}
