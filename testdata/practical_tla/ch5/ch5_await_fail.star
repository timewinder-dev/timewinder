# Awaiting processes should be easy

queue = []
max_queue_size = oneof(range(1, 20))

def bounded():
  return len(queue) <= max_queue_size

def reader():
  global_var("queue")
  while True:
    until(len(queue) != 0)
    current_msg = queue[0]
    queue = queue[1:]

def writer():
  global_var("queue")
  while True:
    step("Send")
    queue.append("msg")
