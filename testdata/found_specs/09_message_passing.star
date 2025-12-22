# Simple Message Passing System in Timewinder
# Models asynchronous communication between sender and receiver
# Common pattern in distributed systems and concurrent programming

channel = []
received = 0

def sender():
    step("send_hello")
    channel.append("hello")

    step("send_world")
    channel.append("world")

def receiver():
    while received < 2:
        step("try_receive")

        if len(channel) > 0:
            channel.pop(0)
            received = received + 1
