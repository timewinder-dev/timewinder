# Simple EventuallyAlways success test
# Property starts false and becomes true (single step)

flag = False

def is_true():
    return flag == True

def main():
    step("set_flag")
    flag = True
