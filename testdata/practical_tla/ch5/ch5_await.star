# Awaiting processes should be easy

queue = []
max_queue_size = oneof(range(1, 20))

def bounded():
  return len(queue) <= max_queue_size

def reader():
  while True:
    until(len(queue) != 0)
    current_msg = queue[0]
    queue = queue[1:]

def writer():
  while True:
    queue.append("msg")
    until(len(queue) < max_queue_size)
