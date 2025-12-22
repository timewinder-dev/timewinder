# Duplication Checker in Timewinder
# Converted from PlusCal example
# Source: https://learntla.com/core/pluscal.html
# Original: Checks if a sequence contains duplicate elements using set operations

seq = [1, 2, 3, 2]
index = 0
seen = []
is_unique = True

def check_duplicates():
    global index, is_unique

    while index < len(seq):
        step("check_element")

        if seq[index] not in seen:
            seen.append(seq[index])
        else:
            is_unique = False

        index = index + 1
