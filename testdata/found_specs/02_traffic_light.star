# Traffic Light Controller in Timewinder
# Converted from PlusCal example
# Source: https://writinglinesofcode.com/blog/pluscal-part-01
# Original: Models a simple traffic light that cycles through red, yellow, and green states

light = {"red": 1, "yellow": 0, "green": 0}

def traffic_light():
    while True:
      # Turn Green
      fstep("turn_green")
      if light["red"] == 1 and light["yellow"] == 0 and light["green"] == 0:
          light["red"] = 0
          light["green"] = 1

      # Turn Yellow
      fstep("turn_yellow")
      if light["red"] == 0 and light["yellow"] == 0 and light["green"] == 1:
          light["green"] = 0
          light["yellow"] = 1

      # Turn Red
      fstep("turn_red")
      if light["red"] == 0 and light["yellow"] == 1 and light["green"] == 0:
          light["red"] = 1
          light["yellow"] = 0
