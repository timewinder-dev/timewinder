# This is going to deadlock, and should hit the deadlock detector

queue = []
max_queue_size = 3

def bounded():
  return len(queue) <= max_queue_size

def add_to_queue(val):
  until(len(queue) < max_queue_size)
  queue = append(queue, val)

def reader():
  while True:
    until(len(queue) != 0)
    current_msg = queue[0]
    queue = queue[1:]
    either = oneof(["skip", "notify"])
    if either == "notify":
      step("NotifyFailure")
      current_msg = "none"
      add_to_queue("fail")

def writer():
  while True:
    add_to_queue("msg")
