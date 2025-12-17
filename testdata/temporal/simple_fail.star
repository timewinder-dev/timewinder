# Simple EventuallyAlways failure test
# Property starts true but becomes false

flag = True

def is_true():
    return flag == True

def main():
    step("set_false")
    flag = False
