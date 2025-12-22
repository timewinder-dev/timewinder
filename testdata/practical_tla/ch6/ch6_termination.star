# Traffic light termination example
# A car waits at a red light. The light cycles between red and green.
# Once the light turns green, the car drives away and the light stops cycling.

at_light = True
light = "red"

def next_color(c):
    if c == "red":
        return "green"
    else:
        return "red"

def light_process():
    while at_light:
        step("Cycle")
        light = next_color(light)

def car_process():
    step("Drive")
    # Wait for green light (equivalent to PlusCal's "when")
    until(light == "green")
    at_light = False
